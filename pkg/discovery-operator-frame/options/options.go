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

package options

import (
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/discovery-operator-frame/types"
)

// FrameOptions is frame options
type FrameOptions struct {
	MetricsAddress         string
	HealthProbeBindAddress string
	LeaderElection         bool
	Registry               types.Registry
	ZapOpts                zap.Options
	ConfigSchema           map[string]interface{}
	RegisterNamespace      string
}

// DefaultOptions return the default frame options
// NOTES: Registry, RegisterNamespace and ConfigSchema is required
func DefaultOptions() FrameOptions {
	return FrameOptions{
		MetricsAddress:         ":8080",
		HealthProbeBindAddress: ":8081",
		LeaderElection:         true,
		ZapOpts: zap.Options{
			Development: true,
			EncoderConfigOptions: []zap.EncoderConfigOption{
				func(config *zapcore.EncoderConfig) { config.CallerKey = "caller" },
			},
		},
		// Registry:          &myRegistry{},
		// ConfigSchema:      map[string]interface{}{},
		// RegisterNamespace: "default",
	}
}
