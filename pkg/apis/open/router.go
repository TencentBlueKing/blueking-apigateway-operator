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

// Package open ...
package open

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apis/open/handler"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// Register registers the API routes
func Register(
	r *gin.RouterGroup,
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore,
) {
	// register resource api
	resourceApi := handler.NewResourceApi(leaderElector, registry, committer, apiSixConfStore)
	r.GET("/leader/", resourceApi.GetLeader)
	r.POST("/resources/apigw/", resourceApi.ApigwList)
	r.POST("/resources/apigw/count/", resourceApi.ApigwStageResourceCount)
	r.POST("/resources/apigw/current-version/", resourceApi.ApigwStageCurrentVersion)
}
