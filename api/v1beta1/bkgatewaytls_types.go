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

// BkGatewayTLSSpec defines the desired state of BkGatewayTLS
type BkGatewayTLSSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Desc                string   `json:"desc"`
	GatewayTLSSecretRef string   `json:"gatewayTLSSecretRef"`
	SNIs                []string `json:"snis"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayTLS is the Schema for the BkGatewayTLS API
type BkGatewayTLS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BkGatewayTLSSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayTLSList contains a list of BkGatewayTLS
type BkGatewayTLSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayTLS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayTLS{}, &BkGatewayTLSList{})
}
