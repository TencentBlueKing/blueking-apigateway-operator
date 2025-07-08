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
)

const (
	DPStatusNormal          = 0
	DPStatusConfigNotSynced = 1000
	DPStatusConfigError     = 1001
	DPStatusPortNotUp       = 1002
	DPStatusControlAPIError = 1003

	DPStatusStringNormal          = "Normal"
	DPStatusStringConfigNotSynced = "ConfigNotSynced"
	DPStatusStringConfigError     = "ConfigError"
	DPStatusStringPortNotUp       = "PortNotUp"
	DPStatusStringControlAPIError = "ControlAPIError"

	CPStatusNormal              = 0
	CPStatusValidConfigFailed   = 2000
	CPStatusConvertConfigFailed = 2001
	CPStatusKubeAPIFailed       = 2002

	CPStatusStringNormal              = "Normal"
	CPStatusStringValidConfigFailed   = "ValidConfigFailed"
	CPStatusStringConvertConfigFailed = "ConvertConfigFailed"
	CPStatusStringKubeAPIFailed       = "KubeAPIFailed"
)

// BkGatewayConfigJwtAuth jwt auth config for bk gateway
type BkGatewayConfigJwtAuth struct {
	// Key key for jwt auth
	Key string `json:"key"`
	// Secret secret for jwt auth
	Secret string `json:"secret"`
}

// BkGatewayConfigController controller field for bk gateway config
type BkGatewayConfigController struct {
	// EdgeController server endpints
	Endpoints []string `json:"endpoints"`
	// EdgeController server base path
	BasePath string `json:"basePath"`
	// JwtAuth jwt auth config
	JwtAuth *BkGatewayConfigJwtAuth `json:"jwtAuth"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BkGatewayConfigSpec defines the desired state of BkGatewayConfig
type BkGatewayConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Controller controller config
	Name       string                     `json:"name"`
	Desc       string                     `json:"desc,omitempty"`
	Controller *BkGatewayConfigController `json:"controller"`
	// +nullable
	InstanceID string `json:"instanceID,omitempty"`
}

// BkGatewayConfigStatus defines the observed state of BkGatewayConfig
type BkGatewayConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status status for bk gateway
	Status string `json:"status"`
	// Message message for bk gateway
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BkGatewayConfig is the Schema for the bkgatewayconfigs API
type BkGatewayConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BkGatewayConfigSpec   `json:"spec,omitempty"`
	Status BkGatewayConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BkGatewayConfigList contains a list of BkGatewayConfig
type BkGatewayConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BkGatewayConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BkGatewayConfig{}, &BkGatewayConfigList{})
}
