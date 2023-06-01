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

package service

import (
	"context"
	"fmt"
	"reflect"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/protocol"
	"github.com/TencentBlueKing/blueking-apigateway-operator/internal/tracer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"

	"github.com/google/go-cmp/cmp"
	json "github.com/json-iterator/go"
	"github.com/pingcap/errors"
	"github.com/rotisserie/eris"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ResourceService struct {
	registry        registry.Registry
	committer       *commiter.Commiter
	apiSixConfStore synchronizer.ApisixConfigStore
	logger          *zap.SugaredLogger
}

func NewResourceService(
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore) *ResourceService {
	return &ResourceService{
		registry:        registry,
		committer:       committer,
		apiSixConfStore: apiSixConfStore,
		logger:          logging.GetLogger(),
	}
}

// Sync ...
func (r *ResourceService) Sync(ctx context.Context, req protocol.SyncReq) error {
	outgoingCtx, span := tracer.NewTracer("httpServer").Start(ctx, "httpServer/sync", trace.WithAttributes(
		attribute.Bool("all", req.All),
		attribute.String("gateway", req.Gateway),
		attribute.String("stage", req.Stage),
	))
	defer span.End()
	if !req.All {
		r.committer.ForceCommit(outgoingCtx, []registry.StageInfo{
			{
				GatewayName: req.Gateway,
				StageName:   req.Stage,
			},
		})
		return nil
	}
	stageList, err := r.registry.ListStages(outgoingCtx)
	if err != nil {
		span.RecordError(err)
		return err
	}
	r.committer.ForceCommit(outgoingCtx, stageList)
	return nil
}

// Diff ...
func (r *ResourceService) Diff(ctx context.Context, req *protocol.DiffReq) (protocol.DiffInfo, error) {
	outgoingCtx, span := tracer.NewTracer("httpServer").Start(ctx, "httpServer/diff", trace.WithAttributes(
		attribute.Bool("all", req.All),
		attribute.String("gateway", req.Gateway),
		attribute.String("stage", req.Stage),
		attribute.String("resource", req.Resource.ToString()),
	))
	defer span.End()
	resp := make(protocol.DiffInfo)
	var err error
	if !req.All {
		si := registry.StageInfo{
			GatewayName: req.Gateway,
			StageName:   req.Stage,
		}
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		originalApiSixResources := r.apiSixConfStore.Get(stageKey)
		apiSixResources, err := r.committer.ConvertEtcdKVToApisixConfiguration(outgoingCtx, si)
		if err != nil {
			// TODO
			span.RecordError(err)
			return nil, err
		}
		resourceKey, err := r.getRouteIDByResourceIdentity(
			originalApiSixResources,
			req.Gateway,
			req.Stage,
			req.Resource,
		)
		if err != nil {
			resourceKey, err = r.getRouteIDByResourceIdentity(apiSixResources, req.Gateway, req.Stage, req.Resource)
			if err != nil {
				return nil, err
			}
		}
		resp[stageKey] = r.diffWithRouteID(originalApiSixResources, apiSixResources, resourceKey)
		return resp, nil
	}
	stageList, err := r.registry.ListStages(outgoingCtx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	allApiSixResources := r.apiSixConfStore.GetAll()
	for _, stage := range stageList {
		stageKey := config.GenStagePrimaryKey(stage.GatewayName, stage.StageName)
		apiSixResources, itemErr := r.committer.ConvertEtcdKVToApisixConfiguration(outgoingCtx, stage)
		if itemErr != nil {
			err = errors.New(fmt.Sprintf("%s [stage %s failed: %s]", stage.StageName, stageKey, eris.ToString(err, true)))
			span.RecordError(err, trace.WithAttributes(
				attribute.String("stageKey", stageKey),
			))
			continue
		}
		diffResult := r.diff(allApiSixResources[stageKey], apiSixResources)
		if !r.isDiffResultEmpty(diffResult) {
			resp[stageKey] = diffResult
		}
	}
	return resp, nil
}

func (r *ResourceService) List(ctx context.Context, req *protocol.ListReq) (protocol.ListInfo, error) {
	_, span := tracer.NewTracer("httpServer").Start(ctx, "httpServer/list", trace.WithAttributes(
		attribute.Bool("all", req.All),
		attribute.String("gateway", req.Gateway),
		attribute.String("stage", req.Stage),
		attribute.String("resource", req.Resource.ToString()),
	))
	defer span.End()
	resp := make(protocol.ListInfo)
	if !req.All {
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		apiSixRes := r.apiSixConfStore.Get(stageKey)
		if req.Resource != nil {
			resourceKey, err := r.getRouteIDByResourceIdentity(apiSixRes, req.Gateway, req.Stage, req.Resource)
			if err != nil {
				return nil, err
			}
			apiSixRes.Routes = map[string]*apisix.Route{
				resourceKey: apiSixRes.Routes[resourceKey],
			}
		}
		stagedApiSixRes := map[string]interface{}{
			stageKey: apiSixRes,
		}
		by, err := json.Marshal(stagedApiSixRes)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(by, &resp)
		return resp, nil
	}
	stagedApiSixRes := r.apiSixConfStore.GetAll()
	by, err := json.Marshal(stagedApiSixRes)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	_ = json.Unmarshal(by, &resp)
	return resp, nil
}

func (r *ResourceService) diffWithRouteID(
	lhs, rhs *apisix.ApisixConfiguration,
	routeID string) *protocol.StageScopedApiSixResources {
	ret := &protocol.StageScopedApiSixResources{}
	ret.Routes = r.diffMap(lhs.Routes, rhs.Routes, routeID)
	ret.Services = r.diffMap(lhs.Services, rhs.Services, "")
	ret.PluginMetadata = r.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, "")
	ret.Ssl = r.diffMap(lhs.SSLs, rhs.SSLs, "")
	return ret
}

func (r *ResourceService) diff(lhs, rhs *apisix.ApisixConfiguration) *protocol.StageScopedApiSixResources {
	ret := &protocol.StageScopedApiSixResources{}
	ret.Routes = r.diffMap(lhs.Routes, rhs.Routes, "")
	ret.Services = r.diffMap(lhs.Services, rhs.Services, "")
	ret.PluginMetadata = r.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, "")
	ret.Ssl = r.diffMap(lhs.SSLs, rhs.SSLs, "")
	return ret
}

func (r *ResourceService) diffMap(lhs, rhs interface{}, id string) map[string]interface{} {
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

func (r *ResourceService) getRouteIDByResourceIdentity(
	conf *apisix.ApisixConfiguration,
	gateway, stage string,
	resource *protocol.ResourceInfo,
) (string, error) {
	if conf == nil || resource == nil {
		return "", eris.New("resource id not found")
	}
	if resource.ResourceId != 0 {
		return fmt.Sprintf("%s.%s.%d", gateway, stage, resource.ResourceId), nil
	}
	// fixme: labels的resource name 超过64不能被apisix从etcd读取, 从源头没有写入 pkg/conversion/resource_api.go:62
	for _, route := range conf.Routes {
		if route.Metadata.Labels[config.BKAPIGatewayLabelKeyResourceName] == resource.ResourceName {
			return route.ID, nil
		}
	}
	return "", eris.New("resource id not found")
}

func (r *ResourceService) isDiffResultEmpty(result *protocol.StageScopedApiSixResources) bool {
	if result == nil {
		return true
	}
	return len(result.Routes) == 0 && len(result.Services) == 0 && len(result.Ssl) == 0 &&
		len(result.PluginMetadata) == 0
}
