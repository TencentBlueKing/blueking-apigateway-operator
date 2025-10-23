/*
 *  TencentBlueKing is pleased to support the open source community by making
 *  蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 *  Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 *  Licensed under the MIT License (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at
 *
 *      http://opensource.org/licenses/MIT
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under
 *  the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 *  either express or implied. See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 *   We undertake not to change the open source license (MIT license) applicable
 *   to the current version of the project delivered to anyone in the future.
 */

// Package entity ...
package entity

import (
	"context"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	json "github.com/json-iterator/go"
	"github.com/rotisserie/eris"
	"github.com/spf13/cast"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
)

// ApisixResource defines common function for apisix resources
type ApisixResource interface {
	GetID() string
	GetStageFromLabel() string

	GetCreateTime() int64
	GetUpdateTime() int64
	SetCreateTime(int64)
	SetUpdateTime(int64)

	ClearUnusedFields()
}

// Service apisix service object
// +k8s:deepcopy-gen=true
type Service struct {
	apisixv1.Metadata `json:",inline" yaml:",inline"`

	Upstream        *Upstream        `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	EnableWebsocket bool             `json:"enable_websocket,omitempty" yaml:"enable_websocket,omitempty"`
	Hosts           []string         `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Plugins         apisixv1.Plugins `json:"plugins" yaml:"plugins"`

	CreateTime int64 `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	UpdateTime int64 `json:"update_time,omitempty" yaml:"update_time,omitempty"`
}

// GetID will return the resource id
func (s *Service) GetID() string {
	return s.ID
}

// GetStageFromLabel will build the stage key from resource label
func (s *Service) GetStageFromLabel() string {
	return config.GenStagePrimaryKey(
		s.Labels[config.BKAPIGatewayLabelKeyGatewayName],
		s.Labels[config.BKAPIGatewayLabelKeyGatewayStage],
	)
}

// GetCreateTime GetCreateTime
func (s *Service) GetCreateTime() int64 { return s.CreateTime }

// GetUpdateTime GetUpdateTime
func (s *Service) GetUpdateTime() int64 { return s.UpdateTime }

// SetCreateTime SetCreateTime
func (s *Service) SetCreateTime(t int64) { s.CreateTime = t }

// SetUpdateTime SetUpdateTime
func (s *Service) SetUpdateTime(t int64) { s.UpdateTime = t }

// ClearUnusedFields clear desc
func (s *Service) ClearUnusedFields() { s.Desc = "" }

// Upstream route upstream
// +k8s:deepcopy-gen=true
type Upstream struct {
	Type          *string                       `json:"type,omitempty" yaml:"type,omitempty"`
	DiscoveryType *string                       `json:"discovery_type,omitempty" yaml:"discovery_type,omitempty"`
	ServiceName   *string                       `json:"service_name,omitempty" yaml:"service_name,omitempty"`
	HashOn        *string                       `json:"hash_on,omitempty" yaml:"hash_on,omitempty"`
	Key           *string                       `json:"key,omitempty" yaml:"key,omitempty"`
	Checks        *apisixv1.UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
	Scheme        *string                       `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	Retries       *int                          `json:"retries,omitempty" yaml:"retries,omitempty"`
	RetryTimeout  *int                          `json:"retry_timeout,omitempty" yaml:"retry_timeout,omitempty"`
	PassHost      *string                       `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`
	UpstreamHost  *string                       `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`
	Timeout       *apisixv1.UpstreamTimeout     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	TLS           *UpstreamTLS                  `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// UpstreamTLS tls info for upstream
type UpstreamTLS struct {
	ClientCert string `json:"client_cert" yaml:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key"`
}

// Route apisix route object
// +k8s:deepcopy-gen=true
type Route struct {
	apisixv1.Route `json:",inline" yaml:",inline"`
	Status         *int `json:"status,omitempty" yaml:"status,omitempty"`

	Upstream  *Upstream `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	ServiceID string    `json:"service_id,omitempty" yaml:"service_id,omitempty"`

	CreateTime int64 `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	UpdateTime int64 `json:"update_time,omitempty" yaml:"update_time,omitempty"`
}

