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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BkGatewayPlugin ...
type BkGatewayPlugin struct {
	// Name name of plugin
	Name string `json:"name"`
	// Config parameter of plugin
	//+kubebuilder:pruning:PreserveUnknownFields
	Config runtime.RawExtension `json:"config,omitempty"`
}

// BkGatewayResourceHTTPRewrite ...
type BkGatewayResourceHTTPRewrite struct {
	Enabled bool              `json:"enabled,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Path    string            `json:"path,omitempty"`
	Method  string            `json:"method,omitempty"`

	// default: (priority low) resource header -> stage header -> service header
	StageHeadersMode   string `json:"stageHeaders,omitempty"`
	ServiceHeadersMode string `json:"serviceHeaders,omitempty"`
}

// BkGatewayResourceSpec ..
type BkGatewayResourceSpec struct {
	// just adapt for bkapigateway
	Name string `json:"name"`
	// +nullable
	ID           *intstr.IntOrString `json:"id,omitempty"`
	Desc         string              `json:"desc,omitempty"`
	Protocol     string              `json:"protocol,omitempty"`
	URI          string              `json:"uri"`
	MatchSubPath bool                `json:"matchSubPath,omitempty"`
	Methods      []string            `json:"methods"`
	// +nullable
	Timeout *UpstreamTimeout `json:"timeout,omitempty"`
	Service string           `json:"service,omitempty"`
	// +nullable
	Upstream *BkGatewayUpstreamConfig      `json:"upstream,omitempty"`
	Rewrite  *BkGatewayResourceHTTPRewrite `json:"rewrite,omitempty"`
	// +nullable
	Plugins         []*BkGatewayPlugin `json:"plugins,omitempty"`
	EnableWebsocket bool               `json:"enableWebsocket,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BkGatewayResourceStatus defines the observed state of BkGatewayResource
type BkGatewayResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status status for bk gateway stage
	Status string `json:"status"`
	// Message message for bk gateway stage
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayResource is the Schema for the bkgatewayresources API
type BkGatewayResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayResourceSpec   `json:"spec,omitempty"`
	Status BkGatewayResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayResourceList contains a list of BkGatewayResource
type BkGatewayResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayResource{}, &BkGatewayResourceList{})
}

// BkGatewayResourceSorter ...
type BkGatewayResourceSorter []*BkGatewayResource

// Len ...
func (bgrs BkGatewayResourceSorter) Len() int { return len(bgrs) }

// Swap ...
func (bgrs BkGatewayResourceSorter) Swap(i, j int) { bgrs[i], bgrs[j] = bgrs[j], bgrs[i] }

// Less ...
func (bgrs BkGatewayResourceSorter) Less(i, j int) bool {
	if bgrs[i].GetNamespace() < bgrs[j].GetNamespace() {
		return true
	} else if bgrs[i].GetNamespace() == bgrs[j].GetNamespace() {
		if bgrs[i].GetName() < bgrs[j].GetName() {
			return true
		} else if bgrs[i].GetName() == bgrs[j].GetName() {
			return true
		}
		return false
	}
	return false
}
