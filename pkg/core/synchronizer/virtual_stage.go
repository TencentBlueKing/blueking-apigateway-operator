// Package synchronizer ...
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

// Package synchronizer ...
package synchronizer

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

// HealthZRouteIDInner ...
const (
	HealthZRouteIDOuter = "micro-gateway-operator-healthz-outer"
	NotFoundHandling    = "micro-gateway-not-found-handling"
)

// VirtualStage combine some builtin routes
type VirtualStage struct {
	entity.ResourceMetadata
	apisixHealthzURI string

	logger *zap.SugaredLogger
}

// NewVirtualStage creates a new virtual stage
func NewVirtualStage(apisixHealthzURI string) *VirtualStage {
	metadata := entity.ResourceMetadata{
		Labels: entity.LabelInfo{
			Gateway: virtualGatewayName,
			Stage:   virtualStageName,
		},
	}

	return &VirtualStage{
		ResourceMetadata: metadata,
		apisixHealthzURI: apisixHealthzURI,
		logger:           logging.GetLogger().Named("virtual-stage"),
	}
}

func (s *VirtualStage) makeRouteMetadata(id string) entity.ResourceMetadata {
	return entity.ResourceMetadata{
		ID:     id,
		Name:   id,
		Labels: s.ResourceMetadata.Labels,
	}
}

func (s *VirtualStage) make404DefaultRoute() *entity.Route {
	return &entity.Route{
		ResourceMetadata: s.makeRouteMetadata(NotFoundHandling),
		URI:              "/*",
		Uris:             nil,
		Priority:         -100,
		Methods:          nil,
		Host:             "",
		Hosts:            nil,
		RemoteAddr:       "",
		RemoteAddrs:      nil,
		Vars:             nil,
		FilterFunc:       "",
		Script:           nil,
		ScriptID:         nil,
		Plugins: map[string]interface{}{
			"bk-error-wrapper":     map[string]interface{}{},
			"bk-not-found-handler": map[string]interface{}{},
			"file-logger": map[string]interface{}{
				"path": fileLoggerLogPath,
			}},
		PluginConfigID:  nil,
		Upstream:        nil,
		ServiceID:       nil,
		UpstreamID:      nil,
		ServiceProtocol: "",
		EnableWebsocket: false,
		Status:          1,
	}
}

func (s *VirtualStage) makeOuterHealthzRoute() *entity.Route {
	plugins := map[string]interface{}{
		"limit-req": map[string]interface{}{
			"rate":  float64(10),
			"burst": float64(10),
			"key":   "server_addr",
		},
		"mocking": map[string]interface{}{
			"content_type":     "text/plain",
			"response_example": "ok",
		},
	}

	return &entity.Route{
		ResourceMetadata: s.makeRouteMetadata(HealthZRouteIDOuter),
		URI:              s.apisixHealthzURI,
		Priority:         -100,
		Methods:          []string{"GET"},
		Plugins:          plugins,
		Status:           1,
	}
}

func (s *VirtualStage) makeExtraConfiguration() *entity.ApisixStageResource {
	var configuration entity.ApisixStageResource

	if extraApisixResourcesPath == "" {
		return &configuration
	}

	file, err := os.Open(extraApisixResourcesPath)
	if err != nil {
		s.logger.Errorw("open resource path", "err", err, "path", extraApisixResourcesPath)
		return &configuration
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		s.logger.Error("parse resource path", "err", err, "path", extraApisixResourcesPath)
	}

	return &configuration
}

// MakeConfiguration return the apisix configuration of virtual stage
func (s *VirtualStage) MakeConfiguration() *entity.ApisixStageResource {
	ret := entity.NewEmptyApisixConfiguration()
	extraConfiguration := s.makeExtraConfiguration()

	for _, service := range extraConfiguration.Services {
		if service != nil && service.ID != "" {
			service.Labels = s.Labels
			ret.Services[service.ID] = service
		}
	}

	for _, ssl := range extraConfiguration.SSLs {
		if ssl != nil && ssl.ID != "" {
			ssl.Labels = s.Labels
			ret.SSLs[ssl.ID] = ssl
		}
	}

	for _, route := range extraConfiguration.Routes {
		if route != nil && route.ID != "" {
			route.Labels = s.Labels
			ret.Routes[route.ID] = route
		}
	}

	for _, fn := range []func() *entity.Route{
		s.make404DefaultRoute,
		s.makeOuterHealthzRoute,
	} {
		route := fn()
		ret.Routes[route.ID] = route
	}

	return ret
}
