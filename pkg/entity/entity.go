/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 微网关(BlueKing - Micro APIGateway) available.
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

package entity

import (
	"context"
	"encoding/json"

	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
)

// ApisixResource defines common function for apisix resources
type ApisixResource interface {
	GetID() string
	GetReleaseInfo() *ReleaseInfo
	GetStageName() string
}

// 网关环境资源配置
type ApisixStageResource struct {
	Routes          map[string]*Route          `json:"routes,omitempty"`
	Services        map[string]*Service        `json:"services,omitempty"`
	SSLs            map[string]*SSL            `json:"ssls,omitempty"`
	PluginMetadatas map[string]*PluginMetadata `json:"plugin_metadatas,omitempty"`
}

// NewEmptyApisixConfiguration will build a new apisix configuration object
func NewEmptyApisixConfiguration() *ApisixStageResource {
	return &ApisixStageResource{
		Routes:          make(map[string]*Route),
		Services:        make(map[string]*Service),
		PluginMetadatas: make(map[string]*PluginMetadata),
		SSLs:            make(map[string]*SSL),
	}
}

// NewEmptyApisixGlobalResource ...
func NewEmptyApisixGlobalResource() *ApisixGlobalResource {
	return &ApisixGlobalResource{
		PluginMetadata: make(map[string]*PluginMetadata),
	}
}

// 全局资源配置
type ApisixGlobalResource struct {
	PluginMetadata map[string]*PluginMetadata `json:"plugin_metadata,omitempty"`
}

// Status ...
type Status uint8

// Route ...
type Route struct {
	ResourceMetadata
	URI             string                 `json:"uri,omitempty"`
	Uris            []string               `json:"uris,omitempty"`
	Desc            string                 `json:"desc,omitempty"`
	Priority        int                    `json:"priority,omitempty"`
	Methods         []string               `json:"methods,omitempty"`
	Host            string                 `json:"host,omitempty"`
	Hosts           []string               `json:"hosts,omitempty"`
	RemoteAddr      string                 `json:"remote_addr,omitempty"`
	RemoteAddrs     []string               `json:"remote_addrs,omitempty"`
	Vars            []interface{}          `json:"vars,omitempty"`
	FilterFunc      string                 `json:"filter_func,omitempty"`
	Script          interface{}            `json:"script,omitempty"`
	ScriptID        interface{}            `json:"script_id,omitempty"`
	Plugins         map[string]interface{} `json:"plugins,omitempty"`
	PluginConfigID  interface{}            `json:"plugin_config_id,omitempty"`
	Upstream        *UpstreamDef           `json:"upstream,omitempty"`
	ServiceID       interface{}            `json:"service_id,omitempty"`
	UpstreamID      interface{}            `json:"upstream_id,omitempty"`
	ServiceProtocol string                 `json:"service_protocol,omitempty"`
	EnableWebsocket bool                   `json:"enable_websocket,omitempty"`
	Status          Status                 `json:"status"`
}

// TimeoutValue ...
type (
	TimeoutValue float32
	Timeout      struct {
		Connect TimeoutValue `json:"connect,omitempty"`
		Send    TimeoutValue `json:"send,omitempty"`
		Read    TimeoutValue `json:"read,omitempty"`
	}
)

// Node ...
type Node struct {
	Host     string      `json:"host,omitempty"`
	Port     int         `json:"port,omitempty"`
	Weight   int         `json:"weight"`
	Metadata interface{} `json:"metadata,omitempty"`
	Priority int         `json:"priority,omitempty"`
}

// Healthy ...
type Healthy struct {
	Interval     int   `json:"interval,omitempty"`
	HttpStatuses []int `json:"http_statuses,omitempty"`
	Successes    int   `json:"successes,omitempty"`
}

// UnHealthy ...
type UnHealthy struct {
	Interval     int   `json:"interval,omitempty"`
	HTTPStatuses []int `json:"http_statuses,omitempty"`
	TCPFailures  int   `json:"tcp_failures,omitempty"`
	Timeouts     int   `json:"timeouts,omitempty"`
	HTTPFailures int   `json:"http_failures,omitempty"`
}

// Active ...
type Active struct {
	Type                   string       `json:"type,omitempty"`
	Timeout                TimeoutValue `json:"timeout,omitempty"`
	Concurrency            int          `json:"concurrency,omitempty"`
	Host                   string       `json:"host,omitempty"`
	Port                   int          `json:"port,omitempty"`
	HTTPPath               string       `json:"http_path,omitempty"`
	HTTPSVerifyCertificate bool         `json:"https_verify_certificate,omitempty"`
	Healthy                Healthy      `json:"healthy,omitempty"`
	UnHealthy              UnHealthy    `json:"unhealthy,omitempty"`
	ReqHeaders             []string     `json:"req_headers,omitempty"`
}

