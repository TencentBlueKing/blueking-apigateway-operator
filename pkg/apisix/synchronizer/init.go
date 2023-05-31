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

package synchronizer

import "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"

var (
	virtualGatewayName string = "-"
	virtualStageName   string = "-"

	fileLoggerLogPath               string = "/usr/local/apisix/logs/access.log"
	operatorExternalHost            string = "127.0.0.1"
	operatorExternalHealthProbePort int    = 8081
	extraApisixResourcesPath        string
)

// Init ...
func Init(cfg *config.Config) {
	virtualGatewayName = cfg.Apisix.VirtualStage.VirtualGateway
	virtualStageName = cfg.Apisix.VirtualStage.VirtualStage

	fileLoggerLogPath = cfg.Apisix.VirtualStage.FileLoggerLogPath
	operatorExternalHost = cfg.Apisix.VirtualStage.OperatorExternalHost
	operatorExternalHealthProbePort = cfg.Apisix.VirtualStage.OperatorExternalHealthProbePort
	extraApisixResourcesPath = cfg.Apisix.VirtualStage.ExtraApisixResources
}
