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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BkGatewayStreamResourceSpec defines the desired state of BkGatewayStreamResource
type BkGatewayStreamResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// just adapt for bkapigateway
	Name string `json:"name"`
	// +nullable
	ID *intstr.IntOrString `json:"id,omitempty"`
	// +nullable
	Desc string `json:"desc,omitempty"`
	// +nullable
	Service string `json:"service,omitempty"`
	// +nullable
	Upstream *BkGatewayUpstreamConfig `json:"upstream,omitempty"`
	// +nullable
	RemoteAddr string `json:"remote_addr,omitempty"`
	// +nullable
	ServerAddr string `json:"server_addr,omitempty"`
	// +nullable
	ServerPort int `json:"server_port,omitempty"`
	// +nullable
	SNI string `json:"sni,omitempty"`
}

// BkGatewayStreamResourceStatus defines the observed state of BkGatewayStreamResource
type BkGatewayStreamResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayStreamResource is the Schema for the bkgatewaystreamresources API
type BkGatewayStreamResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayStreamResourceSpec   `json:"spec,omitempty"`
	Status BkGatewayStreamResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayStreamResourceList contains a list of BkGatewayStreamResource
type BkGatewayStreamResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayStreamResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayStreamResource{}, &BkGatewayStreamResourceList{})
}
