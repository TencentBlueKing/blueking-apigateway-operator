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
	"k8s.io/apimachinery/pkg/runtime"
)

type BkGatewayOperatorStatusItem string

const (
	BkGatewayOperatorStatusReady    BkGatewayOperatorStatusItem = "Ready"
	BkGatewayOperatorStatusNotReady BkGatewayOperatorStatusItem = "NotReady"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BkGatewayOperatorSpec defines the desired state of BkGatewayOperator
type BkGatewayOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DiscoveryType string               `json:"discoveryType"`
	ConfigSchema  runtime.RawExtension `json:"configSchema"`
}

// BkGatewayOperatorStatus defines the observed state of BkGatewayOperator
type BkGatewayOperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status status for bk gateway stage
	Status BkGatewayOperatorStatusItem `json:"status"`
	// Message message for bk gateway stage
	Message string `json:"message"`
	// when operator status is Ready and time pass the ReadyUntil, operator
	// should regard status as NotReady
	ReadyUntil metav1.Time `json:"readyUntil"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayOperator is the Schema for the BkGatewayOperator API
type BkGatewayOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayOperatorSpec   `json:"spec,omitempty"`
	Status BkGatewayOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayOperatorList contains a list of BkGatewayOperator
type BkGatewayOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayOperator{}, &BkGatewayOperatorList{})
}