// Passive ...
type Passive struct {
	Type      string    `json:"type,omitempty"`
	Healthy   Healthy   `json:"healthy,omitempty"`
	UnHealthy UnHealthy `json:"unhealthy,omitempty"`
}

// HealthChecker ...
type HealthChecker struct {
	Active  Active  `json:"active,omitempty"`
	Passive Passive `json:"passive,omitempty"`
}

// UpstreamTLS ...
type UpstreamTLS struct {
	ClientCert   string `json:"client_cert,omitempty"`
	ClientKey    string `json:"client_key,omitempty"`
	ClientCertId string `json:"client_cert_id,omitempty"`
}

// UpstreamKeepalivePool ...
type UpstreamKeepalivePool struct {
	IdleTimeout *TimeoutValue `json:"idle_timeout,omitempty"`
	Requests    int           `json:"requests,omitempty"`
	Size        int           `json:"size"`
}

// UpstreamDef ...
type UpstreamDef struct {
	ResourceMetadata
	Nodes         interface{}            `json:"nodes,omitempty"`
	Retries       *int                   `json:"retries,omitempty"`
	Timeout       *Timeout               `json:"timeout,omitempty"`
	Type          string                 `json:"type,omitempty"`
	Checks        interface{}            `json:"checks,omitempty"`
	HashOn        string                 `json:"hash_on,omitempty"`
	Key           string                 `json:"key,omitempty"`
	Scheme        string                 `json:"scheme,omitempty"`
	DiscoveryType string                 `json:"discovery_type,omitempty"`
	DiscoveryArgs map[string]interface{} `json:"discovery_args,omitempty"`
	PassHost      string                 `json:"pass_host,omitempty"`
	UpstreamHost  string                 `json:"upstream_host,omitempty"`
	Desc          string                 `json:"desc,omitempty"`
	ServiceName   string                 `json:"service_name,omitempty"`
	TLS           *UpstreamTLS           `json:"tls,omitempty"`
	KeepalivePool *UpstreamKeepalivePool `json:"keepalive_pool,omitempty"`
	RetryTimeout  TimeoutValue           `json:"retry_timeout,omitempty"`
}

// Upstream ...
type Upstream struct {
	UpstreamDef
}

// Consumer ...
type Consumer struct {
	Username   string                 `json:"username"`
	Desc       string                 `json:"desc,omitempty"`
	Plugins    map[string]interface{} `json:"plugins,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	CreateTime int64                  `json:"create_time,omitempty"`
	UpdateTime int64                  `json:"update_time,omitempty"`
	GroupID    string                 `json:"group_id,omitempty"`
}

// ConsumerGroup ...
type ConsumerGroup struct {
	Desc       string                 `json:"desc,omitempty"`
	Plugins    map[string]interface{} `json:"plugins,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	CreateTime int64                  `json:"create_time,omitempty"`
	UpdateTime int64                  `json:"update_time,omitempty"`
}

// Service ...
type Service struct {
	ResourceMetadata
	Desc            string                 `json:"desc,omitempty"`
	Upstream        *UpstreamDef           `json:"upstream,omitempty"`
	UpstreamID      interface{}            `json:"upstream_id,omitempty"`
	Plugins         map[string]interface{} `json:"plugins,omitempty"`
	Script          string                 `json:"script,omitempty"`
	EnableWebsocket bool                   `json:"enable_websocket,omitempty"`
	Hosts           []string               `json:"hosts,omitempty"`
}

// GlobalRule ...
type GlobalRule struct {
	ResourceMetadata
	Plugins map[string]interface{} `json:"plugins"`
}

// PluginMetadataConf ...
type PluginMetadataConf map[string]interface{}

// PluginMetadata ...
type PluginMetadata struct {
	ResourceMetadata
	PluginMetadataConf
}

// UnmarshalJSON 解析PluginMetadataConf
func (c *PluginMetadataConf) UnmarshalJSON(dAtA []byte) error {
	temp := make(map[string]interface{})
	if err := json.Unmarshal(dAtA, &temp); err != nil {
		return err
	}
	*c = temp
	return nil
}

// MarshalJSON 将PluginMetadataConf转换为json
func (c *PluginMetadataConf) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(*c))
}

