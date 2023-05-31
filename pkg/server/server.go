/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"reflect"
	"strconv"
	"strings"

	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/google/go-cmp/cmp"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rotisserie/eris"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/serverpb"
	"github.com/TencentBlueKing/blueking-apigateway-operator/internal/tracer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// Server ...
type Server struct {
	serverpb.UnimplementedResourcesServer
	LeaderElector   leaderelection.LeaderElector
	registry        registry.Registry
	commiter        *commiter.Commiter
	apisixConfStore synchronizer.ApisixConfigStore

	mux *http.ServeMux

	logger *zap.SugaredLogger
}

// 整合了http与 grpc的请求处理
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

// NewServer ...
func NewServer(
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	apisixConfStore synchronizer.ApisixConfigStore,
	commiter *commiter.Commiter,
) *Server {
	return &Server{
		LeaderElector:   leaderElector,
		registry:        registry,
		apisixConfStore: apisixConfStore,
		commiter:        commiter,

		mux: http.NewServeMux(),

		logger: logging.GetLogger().Named("server"),
	}
}

// RegisterMetric ...
func (s *Server) RegisterMetric(gatherer prometheus.Gatherer) {
	handler := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	})

	s.mux.Handle("/metrics", handler)
}

// Run ...
func (s *Server) Run(ctx context.Context, config *config.Config) error {
	if config.Debug {
		s.mux.HandleFunc("/debug/", pprof.Index)
	}

	// 注册grpc相关的方法
	rpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))),
	)
	serverpb.RegisterResourcesServer(rpcServer, s)

	// 用grpcapigw的方式注册路由, 使得grpc的服务可以被http的服务访问
	gwmux := runtime.NewServeMux()
	s.mux.Handle("/", gwmux)

	// run http server
	var addr, addrv6 string
	if config.HttpServer.BindAddressV6 != "" {
		addrv6 = config.HttpServer.BindAddressV6 + ":" + strconv.Itoa(
			config.HttpServer.BindPort,
		)
		go MustServeHTTP(ctx, addrv6, "tcp6", grpcHandlerFunc(rpcServer, s.mux))
	}

	if config.HttpServer.BindAddress != "" {
		addr = config.HttpServer.BindAddress + ":" + strconv.Itoa(
			config.HttpServer.BindPort,
		)
		go MustServeHTTP(ctx, addr, "tcp4", grpcHandlerFunc(rpcServer, s.mux))
	}

	grpcEndpoint := ""
	if len(addr) != 0 {
		grpcEndpoint = addr
	} else {
		grpcEndpoint = addrv6
	}

	// register grpc gw http handler to mux with grpc client
	dopts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := serverpb.RegisterResourcesGwFromEndpoint(ctx, gwmux, grpcEndpoint, dopts)
	return err
}

