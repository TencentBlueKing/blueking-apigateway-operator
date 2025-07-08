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

package types

import (
	"context"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
)

// SupportMethods is supported discovery methods for registry
type SupportMethods int

// WatchSupported ...
const (
	// WatchSupported means supporting watch discovery
	WatchSupported SupportMethods = iota
	// ListSupported means supporting list discovery
	ListSupported
	// WatchAndListSupported means supporting both watch and list discovery
	WatchAndListSupported
)

// ManagedByLabelTag ...
const (
	// ManagedByLabelTag is label tag for BkGatewayEndpoints
	ManagedByLabelTag = "gateway.bk.tencent.com/managed-by"
	// EndpointsNameSeparator is separator for BkGatewayEndpoint name between registry name and service name
	EndpointsNameSeparator = ".bksp."
)

// Registry is user considered registry interface
type Registry interface {
	// Watch is a sync function which will register a callBack function.
	// If watch connection is closed with error, return the error message with close info.
	// If context is done, return watch function without error.
	// It is IMPORTANT to exit function when context is done, or memory leak would happen.
	Watch(
	ctx context.Context,
	svcName string,
	namespace string,
	svcConfig map[string]interface{},
	callBack CallBack,
	) error
	// List will list the endpoints of service name with service discovery config.
	List(
	svcName string,
	namespace string,
	svcConfig map[string]interface{},
	) (*gatewayv1beta1.BkGatewayEndpointsSpec, error)
	// DiscoveryMethods will inform the operator frame which method should be considered to discover the service endpoints.
	DiscoveryMethods() SupportMethods
	// Name is required to specify the name of the registry.
	// It will be the name of your operator, and cannot be changed or be identical to another operator's name
	// Your operator's name should satisfy the requirements of DNS subdomain
	Name() string
}

// CallBack is function for registry to transfer endpoints when service events happens.
type CallBack func(endpoints *gatewayv1beta1.BkGatewayEndpointsSpec) error
