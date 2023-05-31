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

package synchronizer

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

// HealthZRouteIDInner ...
const (
	HealthZRouteIDInner = "micro-gateway-operator-healthz-inner"
	HealthZRouteIDOuter = "micro-gateway-operator-healthz-outer"
	NotFoundHandling    = "micro-gateway-not-found-handling"
	// HTTPHeaderKeyGatewayOperatorVersion http header key for GatewayOperatorVersion
	HTTPHeaderKeyGatewayOperatorVersion = "Gateway-Operator-Version"
)

// VirtualStage combine some builtin routes
type VirtualStage struct {
	labels          map[string]string
	resourceVersion string
	operatorURL     string

	logger *zap.SugaredLogger
}

// NewVirtualStage creates a new virtual stage
func NewVirtualStage(resourceVersion, operatorURL string) *VirtualStage {
	labels := make(map[string]string)
	labels[config.BKAPIGatewayLabelKeyGatewayName] = virtualGatewayName
	labels[config.BKAPIGatewayLabelKeyGatewayStage] = virtualStageName

	return &VirtualStage{
		labels:          labels,
		resourceVersion: resourceVersion,
		operatorURL:     operatorURL,
		logger:          logging.GetLogger().Named("virtual-stage"),
	}
}

func (s *VirtualStage) injectVirtualStageLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string, len(s.labels))
	}

	for k, v := range s.labels {
		labels[k] = v
	}

	return labels
}

func (s *VirtualStage) makeRouteMetadata(id string) apisixv1.Metadata {
	return apisixv1.Metadata{
		ID:     id,
		Name:   id,
		Labels: s.labels,
	}
}

func (s *VirtualStage) make404DefaultRoute() *apisix.Route {
	return &apisix.Route{
		Route: apisixv1.Route{
			Metadata: s.makeRouteMetadata(NotFoundHandling),
			Uri:      "/*",
			Priority: -100,
			Plugins: map[string]interface{}{
				"bk-error-wrapper":     map[string]interface{}{},
				"bk-not-found-handler": map[string]interface{}{},
				"file-logger": map[string]interface{}{
					"path": fileLoggerLogPath,
				},
			},
		},
		Status: utils.IntPtr(1),
	}
}

func (s *VirtualStage) makeInnerHealthzRoute() *apisix.Route {
	return &apisix.Route{
		Route: apisixv1.Route{
			Metadata: s.makeRouteMetadata(HealthZRouteIDInner),
			Host:     "localhost",
			Uri:      "/" + s.resourceVersion,
			Methods:  []string{"GET"},
			Timeout: &apisixv1.UpstreamTimeout{
				Connect: 2,
				Read:    2,
				Send:    2,
			},
			Plugins: map[string]interface{}{
				"proxy-rewrite": map[string]interface{}{
					"uri": s.operatorURL,
				},
				"response-rewrite": map[string]interface{}{
					"headers": map[string]interface{}{
						HTTPHeaderKeyGatewayOperatorVersion: s.resourceVersion,
					},
				},
			},
		},
		Upstream: &apisix.Upstream{
			Type: utils.StringPtr("roundrobin"),
			Nodes: []v1beta1.BkGatewayNode{
				{
					Host:   operatorExternalHost,
					Port:   operatorExternalHealthProbePort,
					Weight: 10,
				},
			},
		},
		Status: utils.IntPtr(1),
	}
}

func (s *VirtualStage) makeOuterHealthzRoute() *apisix.Route {
	plugins := map[string]interface{}{
		"proxy-rewrite": map[string]interface{}{
			"uri": s.operatorURL,
		},
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

	return &apisix.Route{
		Route: apisixv1.Route{
			Metadata: s.makeRouteMetadata(HealthZRouteIDOuter),
			Uri:      s.operatorURL,
			Priority: -100,
			Methods:  []string{"GET"},
			Plugins:  plugins,
		},
		Status: utils.IntPtr(1),
	}
}

func (s *VirtualStage) makeExtraConfiguration() *apisix.ApisixConfigurationStandalone {
	var configuration apisix.ApisixConfigurationStandalone

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
func (s *VirtualStage) MakeConfiguration() *apisix.ApisixConfiguration {
	ret := apisix.NewEmptyApisixConfiguration()
	extraConfiguration := s.makeExtraConfiguration()

	for _, service := range extraConfiguration.Services {
		if service != nil && service.ID != "" {
			service.Labels = s.injectVirtualStageLabels(service.Labels)
			ret.Services[service.ID] = service
		}
	}

	for _, ssl := range extraConfiguration.SSLs {
		if ssl != nil && ssl.ID != "" {
			ssl.Labels = s.injectVirtualStageLabels(ssl.Labels)
			ret.SSLs[ssl.ID] = ssl
		}
	}

	for _, route := range extraConfiguration.Routes {
		if route != nil && route.ID != "" {
			route.Labels = s.injectVirtualStageLabels(route.Labels)
			ret.Routes[route.ID] = route
		}
	}

	for _, fn := range []func() *apisix.Route{
		s.make404DefaultRoute,
		s.makeInnerHealthzRoute,
		s.makeOuterHealthzRoute,
	} {
		route := fn()
		ret.Routes[route.ID] = route
	}

	return ret
}
