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

// +kubebuilder:validation:Optional
package v1beta1

import (
	"net"
	"strconv"
	"time"

	"micro-gateway/pkg/config"
	"micro-gateway/pkg/utils"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	json "github.com/json-iterator/go"
	"github.com/rotisserie/eris"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	BkGatewayResourceTypeName       = "BkGatewayResource"
	BkGatewayServiceTypeName        = "BkGatewayService"
	BkGatewayTLSTypeName            = "BkGatewayTLS"
	BkGatewayPluginMetadataTypeName = "BkGatewayPluginMetadata"
	BkGatewayStageTypeName          = "BkGatewayStage"
	BkGatewayConfigTypeName         = "BkGatewayConfig"
	BkGatewayInstanceTypeName       = "BkGatewayInstance"
)

// BkGatewayNode node of upstream
type BkGatewayNode struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Weight   int    `json:"weight" yaml:"weight"`
	Priority *int   `json:"priority,omitempty" yaml:"priority,omitempty"`
}

type (
	BkGatewayNodeList      []BkGatewayNode
	apisixCompatedNodeList map[string]int
)

// MarshalJSON ...
func (nodes *BkGatewayNodeList) MarshalJSON() ([]byte, error) {
	var nodeList []BkGatewayNode = *nodes
	return json.Marshal(&nodeList)
}

// UnmarshalJSON ...
func (nodes *BkGatewayNodeList) UnmarshalJSON(in []byte) error {
	nodeList := make([]BkGatewayNode, 0)
	err := json.Unmarshal(in, &nodeList)
	if err == nil {
		*nodes = nodeList
		return nil
	}

	nodeList = make([]BkGatewayNode, 0)
	compactedNodeList := make(apisixCompatedNodeList)
	err = json.Unmarshal(in, &compactedNodeList)
	if err != nil {
		return err
	}
	for node, weight := range compactedNodeList {
		host, portstr, err := net.SplitHostPort(node)
		if err != nil {
			setupLog.Error(
				err,
				"Split host port from apisix upstream node failed",
				"node",
				node,
				"nodeMap",
				compactedNodeList,
			)
			continue
		}
		port, err := strconv.Atoi(portstr)
		if err != nil {
			setupLog.Error(
				err,
				"convert port to integer failed",
				"port",
				port,
				"node",
				node,
				"nodeMap",
				compactedNodeList,
			)
			continue
		}
		nodeList = append(nodeList, BkGatewayNode{Host: host, Port: port, Weight: weight, Priority: utils.IntPtr(0)})
	}
	*nodes = nodeList
	return nil
}

// MarshalYAML ...
func (nodes *BkGatewayNodeList) MarshalYAML() (interface{}, error) {
	var nodeList []BkGatewayNode = *nodes
	return nodeList, nil
}

// UnmarshalYAML ...
func (nodes *BkGatewayNodeList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	nodeList := make([]BkGatewayNode, 0)
	err := unmarshal(&nodeList)
	if err == nil {
		*nodes = nodeList
		return nil
	}

	nodeList = make([]BkGatewayNode, 0)
	compactedNodeList := make(apisixCompatedNodeList)
	err = unmarshal(&compactedNodeList)
	if err != nil {
		return err
	}
	for node, weight := range compactedNodeList {
		host, portstr, err := net.SplitHostPort(node)
		if err != nil {
			setupLog.Error(
				err,
				"Split host port from apisix upstream node failed",
				"node",
				node,
				"nodeMap",
				compactedNodeList,
			)
			continue
		}
		port, err := strconv.Atoi(portstr)
		if err != nil {
			setupLog.Error(
				err,
				"convert port to integer failed",
				"port",
				port,
				"node",
				node,
				"nodeMap",
				compactedNodeList,
			)
			continue
		}
		nodeList = append(nodeList, BkGatewayNode{Host: host, Port: port, Weight: weight, Priority: utils.IntPtr(0)})
	}
	*nodes = nodeList
	return nil
}

