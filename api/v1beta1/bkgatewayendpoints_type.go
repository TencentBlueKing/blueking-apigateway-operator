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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BkGatewayEndpointsSpec defines the desired state of BkGatewayEndpoints
type BkGatewayEndpointsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Nodes []BkGatewayNode `json:"nodes"`
}

// BkGatewayEndpointsNode  ...
type BkGatewayEndpointsNode struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	Weight   int    `json:"weight"`
	Priority int    `json:"priority"`
}

// BkGatewayEndpointsStatus defines the observed state of BkGatewayEndpoints
type BkGatewayEndpointsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status status for bk gateway stage
	Status string `json:"status"`
	// Message message for bk gateway stage
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayEndpoints is the Schema for the BkGatewayEndpoints API
type BkGatewayEndpoints struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayEndpointsSpec   `json:"spec,omitempty"`
	Status BkGatewayEndpointsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayEndpointsList contains a list of BkGatewayEndpoints
type BkGatewayEndpointsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayEndpoints `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayEndpoints{}, &BkGatewayEndpointsList{})
}