// GetID will return the resource id
func (r *Route) GetID() string {
	return r.ID
}

// GetStageFromLabel will build the stage key from resource label
func (r *Route) GetStageFromLabel() string {
	return config.GenStagePrimaryKey(
		r.Labels[config.BKAPIGatewayLabelKeyGatewayName],
		r.Labels[config.BKAPIGatewayLabelKeyGatewayStage],
	)
}

// GetCreateTime GetCreateTime
func (r *Route) GetCreateTime() int64 { return r.CreateTime }

// GetUpdateTime GetUpdateTime
func (r *Route) GetUpdateTime() int64 { return r.UpdateTime }

// SetCreateTime SetCreateTime
func (r *Route) SetCreateTime(t int64) { r.CreateTime = t }

// SetUpdateTime SetUpdateTime
func (r *Route) SetUpdateTime(t int64) { r.UpdateTime = t }

// ClearUnusedFields clear desc
func (r *Route) ClearUnusedFields() { r.Desc = "" }

// SSL ...
// +k8s:deepcopy-gen=true
type SSL struct {
	apisixv1.Ssl `json:",inline" yaml:",inline"`

	CreateTime int64 `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	UpdateTime int64 `json:"update_time,omitempty" yaml:"update_time,omitempty"`
}

// GetID will return the resource id
func (s *SSL) GetID() string {
	return s.ID
}

// GetStageFromLabel will build the stage key from resource label
func (s *SSL) GetStageFromLabel() string {
	return config.GenStagePrimaryKey(
		s.Labels[config.BKAPIGatewayLabelKeyGatewayName],
		s.Labels[config.BKAPIGatewayLabelKeyGatewayStage],
	)
}

// GetCreateTime GetCreateTime
func (s *SSL) GetCreateTime() int64 { return s.CreateTime }

// GetUpdateTime GetUpdateTime
func (s *SSL) GetUpdateTime() int64 { return s.UpdateTime }

// SetCreateTime SetCreateTime
func (s *SSL) SetCreateTime(t int64) { s.CreateTime = t }

// SetUpdateTime SetUpdateTime
func (s *SSL) SetUpdateTime(t int64) { s.UpdateTime = t }

// ClearUnusedFields clear desc
func (s *SSL) ClearUnusedFields() {}

// PluginMetadata is resource definition for apisix plugin_metadata
// +k8s:deepcopy-gen=true
type PluginMetadata struct {
	runtime.RawExtension `json:",inline" yaml:",inline"`

	ID string `json:"id" yaml:"id"`
}

// GetID will return the resource id
func (pm *PluginMetadata) GetID() string {
	return pm.ID
}

// GetStageFromLabel will build the stage key from resource label
func (pm *PluginMetadata) GetStageFromLabel() string {
	return config.DefaultStageKey
}

// MarshalJSON is serializing method for plugin_metadata
func (pm *PluginMetadata) MarshalJSON() ([]byte, error) {
	by, err := json.Marshal(pm.RawExtension)
	if err != nil {
		return nil, err
	}
	var resMap map[string]interface{}
	json.Unmarshal(by, &resMap)
	resMap["id"] = pm.ID
	return json.Marshal(resMap)
}

// UnmarshalJSON is deserializing method for plugin_metadata
func (pm *PluginMetadata) UnmarshalJSON(in []byte) error {
	var resMap map[string]interface{}
	err := json.Unmarshal(in, &resMap)
	if err != nil {
		return err
	}
	var ok bool
	pm.ID, ok = resMap["id"].(string)
	if !ok {
		return eris.Errorf("unmarshal json failed: ID field is not string")
	}
	pm.Raw = in
	return nil
}

// MarshalYAML is serializing method for plugin_metadata
func (pm *PluginMetadata) MarshalYAML() (interface{}, error) {
	by, err := json.Marshal(pm.RawExtension)
	if err != nil {
		return nil, err
	}
	var resMap map[string]interface{}
	json.Unmarshal(by, &resMap)
	resMap["id"] = pm.ID
	return resMap, nil
}

// UnmarshalYAML is serializing method for plugin_metadata
func (pm *PluginMetadata) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var resMap map[string]interface{}
	err := unmarshal(&resMap)
	if err != nil {
		return err
	}
	var ok bool
	pm.ID, ok = resMap["id"].(string)
	if !ok {
		return eris.Errorf("unmarshal yaml failed: ID field is not string")
	}
	delete(resMap, "id")
	pm.Raw, _ = json.Marshal(resMap)
	return nil
}

// GetCreateTime GetCreateTime
func (pm *PluginMetadata) GetCreateTime() int64 { return 0 }

// GetUpdateTime GetUpdateTime
func (pm *PluginMetadata) GetUpdateTime() int64 { return 0 }

// SetCreateTime SetCreateTime
func (pm *PluginMetadata) SetCreateTime(t int64) {}

// SetUpdateTime SetUpdateTime
func (pm *PluginMetadata) SetUpdateTime(t int64) {}

// ClearUnusedFields clear desc
func (pm *PluginMetadata) ClearUnusedFields() {}

// NewPluginMetadata will build a new plugin metadata object
func NewPluginMetadata(name string, config map[string]interface{}) *PluginMetadata {
	var ret PluginMetadata
	by, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	ret.Raw = by
	ret.ID = name
	return &ret
}

// StreamRoute apisix stream route object
// +k8s:deepcopy-gen=true
type StreamRoute struct {
	apisixv1.Metadata `json:",inline" yaml:",inline"`
	Status            *int `json:"status,omitempty" yaml:"status,omitempty"`

	RemoteAddr string    `json:"remote_addr,omitempty" yaml:"remote_addr,omitempty"`
	ServerAddr string    `json:"server_addr,omitempty" yaml:"server_addr,omitempty"`
	ServerPort int       `json:"server_port,omitempty" yaml:"server_port,omitempty"`
	SNI        string    `json:"sni,omitempty" yaml:"sni,omitempty"`
	Upstream   *Upstream `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	ServiceID  string    `json:"service_id,omitempty" yaml:"service_id,omitempty"`

	CreateTime int64 `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	UpdateTime int64 `json:"update_time,omitempty" yaml:"update_time,omitempty"`
}

// GetID will return the resource id
func (r *StreamRoute) GetID() string {
	return r.ID
}

// GetStageFromLabel will build the stage key from resource label
func (r *StreamRoute) GetStageFromLabel() string {
	return config.GenStagePrimaryKey(
		r.Labels[config.BKAPIGatewayLabelKeyGatewayName],
		r.Labels[config.BKAPIGatewayLabelKeyGatewayStage],
	)
}

// GetCreateTime GetCreateTime
func (r *StreamRoute) GetCreateTime() int64 { return r.CreateTime }

// GetUpdateTime GetUpdateTime
func (r *StreamRoute) GetUpdateTime() int64 { return r.UpdateTime }

// SetCreateTime SetCreateTime
func (r *StreamRoute) SetCreateTime(t int64) { r.CreateTime = t }

// SetUpdateTime SetUpdateTime
func (r *StreamRoute) SetUpdateTime(t int64) { r.UpdateTime = t }

// ClearUnusedFields clear desc
func (r *StreamRoute) ClearUnusedFields() { r.Desc = "" }

// ApisixConfigurationStandalone apisix configuration structure
// +k8s:deepcopy-gen=true
type ApisixConfigurationStandalone struct {
	Routes          []*Route          `json:"routes,omitempty" yaml:"routes,omitempty"`
	StreamRoutes    []*StreamRoute    `json:"stream_routes,omitempty" yaml:"stream_routes,omitempty"`
	Services        []*Service        `json:"services,omitempty" yaml:"services,omitempty"`
	PluginMetadatas []*PluginMetadata `json:"plugin_metadata,omitempty" yaml:"plugin_metadata,omitempty"`
	SSLs            []*SSL            `json:"ssls,omitempty" yaml:"ssls,omitempty"`
}

// ApisixConfiguration apisix configuration structure
// +k8s:deepcopy-gen=true
type ApisixConfiguration struct {
	Routes          map[string]*Route          `json:"routes,omitempty" yaml:"routes,omitempty"`
	StreamRoutes    map[string]*StreamRoute    `json:"stream_routes,omitempty" yaml:"stream_routes,omitempty"`
	Services        map[string]*Service        `json:"services,omitempty" yaml:"services,omitempty"`
	PluginMetadatas map[string]*PluginMetadata `json:"plugin_metadata,omitempty" yaml:"plugin_metadata,omitempty"`
	SSLs            map[string]*SSL            `json:"ssls,omitempty" yaml:"ssls,omitempty"`
}

// NewEmptyApisixConfiguration will build a new apisix configuration object
func NewEmptyApisixConfiguration() *ApisixConfiguration {
	return &ApisixConfiguration{
		Routes:          make(map[string]*Route),
		StreamRoutes:    make(map[string]*StreamRoute),
		Services:        make(map[string]*Service),
		PluginMetadatas: make(map[string]*PluginMetadata),
		SSLs:            make(map[string]*SSL),
	}
}

// MergeFrom will merge input configuration into object itself
func (in *ApisixConfiguration) MergeFrom(out *ApisixConfiguration) {
	if in == nil || out == nil {
		return
	}
	for key, val := range out.Routes {
		in.Routes[key] = val
	}
	for key, val := range out.StreamRoutes {
		in.StreamRoutes[key] = val
	}
	for key, val := range out.Services {
		in.Services[key] = val
	}
	for key, val := range out.PluginMetadatas {
		in.PluginMetadatas[key] = val
	}
	for key, val := range out.SSLs {
		in.SSLs[key] = val
	}
}

// ToStandalone will convert apisix configuration into standalone mode
func (in *ApisixConfiguration) ToStandalone() *ApisixConfigurationStandalone {
	if in == nil {
		return nil
	}
	ret := &ApisixConfigurationStandalone{}
	for _, val := range in.Routes {
		ret.Routes = append(ret.Routes, val)
	}
	for _, val := range in.StreamRoutes {
		ret.StreamRoutes = append(ret.StreamRoutes, val)
	}
	for _, val := range in.Services {
		ret.Services = append(ret.Services, val)
	}
	for _, val := range in.SSLs {
		ret.SSLs = append(ret.SSLs, val)
	}
	for _, val := range in.PluginMetadatas {
		ret.PluginMetadatas = append(ret.PluginMetadatas, val)
	}
	return ret
}

// ToApisix will convert apisix standalone mode configuration into normal mode
func (in *ApisixConfigurationStandalone) ToApisix() *ApisixConfiguration {
	if in == nil {
		return nil
	}
	ret := NewEmptyApisixConfiguration()
	for _, val := range in.Routes {
		ret.Routes[val.GetID()] = val
	}
	for _, val := range in.StreamRoutes {
		ret.StreamRoutes[val.GetID()] = val
	}
	for _, val := range in.Services {
		ret.Services[val.GetID()] = val
	}
	for _, val := range in.SSLs {
		ret.SSLs[val.GetID()] = val
	}
	for _, val := range in.PluginMetadatas {
		ret.PluginMetadatas[val.GetID()] = val
	}
	return ret
}

// ExtractStagedConfiguration will extract a staged scoped apisix configuration with provided stage key
func (in *ApisixConfiguration) ExtractStagedConfiguration(stageName string) *ApisixConfiguration {
	if in == nil {
		return nil
	}
	ret := NewEmptyApisixConfiguration()
	for key, val := range in.Routes {
		if val.GetStageFromLabel() == stageName {
			ret.Routes[key] = val
		}
	}
	for key, val := range in.StreamRoutes {
		if val.GetStageFromLabel() == stageName {
			ret.StreamRoutes[key] = val
		}
	}
	for key, val := range in.Services {
		if val.GetStageFromLabel() == stageName {
			ret.Services[key] = val
		}
	}
	for key, val := range in.SSLs {
		if val.GetStageFromLabel() == stageName {
			ret.SSLs[key] = val
		}
	}
	for key, val := range in.PluginMetadatas {
		if val.GetStageFromLabel() == stageName {
			ret.PluginMetadatas[key] = val
		}
	}
	return ret
}

// ToStagedConfiguration will convert apisix configuration to staged apisix configuration map
func (in *ApisixConfiguration) ToStagedConfiguration() map[string]*ApisixConfiguration {
	if in == nil {
		return nil
	}
	ret := make(map[string]*ApisixConfiguration)
	stages := make(map[string]struct{})
	for _, val := range in.Routes {
		stages[val.GetStageFromLabel()] = struct{}{}
	}
	for _, val := range in.StreamRoutes {
		stages[val.GetStageFromLabel()] = struct{}{}
	}
	for _, val := range in.SSLs {
		stages[val.GetStageFromLabel()] = struct{}{}
	}
	for _, val := range in.Services {
		stages[val.GetStageFromLabel()] = struct{}{}
	}
	for _, val := range in.PluginMetadatas {
		stages[val.GetStageFromLabel()] = struct{}{}
	}
	for stage := range stages {
		ret[stage] = in.ExtractStagedConfiguration(stage)
	}
	return ret
}

// Statistic ...
func (in *ApisixConfiguration) Statistic() map[string]interface{} {
	ret := make(map[string]interface{})
	if in == nil {
		return ret
	}
	ret["routes_cnt"] = len(in.Routes)
	ret["services_cnt"] = len(in.Services)
	ret["plugin_metadata_cnt"] = len(in.PluginMetadatas)
	ret["ssl_cnt"] = len(in.SSLs)
	return ret
}

var emptyStageInfo = StageInfo{}

// ResourceKey ...
type ResourceKey struct {
	StageInfo
	ResourceName string
}

// StageInfo ...
type StageInfo struct {
	GatewayName string
	StageName   string
	PublishID   string
	Ctx         context.Context

	RetryCount int64
}

type Label struct {
	Gateway       string `json:"gateway.bk.tencent.com/gateway"`
	Stage         string `json:"gateway.bk.tencent.com/stage"`
	PublishId     string `json:"gateway.bk.tencent.com/publish-id"`
	ApisixVersion string `json:"gateway.bk.tencent.com/apisix-version"`
}

// ResourceMetadata describes the metadata of a resource object, which includes the
// resource kind and name. It is used by the watch process of the APIGEtcdWWatcher type.
type ResourceMetadata struct {
	Labels     Label `json:"labels"`
	APIVersion string
	Kind       constant.APISIXResource
	Name       string
	RetryCount int64 `json:"-" yaml:"-"`
	Ctx        context.Context
}

func (rm *ResourceMetadata) GetReleaseStageInfo() *ReleaseStageInfo {
	return &ReleaseStageInfo{
		Id:            rm.GetStageID(),
		PublishId:     cast.ToInt(rm.Labels.PublishId),
		ApisixVersion: rm.Labels.ApisixVersion,
		Ctx:           rm.Ctx,
	}
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

func (rm *ResourceMetadata) GetStageID() string {
	return config.GenStagePrimaryKey(rm.Labels.Gateway, rm.Labels.Stage)
}

type ReleaseStageInfo struct {
	ResourceMetadata
	Id              string `json:"id"`
	PublishId       int    `json:"publish_id"`
	PublishTime     string `json:"publish_time"`
	ApisixVersion   string `json:"apisix_version"`
	ResourceVersion string `json:"resource_version"`
	Ctx             context.Context
}