// BkGatewayUpstreamConfig upstream config for bk gateway
type BkGatewayUpstreamConfig struct { // +nullable
	// +nullable
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// +nullable
	HashOn string `json:"hashOn,omitempty" yaml:"hashOn,omitempty"`
	// +nullable
	Key string `json:"key,omitempty" yaml:"key,omitempty"`
	// +nullable
	Checks *UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
	// +nullable
	Nodes BkGatewayNodeList `json:"nodes" yaml:"nodes"`
	// +nullable
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	// +nullable
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`
	// +nullable
	RetryTimeout *int `json:"retryTimeout,omitempty" yaml:"retryTimeout,omitempty"`
	// +nullable
	Timeout *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// +nullable
	DiscoveryType string `json:"discoveryType"`
	// +nullable
	ServiceName string `json:"serviceName"`
	// +nullable
	PassHost string `json:"passHost,omitempty" yaml:"passHost,omitempty"`
	// +nullable
	UpstreamHost string `json:"upstreamHost,omitempty" yaml:"upstreamHost,omitempty"`
	// +nullable
	ExternalDiscoveryType string `json:"externalDiscoveryType,omitempty"`
	//+kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	ExternalDiscoveryConfig runtime.RawExtension `json:"externalDiscoveryConfig,omitempty"`
	// +nullable
	TLSEnable bool `json:"tlsEnable"`
}

// UpstreamTimeout is settings for the read, send and connect to the upstream.
type UpstreamTimeout struct {
	// +nullable
	Connect *intstr.IntOrString `json:"connect,omitempty" yaml:"connect,omitempty"`
	// +nullable
	Send *intstr.IntOrString `json:"send,omitempty" yaml:"send,omitempty"`
	// +nullable
	Read *intstr.IntOrString `json:"read,omitempty" yaml:"read,omitempty"`
}

// TLS ...
type TLS struct {
	TLSCert             `json:",inline" yaml:",inline"`
	GatewayTLSSecretRef string `json:"gatewayTLSSecretRef"`
}

// TLSCert ...
type TLSCert struct {
	CACert string   `json:"-"`
	Cert   string   `json:"-"`
	Key    string   `json:"-"`
	SNIs   []string `json:"-"`
}

// GetTLSCertFromSecret ...
func GetTLSCertFromSecret(secret *v1.Secret) (*TLSCert, error) {
	var ok bool
	tlsCert := &TLSCert{}

	caData, ok := secret.Data[config.SecretCACertKey]
	if !ok {
		tlsCert.CACert = secret.StringData[config.SecretCACertKey]
	} else {
		tlsCert.CACert = string(caData)
	}

	certData, ok := secret.Data[config.SecretCertKey]
	if !ok {
		tlsCert.Cert, ok = secret.StringData[config.SecretCertKey]
		if !ok {
			return nil, eris.New("No cert field in secret")
		}
	} else {
		tlsCert.Cert = string(certData)
	}

	keyData, ok := secret.Data[config.SecretKeyKey]
	if !ok {
		tlsCert.Key, ok = secret.StringData[config.SecretKeyKey]
		if !ok {
			return nil, eris.New("No key field in secret")
		}
	} else {
		tlsCert.Key = string(keyData)
	}

	return tlsCert, nil
}

// BkGatewayRewrite ...
type BkGatewayRewrite struct {
	// Enabled if rewrite is enabled
	Enabled bool `json:"enabled,omitempty"`
	// Headers headers for rewrite
	Headers map[string]string `json:"headers,omitempty"`
}

// UpstreamHealthCheck defines the active and/or passive health check for an Upstream,
// with the upstream health check feature, pods can be kicked out or joined in quickly,
// if the feedback of Kubernetes liveness/readiness probe is long.
// +k8s:deepcopy-gen=true
type UpstreamHealthCheck struct {
	// +nullable
	Active *UpstreamActiveHealthCheck `json:"active,omitempty" yaml:"active,omitempty"`
	// +nullable
	Passive *UpstreamPassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// UpstreamActiveHealthCheck defines the active kind of upstream health check.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheck struct {
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// +nullable
	Timeout int `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// +nullable
	Concurrency int `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	// +nullable
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	// +nullable
	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`
	// +nullable
	HTTPPath string `json:"httpPath,omitempty" yaml:"httpPath,omitempty"`
	// +nullable
	HTTPSVerifyCert bool `json:"httpsVerifyCertificate,omitempty" yaml:"httpsVerifyCertificate,omitempty"`
	// +nullable
	HTTPRequestHeaders []string `json:"reqHeaders,omitempty" yaml:"reqHeaders,omitempty"`
	// +nullable
	Healthy UpstreamActiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	// +nullable
	Unhealthy UpstreamActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamPassiveHealthCheck defines the passive kind of upstream health check.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheck struct {
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// +nullable
	Healthy UpstreamPassiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	// +nullable
	Unhealthy UpstreamPassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamActiveHealthCheckHealthy defines the conditions to judge whether
// an upstream node is healthy with the active manner.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheckHealthy struct {
	UpstreamPassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	// +nullable
	Interval int `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// UpstreamPassiveHealthCheckHealthy defines the conditions to judge whether
// an upstream node is healthy with the passive manner.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheckHealthy struct {
	// +nullable
	HTTPStatuses []int `json:"httpStatuses,omitempty" yaml:"httpStatuses,omitempty"`
	// +nullable
	Successes int `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// UpstreamActiveHealthCheckUnhealthy defines the conditions to judge whether
// an upstream node is unhealthy with the active manager.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheckUnhealthy struct {
	UpstreamPassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	// +nullable
	Interval int `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// UpstreamPassiveHealthCheckUnhealthy defines the conditions to judge whether
// an upstream node is unhealthy with the passive manager.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheckUnhealthy struct {
	// +nullable
	HTTPStatuses []int `json:"httpStatuses,omitempty" yaml:"httpStatuses,omitempty"`
	// +nullable
	HTTPFailures int `json:"httpFailures,omitempty" yaml:"httpFailures,omitempty"`
	// +nullable
	TCPFailures int `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`
	// +nullable
	Timeouts int `json:"timeouts,omitempty" yaml:"timeouts,omitempty"`
}

// ConvertToAPISIXv1Check convert UpstreamHealthCheck to apisix v1 UpstreamHealthCheck
func (check *UpstreamHealthCheck) ConvertToAPISIXv1Check() *apisixv1.UpstreamHealthCheck {
	if check == nil || (check.Active == nil && check.Passive == nil) {
		return nil
	}
	ret := &apisixv1.UpstreamHealthCheck{}
	if check.Active != nil {
		ret.Active = &apisixv1.UpstreamActiveHealthCheck{
			Type:               check.Active.Type,
			Timeout:            check.Active.Timeout,
			Concurrency:        check.Active.Concurrency,
			Host:               check.Active.Host,
			Port:               check.Active.Port,
			HTTPPath:           check.Active.HTTPPath,
			HTTPSVerifyCert:    check.Active.HTTPSVerifyCert,
			HTTPRequestHeaders: check.Active.HTTPRequestHeaders,
			Healthy: apisixv1.UpstreamActiveHealthCheckHealthy{
				UpstreamPassiveHealthCheckHealthy: apisixv1.UpstreamPassiveHealthCheckHealthy{
					HTTPStatuses: check.Active.Healthy.HTTPStatuses,
					Successes:    check.Active.Healthy.Successes,
				},
				Interval: check.Active.Healthy.Interval,
			},
			Unhealthy: apisixv1.UpstreamActiveHealthCheckUnhealthy{
				UpstreamPassiveHealthCheckUnhealthy: apisixv1.UpstreamPassiveHealthCheckUnhealthy{
					HTTPStatuses: check.Active.Unhealthy.HTTPStatuses,
					HTTPFailures: check.Active.Unhealthy.HTTPFailures,
					TCPFailures:  check.Active.Unhealthy.TCPFailures,
					Timeouts:     check.Active.Unhealthy.Timeouts,
				},
				Interval: check.Active.Unhealthy.Interval,
			},
		}
	}
	if check.Passive != nil {
		ret.Passive = &apisixv1.UpstreamPassiveHealthCheck{
			Type: check.Passive.Type,
			Healthy: apisixv1.UpstreamPassiveHealthCheckHealthy{
				HTTPStatuses: check.Active.Healthy.HTTPStatuses,
				Successes:    check.Active.Healthy.Successes,
			},
			Unhealthy: apisixv1.UpstreamPassiveHealthCheckUnhealthy{
				HTTPStatuses: check.Active.Unhealthy.HTTPStatuses,
				HTTPFailures: check.Active.Unhealthy.HTTPFailures,
				TCPFailures:  check.Active.Unhealthy.TCPFailures,
				Timeouts:     check.Active.Unhealthy.Timeouts,
			},
		}
	}
	return ret
}

// ParseDuration ...
func ParseDuration(d *intstr.IntOrString) time.Duration {
	if d == nil {
		return time.Duration(0)
	}
	if d.Type == intstr.Int {
		ret := time.Duration(d.IntVal) * time.Second
		return ret
	} else {
		ret, err := time.ParseDuration(d.StrVal)
		if err != nil {
			return time.Duration(0)
		}
		return ret
	}
}

// FormatDuration ...
func FormatDuration(d time.Duration) *intstr.IntOrString {
	ret := intstr.FromString(d.String())
	return &ret
}
