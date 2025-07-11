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

// Package handler ...
package handler

import "fmt"

// SyncReq Sync api req
type SyncReq struct {
	Gateway string `json:"gateway,omitempty"`
	Stage   string `json:"stage,omitempty"`
	All     bool   `json:"all,omitempty"`
}

// DiffReq Diff api req
type DiffReq struct {
	Gateway  string        `json:"gateway"`
	Stage    string        `json:"stage"`
	Resource *ResourceInfo `json:"resource"`
	All      bool          `json:"all"`
}

// ResourceInfo resource
type ResourceInfo struct {
	ResourceId   int64  `json:"resource_id"`
	ResourceName string `json:"resource_name"`
}

// ToString resource ToString
func (r *ResourceInfo) ToString() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("resource_id:%d,resource_name:%s", r.ResourceId, r.ResourceName)
}

// DiffInfo diff api result.data
type DiffInfo map[string]*StageScopedApisixResources

// StageScopedApisixResources apisix resource
type StageScopedApisixResources struct {
	Routes         map[string]interface{} `json:"routes,omitempty"`
	Services       map[string]interface{} `json:"services,omitempty"`
	PluginMetadata map[string]interface{} `json:"plugin_metadata,omitempty"`
	Ssl            map[string]interface{} `json:"ssl,omitempty"`
}

// ListReq list api req
type ListReq struct {
	Gateway  string        `json:"gateway,omitempty"`
	Stage    string        `json:"stage,omitempty"`
	Resource *ResourceInfo `json:"resource,omitempty"`
	All      bool          `json:"all,omitempty"`
}

// ListInfo list api result.data
type ListInfo map[string]*StageScopedApisixResources
