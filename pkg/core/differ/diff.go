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

// Package differ ...
package differ

import (
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	json "github.com/json-iterator/go"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
)

type ConfigDiffer struct{}

// NewConfigDiffer creates and returns a new instance of ConfigDiffer
// It serves as a constructor function for the ConfigDiffer struct
func NewConfigDiffer() *ConfigDiffer {
	// Return a new instance of ConfigDiffer
	return &ConfigDiffer{}
}

// transformMap: 需要单独针对于map类型添加一个对比转换器，由于value是一个interface类型,对于不同的序列化方式会存在类型不一致
// eg： value存在map[any]any和map[string]any和map[interface]any的问题
func transformMap(mapType map[string]interface{}) map[string]interface{} {
	mapTypeJson, _ := json.Marshal(mapType)
	var newMap map[string]interface{}
	_ = json.Unmarshal(mapTypeJson, &newMap)
	return newMap
}

// ignoreApisixMetadata: 忽略apisixeMetadata的部分成员
var ignoreApisixMetadataCmpOpt = cmpopts.IgnoreFields(apisixv1.Metadata{}, "Desc", "Labels")

// ignoreCreateTimeAndUpdateTimeCmpOpt: 忽略typ 创建、更新时间
var ignoreCreateTimeAndUpdateTimeCmpOptFunc = func(typ interface{}) cmp.Option {
	return cmpopts.IgnoreFields(typ, "CreateTime", "UpdateTime")
}

// CmpReporter ...
type CmpReporter struct {
	Gateway      string
	Stage        string
	ResourceType string
	CmpReported  bool
	DiffReported bool
}

// PushStep ...
func (r *CmpReporter) PushStep(ps cmp.PathStep) {
}

// PopStep ...
func (r *CmpReporter) PopStep() {
}

// Report ...
func (r *CmpReporter) Report(rs cmp.Result) {
	// report sync cmp metric
	if !r.CmpReported {
		metric.ReportSyncCmpMetric(
			r.Gateway,
			r.Stage,
			r.ResourceType,
		)
		r.CmpReported = true
	}

	// report sync cmp diff  metric
	if !rs.Equal() && !r.DiffReported {
		metric.ReportSyncCmpDiffMetric(
			r.Gateway,
			r.Stage,
			r.ResourceType,
		)
		r.DiffReported = true
	}
}

func (d *ConfigDiffer) Diff(
	old, new *entity.ApisixStageResource,
) (put *entity.ApisixStageResource, delete *entity.ApisixStageResource) {
	if old == nil {
		return new, nil
	}
	if new == nil {
		return nil, old
	}
	put = &entity.ApisixStageResource{}
	delete = &entity.ApisixStageResource{}
	put.Routes, delete.Routes = d.DiffRoutes(old.Routes, new.Routes)
	put.Services, delete.Services = d.DiffServices(old.Services, new.Services)
	put.SSLs, delete.SSLs = d.DiffSSLs(old.SSLs, new.SSLs)
	return put, delete
}

func (d *ConfigDiffer) DiffRoutes(
	old map[string]*entity.Route,
	new map[string]*entity.Route,
) (putList map[string]*entity.Route, deleteList map[string]*entity.Route) {
	oldResMap := make(map[string]*entity.Route)
	putList = make(map[string]*entity.Route)
	deleteList = make(map[string]*entity.Route)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}

		if !cmp.Equal(
			oldRes,
			newRes,
			cmp.Transformer("transformerMap", transformMap),
			ignoreApisixMetadataCmpOpt,
			ignoreCreateTimeAndUpdateTimeCmpOptFunc(entity.Route{}),
			cmp.Reporter(&CmpReporter{
				Gateway:      newRes.GetReleaseInfo().GetGatewayName(),
				Stage:        newRes.GetReleaseInfo().GetStageName(),
				ResourceType: constant.ApisixResourceTypeRoutes,
			}),
		) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}

func (d *ConfigDiffer) DiffServices(
	old map[string]*entity.Service,
	new map[string]*entity.Service,
) (putList map[string]*entity.Service, deleteList map[string]*entity.Service) {
	oldResMap := make(map[string]*entity.Service)
	putList = make(map[string]*entity.Service)
	deleteList = make(map[string]*entity.Service)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}
		if !cmp.Equal(
			oldRes,
			newRes,
			cmp.Transformer("transformerMap", transformMap),
			ignoreApisixMetadataCmpOpt,
			ignoreCreateTimeAndUpdateTimeCmpOptFunc(entity.Service{}),
			cmp.Reporter(&CmpReporter{
				Gateway:      newRes.GetReleaseInfo().GetGatewayName(),
				Stage:        newRes.GetReleaseInfo().GetStageName(),
				ResourceType: constant.ApisixResourceTypeServices,
			}),
		) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}

func (d *ConfigDiffer) DiffPluginMetadatas(
	old map[string]*entity.PluginMetadata,
	new map[string]*entity.PluginMetadata,
) (putList map[string]*entity.PluginMetadata, deleteList map[string]*entity.PluginMetadata) {
	oldResMap := make(map[string]*entity.PluginMetadata)
	putList = make(map[string]*entity.PluginMetadata)
	deleteList = make(map[string]*entity.PluginMetadata)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}
		if !cmp.Equal(
			oldRes,
			newRes,
			cmp.Transformer("transformerMap", transformMap),
		) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}

func (d *ConfigDiffer) DiffSSLs(
	old map[string]*entity.SSL,
	new map[string]*entity.SSL,
) (putList map[string]*entity.SSL, deleteList map[string]*entity.SSL) {
	oldResMap := make(map[string]*entity.SSL)
	putList = make(map[string]*entity.SSL)
	deleteList = make(map[string]*entity.SSL)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}
		if !cmp.Equal(oldRes, newRes,
			cmp.Transformer("transformerMap", transformMap),
			ignoreCreateTimeAndUpdateTimeCmpOptFunc(entity.SSL{}),
			cmp.Reporter(&CmpReporter{
				Gateway:      newRes.GetReleaseInfo().GetGatewayName(),
				Stage:        newRes.GetReleaseInfo().GetStageName(),
				ResourceType: constant.ApisixResourceTypeSSL,
			}),
		) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}
