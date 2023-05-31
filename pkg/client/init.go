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

package client

import (
	"fmt"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
)

var (
	serverAddr     string
	serverBindPort int = 6004
)

// Init ...
func Init(cfg *config.Config) {
	switch {
	case cfg.HttpServer.BindAddress != "":
		serverAddr = fmt.Sprintf(
			"%s:%d",
			cfg.HttpServer.BindAddress,
			cfg.HttpServer.BindPort,
		)
	case cfg.HttpServer.BindAddressV6 != "":
		serverAddr = fmt.Sprintf(
			"%s:%d",
			cfg.HttpServer.BindAddressV6,
			cfg.HttpServer.BindPort,
		)
	default:
		serverAddr = fmt.Sprintf("127.0.0.1:%d", cfg.HttpServer.BindPort)
	}

	serverBindPort = cfg.HttpServer.BindPort
}
