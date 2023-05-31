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

// BkGatewayServiceSpec defines the desired state of BkGatewayService
type BkGatewayServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +nullable
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Desc            string                   `json:"desc,omitempty"`
	EnableWebsocket bool                     `json:"enableWebsocket,omitempty"`
	Upstream        *BkGatewayUpstreamConfig `json:"upstream,omitempty"`
	Rewrite         *BkGatewayRewrite        `json:"rewrite,omitempty"`
	// +nullable
	Plugins []*BkGatewayPlugin `json:"plugins,omitempty"`
}

// BkGatewayServiceStatus defines the observed state of BkGatewayService
type BkGatewayServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status status for bk gateway service
	Status string `json:"status"`
	// Message message for bk gateway service
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayService is the Schema for the bkgatewayservices API
type BkGatewayService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayServiceSpec   `json:"spec,omitempty"`
	Status BkGatewayServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayServiceList contains a list of BkGatewayService
type BkGatewayServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayService{}, &BkGatewayServiceList{})
}

// BkGatewayServiceSorter  ...
type BkGatewayServiceSorter []*BkGatewayService

// Len ...
func (bgss BkGatewayServiceSorter) Len() int { return len(bgss) }

// Swap ...
func (bgss BkGatewayServiceSorter) Swap(i, j int) { bgss[i], bgss[j] = bgss[j], bgss[i] }

// Less ...
func (bgss BkGatewayServiceSorter) Less(i, j int) bool {
	if bgss[i].GetNamespace() < bgss[j].GetNamespace() {
		return true
	} else if bgss[i].GetNamespace() == bgss[j].GetNamespace() {
		if bgss[i].GetName() < bgss[j].GetName() {
			return true
		} else if bgss[i].GetName() == bgss[j].GetName() {
			return true
		}
		return false
	}
	return false
}
