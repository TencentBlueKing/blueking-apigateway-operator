/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2025 Tencent. All rights reserved.
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

// Package main ...
package main

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	frame "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/options"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/types"
)

// TestRegistry ...
type TestRegistry struct{}

// Watch ...
func (t *TestRegistry) Watch(
	ctx context.Context,
	svcName, namespace string,
	svcConfig map[string]interface{},
	callBack types.CallBack,
) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			eps := &gatewayv1beta1.BkGatewayEndpointsSpec{
				Nodes: []gatewayv1beta1.BkGatewayNode{
					{
						Host:   "0.0.0.0",
						Port:   8081,
						Weight: 100,
					},
					{
						Host:   "0.0.0.1",
						Port:   8080,
						Weight: 100,
					},
				},
			}
			err := callBack(eps)
			if err != nil {
				fmt.Println("Call back failed")
			}
		case <-ctx.Done():
			fmt.Println("Exit watch")
			time.Sleep(time.Minute * 5)
			return nil
		}
	}
}

// List ...
func (t *TestRegistry) List(
	svcName, namespace string,
	svcConfig map[string]interface{},
) (*gatewayv1beta1.BkGatewayEndpointsSpec, error) {
	eps := &gatewayv1beta1.BkGatewayEndpointsSpec{
		Nodes: []gatewayv1beta1.BkGatewayNode{
			{
				Host:   "0.0.0.0",
				Port:   8081,
				Weight: 100,
			},
			{
				Host:   "0.0.0.1",
				Port:   8080,
				Weight: 100,
			},
		},
	}
	return eps, nil
}

// DiscoveryMethods ...
func (t *TestRegistry) DiscoveryMethods() types.SupportMethods {
	return types.WatchAndListSupported
}

// Name ...
func (t *TestRegistry) Name() string {
	return "test"
}

func main() {
	opts := options.DefaultOptions()
	opts.Registry = &TestRegistry{}
	opts.ConfigSchema = make(map[string]interface{})
	opts.RegisterNamespace = "default"
	opts.ZapOpts.Level = zapcore.Level(-4)
	operator, err := frame.NewDiscoveryOperator(opts)
	if err != nil {
		fmt.Printf("Build operator failed: %v\n", err)
		return
	}
	if err = operator.Run(); err != nil {
		fmt.Printf("Start operator failed: %v\n", err)
		return
	}
}
