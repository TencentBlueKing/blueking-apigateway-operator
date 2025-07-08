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

package types

import (
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	gatewayv1beta1 "github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
)

// KindMapping ...
var KindMapping map[string]string = map[string]string{
	"BkGatewayService":  "bkgwsvc",
	"BkGatewayResource": "bkgwres",
}

// AbbObjectFactoryMapping ...
var AbbObjectFactoryMapping map[string]func() ctrl.Object = map[string]func() ctrl.Object{
	"bkgwsvc": func() ctrl.Object { return &gatewayv1beta1.BkGatewayService{} },
	"bkgwres": func() ctrl.Object { return &gatewayv1beta1.BkGatewayResource{} },
}
