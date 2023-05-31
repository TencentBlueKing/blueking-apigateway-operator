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
)

// DataPlaneStatus ...
type DataPlaneStatus struct {
	Type         string `json:"type"`
	Version      string `json:"apisixVersion"`
	ConfigCenter string `json:"configCenter"`
	//+kubebuilder:pruning:PreserveUnknownFields
	PluginSchema runtime.RawExtension `json:"pluginSchema"`
	Status       int                  `json:"status"`
	Message      string               `json:"message"`
}

// DiscoverPluginStatus ...
type DiscoverPluginStatus struct {
	Type       string      `json:"discoveryType"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	Services   []string    `json:"services"`
	Message    string      `json:"message"`
	ReadyUntil metav1.Time `json:"readyUntil"`
}

// ControlPlaneStatus ...
type ControlPlaneStatus struct {
	CurConfigVersion       string                  `json:"curConfigVersion"`
	EffectiveConfigVersion string                  `json:"effectiveConfigVersion"`
	Status                 int                     `json:"status"`
	Message                string                  `json:"message"`
	DiscoveryPlugins       []*DiscoverPluginStatus `json:"discoveryPlugins,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BkGatewayInstanceSpec defines the desired state of BkGatewayInstance
type BkGatewayInstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ReportInterval metav1.Duration `json:"reportInterval"`
}

// BkGatewayInstanceStatus defines the observed state of BkGatewayInstance
type BkGatewayInstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DataPlane      *DataPlaneStatus    `json:"dataPlane"`
	ControlPlane   *ControlPlaneStatus `json:"controlPlane"`
	LastUpdateTime metav1.Time         `json:"lastUpdateTime"`
}

//+kubebuilder:object:root=true

// BkGatewayInstance is the Schema for the bkgatewayinstances API
type BkGatewayInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayInstanceSpec   `json:"spec,omitempty"`
	Status BkGatewayInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayInstanceList contains a list of BkGatewayInstance
type BkGatewayInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayInstance{}, &BkGatewayInstanceList{})
}
