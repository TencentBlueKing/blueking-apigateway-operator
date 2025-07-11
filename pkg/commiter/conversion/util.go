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

// Package conversion ...
package conversion

import (
	"fmt"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
)

const (
	// gateway.stage.ID
	resourceIDFormat = "%s.%s.%s"
)

func getObjectName(name, ns string) string {
	return name
}

func (c *Converter) getID(id, objectName string) string {
	identifier := ""
	if len(id) != 0 && id != "<nil>" {
		identifier = id
	} else {
		identifier = objectName
	}
	return fmt.Sprintf(resourceIDFormat, c.gatewayName, c.stageName, identifier)
}

func (c *Converter) getLabel() map[string]string {
	return map[string]string{
		config.BKAPIGatewayLabelKeyGatewayName:  c.gatewayName,
		config.BKAPIGatewayLabelKeyGatewayStage: c.stageName,
	}
}

func (c *Converter) getOptionalUriGatewayPrefix() string {
	if useUriGatewayPrefix {
		return fmt.Sprintf(
			"%s/%s/%s",
			gatewayResourceBasePrefix,
			c.gatewayName,
			c.stageName,
		)
	}
	return ""
}
