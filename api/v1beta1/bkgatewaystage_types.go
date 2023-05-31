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

// BkGatewayStageSpec defines the desired state of BkGatewayStage
type BkGatewayStageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name string `json:"name"`
	// Domain domain name for stage
	Domain string `json:"domain"`
	// PathPrefix unified prefix for path
	PathPrefix string `json:"pathPrefix"`
	// Desc description for stage
	Desc string `json:"desc,omitempty"`
	// Vars environment vairiables
	Vars map[string]string `json:"vars"`
	// Rewrite rewrite config for stage
	Rewrite *BkGatewayRewrite `json:"rewrite"`
	// Plugins plugins for stage
	Plugins []*BkGatewayPlugin `json:"plugins,omitempty"`
}

// BkGatewayStageStatus defines the observed state of BkGatewayStage
type BkGatewayStageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status status for bk gateway stage
	Status string `json:"status"`
	// Message message for bk gateway stage
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayStage is the Schema for the bkgatewaystages API
type BkGatewayStage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayStageSpec   `json:"spec,omitempty"`
	Status BkGatewayStageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayStageList contains a list of BkGatewayStage
type BkGatewayStageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayStage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayStage{}, &BkGatewayStageList{})
}
