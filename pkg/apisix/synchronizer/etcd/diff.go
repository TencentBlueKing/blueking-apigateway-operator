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

package etcd

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	json "github.com/json-iterator/go"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
)

type configDiffer struct{}

func newConfigDiffer() *configDiffer {
	return &configDiffer{}
}

// transformMap: 需要单独针对于map类型添加一个对比转换器，由于value是一个interface类型,对于不同的序列化方式会存在类型不一致
// eg： value存在map[any]any和map[string]any和map[interface]any的问题
func transformMap(mapType map[string]interface{}) map[string]interface{} {
	mapTypeJson, _ := json.Marshal(mapType)
	var newMap map[string]interface{}
	_ = json.Unmarshal(mapTypeJson, &newMap)
	return newMap
}

func (d *configDiffer) diff(
	old, new *apisix.ApisixConfiguration,
) (put *apisix.ApisixConfiguration, delete *apisix.ApisixConfiguration) {
	if old == nil {
		return new, nil
	}
	if new == nil {
		return nil, old
	}
	put = &apisix.ApisixConfiguration{}
	delete = &apisix.ApisixConfiguration{}
	put.Routes, delete.Routes = d.diffRoutes(old.Routes, new.Routes)
	put.Services, delete.Services = d.diffServices(old.Services, new.Services)
	put.PluginMetadatas, delete.PluginMetadatas = d.diffPluginMetadatas(old.PluginMetadatas, new.PluginMetadatas)
	put.SSLs, delete.SSLs = d.diffSSLs(old.SSLs, new.SSLs)
	return put, delete
}

func (d *configDiffer) diffRoutes(
	old map[string]*apisix.Route,
	new map[string]*apisix.Route,
) (putList map[string]*apisix.Route, deleteList map[string]*apisix.Route) {
	oldResMap := make(map[string]*apisix.Route)
	putList = make(map[string]*apisix.Route)
	deleteList = make(map[string]*apisix.Route)
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
			cmpopts.IgnoreFields(apisix.Route{}, "CreateTime", "UpdateTime")) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}

func (d *configDiffer) diffServices(
	old map[string]*apisix.Service,
	new map[string]*apisix.Service,
) (putList map[string]*apisix.Service, deleteList map[string]*apisix.Service) {
	oldResMap := make(map[string]*apisix.Service)
	putList = make(map[string]*apisix.Service)
	deleteList = make(map[string]*apisix.Service)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}
		if !cmp.Equal(oldRes, newRes, cmpopts.IgnoreFields(apisix.Service{}, "CreateTime", "UpdateTime")) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}

func (d *configDiffer) diffPluginMetadatas(
	old map[string]*apisix.PluginMetadata,
	new map[string]*apisix.PluginMetadata,
) (putList map[string]*apisix.PluginMetadata, deleteList map[string]*apisix.PluginMetadata) {
	oldResMap := make(map[string]*apisix.PluginMetadata)
	putList = make(map[string]*apisix.PluginMetadata)
	deleteList = make(map[string]*apisix.PluginMetadata)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}
		if !cmp.Equal(oldRes, newRes) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}

func (d *configDiffer) diffSSLs(
	old map[string]*apisix.SSL,
	new map[string]*apisix.SSL,
) (putList map[string]*apisix.SSL, deleteList map[string]*apisix.SSL) {
	oldResMap := make(map[string]*apisix.SSL)
	putList = make(map[string]*apisix.SSL)
	deleteList = make(map[string]*apisix.SSL)
	for key, oldRes := range old {
		oldResMap[key] = oldRes
	}
	for key, newRes := range new {
		oldRes, ok := oldResMap[key]
		if !ok {
			putList[key] = newRes
			continue
		}
		if !cmp.Equal(oldRes, newRes, cmpopts.IgnoreFields(apisix.SSL{}, "CreateTime", "UpdateTime")) {
			putList[key] = newRes
		}
		delete(oldResMap, key)
	}
	for key, oldRes := range oldResMap {
		deleteList[key] = oldRes
	}
	return putList, deleteList
}