// Sync ...
func (s *Server) Sync(ctx context.Context, req *serverpb.SyncRequest) (*serverpb.SyncResponse, error) {
	outgoingCtx, span := tracer.NewTracer("grpcServer").Start(ctx, "grpcServer/sync", trace.WithAttributes(
		attribute.Bool("all", req.All),
		attribute.String("gateway", req.Gateway),
		attribute.String("stage", req.Stage),
	))
	defer span.End()
	resp := &serverpb.SyncResponse{}
	if !req.All {
		s.commiter.ForceCommit(outgoingCtx, []registry.StageInfo{
			{
				GatewayName: req.Gateway,
				StageName:   req.Stage,
			},
		})
		return resp, nil
	}
	stageList, err := s.registry.ListStages(outgoingCtx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	s.commiter.ForceCommit(outgoingCtx, stageList)
	return resp, nil
}

// Diff ...
func (s *Server) Diff(ctx context.Context, req *serverpb.DiffRequest) (*serverpb.DiffResponse, error) {
	outgoingCtx, span := tracer.NewTracer("grpcServer").Start(ctx, "grpcServer/diff", trace.WithAttributes(
		attribute.Bool("all", req.All),
		attribute.String("gateway", req.Gateway),
		attribute.String("stage", req.Stage),
		attribute.String("resource", req.Resource.String()),
	))
	defer span.End()
	resp := &serverpb.DiffResponse{}
	if !req.All {
		si := registry.StageInfo{
			GatewayName: req.Gateway,
			StageName:   req.Stage,
		}
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		originalApisixResources := s.apisixConfStore.Get(stageKey)
		apisixResources, err := s.commiter.ConvertEtcdKVToApisixConfiguration(outgoingCtx, si)
		if err != nil {
			// TODO
			span.RecordError(err)
			return nil, err
		}
		resourceKey, err := s.getRouteIDByResourceIdentity(
			originalApisixResources,
			req.Gateway,
			req.Stage,
			req.Resource,
		)
		if err != nil {
			resourceKey, _ = s.getRouteIDByResourceIdentity(apisixResources, req.Gateway, req.Stage, req.Resource)
		}
		resp.Data = map[string]*serverpb.StageScopedApisixResources{
			stageKey: s.diffWithRouteID(outgoingCtx, originalApisixResources, apisixResources, resourceKey),
		}
		return resp, nil
	}
	stageList, err := s.registry.ListStages(outgoingCtx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	resp.Data = make(map[string]*serverpb.StageScopedApisixResources, len(stageList))
	allApisixResources := s.apisixConfStore.GetAll()
	for _, stage := range stageList {
		stageKey := config.GenStagePrimaryKey(stage.GatewayName, stage.StageName)
		apisixResources, err := s.commiter.ConvertEtcdKVToApisixConfiguration(outgoingCtx, stage)
		if err != nil {
			resp.Message = fmt.Sprintf("%s [stage %s failed: %s]", resp.Message, stageKey, eris.ToString(err, true))
			span.RecordError(err, trace.WithAttributes(
				attribute.String("stageKey", stageKey),
			))
			continue
		}
		diffResult := s.diff(outgoingCtx, allApisixResources[stageKey], apisixResources)
		if !s.isDiffResultEmpty(diffResult) {
			resp.Data[stageKey] = diffResult
		}
	}
	return resp, nil
}

func (s *Server) diffWithRouteID(
	ctx context.Context,
	lhs, rhs *apisix.ApisixConfiguration,
	routeID string,
) *serverpb.StageScopedApisixResources {
	ret := &serverpb.StageScopedApisixResources{}
	ret.Routes, _ = structpb.NewStruct(s.diffMap(lhs.Routes, rhs.Routes, routeID))
	ret.Services, _ = structpb.NewStruct(s.diffMap(lhs.Services, rhs.Services, ""))
	ret.PluginMetadata, _ = structpb.NewStruct(s.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, ""))
	ret.Ssl, _ = structpb.NewStruct(s.diffMap(lhs.SSLs, rhs.SSLs, ""))
	return ret
}

func (s *Server) diff(ctx context.Context, lhs, rhs *apisix.ApisixConfiguration) *serverpb.StageScopedApisixResources {
	ret := &serverpb.StageScopedApisixResources{}
	ret.Routes, _ = structpb.NewStruct(s.diffMap(lhs.Routes, rhs.Routes, ""))
	ret.Services, _ = structpb.NewStruct(s.diffMap(lhs.Services, rhs.Services, ""))
	ret.PluginMetadata, _ = structpb.NewStruct(s.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, ""))
	ret.Ssl, _ = structpb.NewStruct(s.diffMap(lhs.SSLs, rhs.SSLs, ""))
	return ret
}

func (s *Server) diffMap(lhs, rhs interface{}, id string) map[string]interface{} {
	lhsValue := reflect.ValueOf(lhs)
	rhsValue := reflect.ValueOf(rhs)
	diff := make(map[string]interface{})
	if lhsValue.Kind() != reflect.Map || rhsValue.Kind() != reflect.Map {
		return diff
	}
	lhsIter := lhsValue.MapRange()
	for lhsIter.Next() {
		key := lhsIter.Key()
		if len(id) != 0 && key.String() != id {
			rhsValue.SetMapIndex(key, reflect.Value{})
			continue
		}
		val := lhsIter.Value()
		rhsItemValue := rhsValue.MapIndex(key)
		if !rhsItemValue.IsValid() {
			diff[key.String()] = cmp.Diff(val.Interface(), nil)
		} else {
			diffStr := cmp.Diff(val.Interface(), rhsItemValue.Interface())
			if len(diffStr) != 0 {
				diff[key.String()] = diffStr
			}
			rhsValue.SetMapIndex(key, reflect.Value{})
		}
	}
	rhsIter := rhsValue.MapRange()
	for rhsIter.Next() {
		key := rhsIter.Key()
		if len(id) != 0 && key.String() != id {
			continue
		}
		val := rhsIter.Value()
		diff[key.String()] = cmp.Diff(nil, val.Interface())
	}
	return diff
}

