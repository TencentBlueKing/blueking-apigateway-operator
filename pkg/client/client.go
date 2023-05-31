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

package client

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rotisserie/eris"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/serverpb"
)

// ResourcesClientWrapper ...
type ResourcesClientWrapper interface {
	serverpb.ResourcesClient
	Close()
}

// ResourcesClientWrapperImpl ...
type ResourcesClientWrapperImpl struct {
	client serverpb.ResourcesClient
	cc     *grpc.ClientConn
}

// GetLeaderResourcesClient ...
func GetLeaderResourcesClient() (ResourcesClientWrapper, error) {
	localAddr := serverAddr
	cc, err := grpc.Dial(
		localAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
		),
		grpc.WithStreamInterceptor(
			otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
		),
	)
	if err != nil {
		return nil, err
	}
	wrapper := &ResourcesClientWrapperImpl{
		client: serverpb.NewResourcesClient(cc),
		cc:     cc,
	}

	// query leader
	resp, err := wrapper.GetLeader(context.Background(), &empty.Empty{})
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return wrapper, eris.New(resp.Message)
	}
	leaderAddr := getAddrFromLeaderName(resp.Data)
	if len(leaderAddr) == 0 {
		return wrapper, eris.New("No leader addr")
	}
	wrapper.Close()
	leaderCC, err := grpc.Dial(
		leaderAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
		),
		grpc.WithStreamInterceptor(
			otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(otel.GetTracerProvider())),
		),
	)
	if err != nil {
		return nil, err
	}
	return &ResourcesClientWrapperImpl{
		client: serverpb.NewResourcesClient(leaderCC),
		cc:     leaderCC,
	}, nil
}

func getAddrFromLeaderName(leader string) string {
	// format somename-ip1,ip2,ip3
	splited := strings.Split(leader, "_")
	addrs := splited[len(splited)-1]
	if len(addrs) == 0 {
		return ""
	}
	addrList := strings.Split(addrs, ",")
	if ip := net.ParseIP(addrList[0]); ip == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", addrList[0], serverBindPort)
}

// Sync ...
func (c *ResourcesClientWrapperImpl) Sync(
	ctx context.Context,
	in *serverpb.SyncRequest,
	opts ...grpc.CallOption,
) (*serverpb.SyncResponse, error) {
	return c.client.Sync(ctx, in, opts...)
}

// Diff ...
func (c *ResourcesClientWrapperImpl) Diff(
	ctx context.Context,
	in *serverpb.DiffRequest,
	opts ...grpc.CallOption,
) (*serverpb.DiffResponse, error) {
	return c.client.Diff(ctx, in, opts...)
}

// List ...
func (c *ResourcesClientWrapperImpl) List(
	ctx context.Context,
	in *serverpb.ListRequest,
	opts ...grpc.CallOption,
) (*serverpb.ListResponse, error) {
	return c.client.List(ctx, in, opts...)
}

// Healthz ...
func (c *ResourcesClientWrapperImpl) Healthz(
	ctx context.Context,
	in *empty.Empty,
	opts ...grpc.CallOption,
) (*serverpb.HealthzResponse, error) {
	return c.client.Healthz(ctx, in, opts...)
}

// GetLeader ...
func (c *ResourcesClientWrapperImpl) GetLeader(
	ctx context.Context,
	in *empty.Empty,
	opts ...grpc.CallOption,
) (*serverpb.GetLeaderResponse, error) {
	return c.client.GetLeader(ctx, in, opts...)
}

// Close ...
func (c *ResourcesClientWrapperImpl) Close() {
	c.cc.Close()
}
