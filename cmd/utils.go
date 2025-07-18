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

// Package cmd ...
package cmd

import (
	"fmt"

	json "github.com/json-iterator/go"
	yaml "gopkg.in/yaml.v3"
)

func printJson(i interface{}) error {
	by, err := json.Marshal(i)
	if err != nil {
		return err
	}
	fmt.Println(string(by))
	return nil
}

func printYaml(i interface{}) error {
	by, err := json.Marshal(i)
	if err != nil {
		return err
	}
	tmp := make(map[string]interface{})
	if err := json.Unmarshal(by, &tmp); err != nil {
		return err
	}
	by, err = yaml.Marshal(tmp)
	if err != nil {
		return err
	}
	fmt.Println(string(by))
	return nil
}