// List ...
func (s *Server) List(ctx context.Context, req *serverpb.ListRequest) (*serverpb.ListResponse, error) {
	_, span := tracer.NewTracer("grpcServer").Start(ctx, "grpcServer/list", trace.WithAttributes(
		attribute.Bool("all", req.All),
		attribute.String("gateway", req.Gateway),
		attribute.String("stage", req.Stage),
		attribute.String("resource", req.Resource.String()),
	))
	defer span.End()
	resp := &serverpb.ListResponse{
		Data: make(map[string]*serverpb.StageScopedApisixResources),
	}
	if !req.All {
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		apisixRes := s.apisixConfStore.Get(stageKey)
		if req.Resource != nil {
			resourceKey, err := s.getRouteIDByResourceIdentity(apisixRes, req.Gateway, req.Stage, req.Resource)
			if err != nil {
				return nil, err
			}
			apisixRes.Routes = map[string]*apisix.Route{
				resourceKey: apisixRes.Routes[resourceKey],
			}
		}
		stagedApisixRes := map[string]interface{}{
			stageKey: apisixRes,
		}
		by, err := json.Marshal(stagedApisixRes)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(by, &resp.Data)
		return resp, nil
	}
	stagedApisixRes := s.apisixConfStore.GetAll()
	by, err := json.Marshal(stagedApisixRes)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	json.Unmarshal(by, &resp.Data)
	return resp, nil
}

// Healthz ...
func (s *Server) Healthz(ctx context.Context, req *empty.Empty) (*serverpb.HealthzResponse, error) {
	return &serverpb.HealthzResponse{
		Message: "ok",
	}, nil
}

// GetLeader ...
func (s *Server) GetLeader(ctx context.Context, in *empty.Empty) (*serverpb.GetLeaderResponse, error) {
	_, span := tracer.NewTracer("grpcServer").Start(ctx, "grpcServer/getLeader")
	defer span.End()
	resp := &serverpb.GetLeaderResponse{}
	if s.LeaderElector == nil {
		span.AddEvent("LeaderElector not found")
		return &serverpb.GetLeaderResponse{Code: 1, Message: "LeaderElector not found"}, nil
	}
	resp.Data = s.LeaderElector.Leader()
	return resp, nil
}

func (s *Server) getRouteIDByResourceIdentity(
	conf *apisix.ApisixConfiguration,
	gateway, stage string,
	resource *serverpb.ResourceIdentity,
) (string, error) {
	if conf == nil || resource == nil {
		return "", eris.New("resource id not found")
	}
	if id, ok := resource.ResourceIdentity.(*serverpb.ResourceIdentity_ResourceId); ok {
		return fmt.Sprintf("%s.%s.%d", gateway, stage, id.ResourceId), nil
	}
	resname_wrapper, ok := resource.ResourceIdentity.(*serverpb.ResourceIdentity_ResourceName)
	if !ok {
		s.logger.Error(
			"error when convert ResourceIdentity into ResourceIdentity_ResourceName",
			zap.Any("ResourceIdentity", resource.ResourceIdentity),
		)
		return "", eris.New("internal error")
	}

	// fixme: labels的resource name 超过64不能被apisix从etcd读取, 从源头没有写入 pkg/conversion/resource.go:62
	for _, route := range conf.Routes {
		if route.Metadata.Labels[config.BKAPIGatewayLabelKeyResourceName] == resname_wrapper.ResourceName {
			return route.ID, nil
		}
	}
	return "", eris.New("resource id not found")
}

func (s *Server) isDiffResultEmpty(result *serverpb.StageScopedApisixResources) bool {
	if result == nil {
		return true
	}
	return len(result.Routes.Fields) == 0 && len(result.Services.Fields) == 0 && len(result.Ssl.Fields) == 0 &&
		len(result.PluginMetadata.Fields) == 0
}
