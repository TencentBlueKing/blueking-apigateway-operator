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

package registry

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
}

// Key returns the stage key
func (s *StageInfo) Key() string {
	return s.GatewayName + "/" + s.StageName
}

// IsEmpty checks if the stage info is absent
func (si StageInfo) IsEmpty() bool {
	return si.GatewayName == "" && si.StageName == ""
}

// ResourceMetadata describes the metadata of a resource object, which includes the
// resource kind and name. It is used by the watch process of the Registry type.
type ResourceMetadata struct {
	StageInfo
	APIVersion string
	Kind       string
	Name       string
	RetryCount int64 `json:"-" yaml:"-"`
}

// IsEmpty check if the metadata object is empty
func (rm *ResourceMetadata) IsEmpty() bool {
	if rm == nil {
		return true
	}
	return rm.StageInfo.IsEmpty()
}

// Registry defines ways of retrieving gateway-related data from different kinds
// of storages, such as etcd and the Kubernetes apiserver.
type Registry interface {
	Get(ctx context.Context, key ResourceKey, obj client.Object) error
	ListStages(ctx context.Context) ([]StageInfo, error)
	List(ctx context.Context, key ResourceKey, obj client.ObjectList) error

	// Watch returns a channel for reading the resource metadata when it has been updated.
	Watch(ctx context.Context) <-chan *ResourceMetadata
}
