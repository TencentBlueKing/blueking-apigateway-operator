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

package conversion

import (
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/rotisserie/eris"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
)

// convertService convert bk gateway service to apisix service
func (c *Converter) convertService(service *v1beta1.BkGatewayService) (*apisix.Service, error) {
	var err error
	newService := &apisix.Service{
		Metadata: apisixv1.Metadata{
			ID:     c.getID(service.Spec.ID, getObjectName(service.GetName(), service.GetNamespace())),
			Name:   getObjectName(service.GetName(), service.GetNamespace()),
			Desc:   service.Spec.Desc,
			Labels: c.getLabel(),
		},
		EnableWebsocket: service.Spec.EnableWebsocket,
	}

	if service.Spec.Upstream != nil {
		newService.Upstream, err = c.convertUpstream(service.TypeMeta, service.ObjectMeta, service.Spec.Upstream)
		if err != nil {
			return nil, eris.Wrapf(err, "convert upstream of service %s/%s failed",
				service.GetName(), service.GetNamespace())
		}

		if service.Spec.Upstream.Timeout != nil {
			newService.Upstream.Timeout = c.convertHTTPTimeout(service.Spec.Upstream.Timeout)
		}
	}

	pluginsMap := make(map[string]interface{})
	c.appendStagePlugins(pluginsMap)

	if len(service.Spec.Plugins) != 0 {
		for _, p := range service.Spec.Plugins {
			pluginName, pluginConfig := c.convertPlugin(p)
			pluginsMap[pluginName] = pluginConfig
		}
	}

	if len(pluginsMap) != 0 {
		newService.Plugins = pluginsMap
	}

	c.logger.Debugw("convert service", "service", service, "apisix service", newService)

	return newService, nil
}
