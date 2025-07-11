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

// Package utils contains some common utils
package utils

import (
	json "github.com/json-iterator/go"
	"k8s.io/apimachinery/pkg/runtime"
)

// RawExtension2Map convert k8s.io/apimachinery/pkg/runtime.RawExtension to map[string]interface{}
func RawExtension2Map(conf runtime.RawExtension) (map[string]interface{}, error) {
	retMap := make(map[string]interface{})
	err := json.Unmarshal(conf.Raw, &retMap)
	if err != nil {
		return nil, err
	}
	return retMap, nil
}

// Map2RawExtension convert map[string]interface{} to k8s.io/apimachinery/pkg/runtime.RawExtension
func Map2RawExtension(conf map[string]interface{}) (runtime.RawExtension, error) {
	raw := runtime.RawExtension{}
	by, err := json.Marshal(conf)
	if err != nil {
		return raw, err
	}
	raw.UnmarshalJSON(by)
	return raw, nil
}