// ServerInfo ...
type ServerInfo struct {
	ResourceMetadata
	LastReportTime int64  `json:"last_report_time,omitempty"`
	UpTime         int64  `json:"up_time,omitempty"`
	BootTime       int64  `json:"boot_time,omitempty"`
	EtcdVersion    string `json:"etcd_version,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	Version        string `json:"version,omitempty"`
}

// PluginConfig ...
type PluginConfig struct {
	ResourceMetadata
	Desc    string                 `json:"desc,omitempty"`
	Plugins map[string]interface{} `json:"plugins"`
}

// SSLClient ...
type SSLClient struct {
	CA               string   `json:"ca,omitempty"`
	Depth            int      `json:"depth,omitempty"`
	SkipMtlsUriRegex []string `json:"skip_mtls_uri_regex,omitempty"`
}

// SSL ...
type SSL struct {
	ResourceMetadata
	Cert          string     `json:"cert,omitempty"`
	Key           string     `json:"key,omitempty"`
	Sni           string     `json:"sni,omitempty"`
	Snis          []string   `json:"snis,omitempty"`
	Certs         []string   `json:"certs,omitempty"`
	Type          string     `json:"type,omitempty"`
	Keys          []string   `json:"keys,omitempty"`
	ExpTime       int64      `json:"exptime,omitempty"`
	Status        int        `json:"status"`
	ValidityStart int64      `json:"validity_start,omitempty"`
	ValidityEnd   int64      `json:"validity_end,omitempty"`
	Client        *SSLClient `json:"client,omitempty"`
	SSLProtocols  []string   `json:"ssl_protocols,omitempty"`
}

// Proto ...
type Proto struct {
	ResourceMetadata
	Desc    string `json:"desc,omitempty"`
	Content string `json:"content"`
}

// StreamRouteProtocol ...
type StreamRouteProtocol struct {
	Name string                 `json:"name,omitempty"`
	Conf map[string]interface{} `json:"conf,omitempty"`
}

// StreamRoute ...
type StreamRoute struct {
	ResourceMetadata
	Desc       string                 `json:"desc,omitempty"`
	RemoteAddr string                 `json:"remote_addr,omitempty"`
	ServerAddr string                 `json:"server_addr,omitempty"`
	ServerPort int                    `json:"server_port,omitempty"`
	SNI        string                 `json:"sni,omitempty"`
	UpstreamID interface{}            `json:"upstream_id,omitempty"`
	Upstream   *UpstreamDef           `json:"upstream,omitempty"`
	ServiceID  interface{}            `json:"service_id,omitempty"`
	Plugins    map[string]interface{} `json:"plugins,omitempty"`
	Protocol   *StreamRouteProtocol   `json:"protocol,omitempty"`
}

// ResourceMetadata describes the metadata of a resource object, which includes the
// resource kind and name. It is used by the watch process of the APIGEtcdWWatcher type.
type ResourceMetadata struct {
	Labels     LabelInfo `json:"labels"`
	APIVersion string
	ID         string `json:"id"`
	Kind       constant.APISIXResource
	Name       string
	RetryCount int64 `json:"-" yaml:"-"`
	Ctx        context.Context
}

func (rm *ResourceMetadata) GetReleaseInfo() *ReleaseInfo {
	return &ReleaseInfo{
		ResourceMetadata: *rm,
		PublishId:        cast.ToInt(rm.Labels.PublishId),
		ApisixVersion:    rm.Labels.ApisixVersion,
		Ctx:              rm.Ctx,
	}
}

func (rm *ResourceMetadata) GetID() string {
	return rm.ID
}

func (rm *ResourceMetadata) GetStageName() string {
	return rm.Labels.Stage
}

func (rm *ResourceMetadata) GetGatewayName() string {
	return rm.Labels.Gateway
}

// IsEmpty check if the metadata object is empty
func (rm *ResourceMetadata) IsEmpty() bool {
	if rm == nil {
		return true
	}
	return rm.Labels.Gateway == "" && rm.Labels.Stage == ""
}

func (rm *ResourceMetadata) GetReleaseID() string {
	// stage相关资源都是按照stage维度来管理的
	if rm.Kind != constant.PluginMetadata {
		return config.GenStagePrimaryKey(rm.Labels.Gateway, rm.Labels.Stage)
	}
	return rm.ID
}

type ReleaseInfo struct {
	ResourceMetadata
	PublishId       int    `json:"publish_id"`
	PublishTime     string `json:"publish_time"`
	ApisixVersion   string `json:"apisix_version"`
	ResourceVersion string `json:"resource_version"`
	Ctx             context.Context
}

type LabelInfo struct {
	Gateway       string `json:"gateway.bk.tencent.com/gateway"`
	Stage         string `json:"gateway.bk.tencent.com/stage"`
	PublishId     string `json:"gateway.bk.tencent.com/publish-id"`
	ApisixVersion string `json:"gateway.bk.tencent.com/apisix-version"`
}

type GlobalResourceInfo struct {
}
