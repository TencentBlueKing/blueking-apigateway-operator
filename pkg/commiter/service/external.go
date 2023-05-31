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

	v1beta1 "micro-gateway/api/v1beta1"
	"micro-gateway/pkg/registry"

	"github.com/rotisserie/eris"
)

/*

从registry获取扩展的的节点

*/

// ExternalNodeDiscoverer ...
type ExternalNodeDiscoverer interface {
	GetNodes(
		kind, workNamespace, serviceType, serviceName string,
		gatewayName string,
		stageName string,
	) ([]v1beta1.BkGatewayNode, error)
}

// RegistryExternalNodeDiscoverer ExternalNodeDiscoverer converts external service to upstream node
type RegistryExternalNodeDiscoverer struct {
	resourceRegistry registry.Registry
}

// NewRegistryExternalNodeDiscoverer NewExternalNodeDiscoverer create new external service converter
func NewRegistryExternalNodeDiscoverer(resourceRegistry registry.Registry) ExternalNodeDiscoverer {
	return &RegistryExternalNodeDiscoverer{
		resourceRegistry: resourceRegistry,
	}
}

// GetNodes get upstream nodes according to external service name
func (rd *RegistryExternalNodeDiscoverer) GetNodes(
	kind, workNamespace, serviceType, serviceName string,
	gatewayName string,
	stageName string,
) ([]v1beta1.BkGatewayNode, error) {
	eps := &v1beta1.BkGatewayEndpoints{}
	if err := rd.resourceRegistry.Get(context.TODO(), registry.ResourceKey{
		ResourceName: fmt.Sprintf("%s.%s.%s", serviceType, kind, serviceName),
		StageInfo: registry.StageInfo{
			GatewayName: gatewayName,
			StageName:   stageName,
		},
	}, eps); err != nil {
		return nil, eris.Wrapf(err,
			"get external service enpdoints failed by type %s and service name %s",
			serviceType,
			serviceName,
		)
	}

	var nodes []v1beta1.BkGatewayNode
	for _, epNode := range eps.Spec.Nodes {
		nodes = append(nodes, v1beta1.BkGatewayNode{
			Host:     epNode.Host,
			Port:     epNode.Port,
			Weight:   epNode.Weight,
			Priority: epNode.Priority,
		})
	}
	return nodes, nil
}
