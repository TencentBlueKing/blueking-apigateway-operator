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

package api

import (
	"net/http"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/protocol"
	"github.com/TencentBlueKing/blueking-apigateway-operator/internal/tracer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/service"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"

	"github.com/gin-gonic/gin"
)

type ResourceApi struct {
	LeaderElector   leaderelection.LeaderElector
	registry        registry.Registry
	committer       *commiter.Commiter
	apiSixConfStore synchronizer.ApisixConfigStore
	resourceSvc     *service.ResourceService
}

func NewResourceApi(
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore) *ResourceApi {
	return &ResourceApi{
		LeaderElector:   leaderElector,
		registry:        registry,
		committer:       committer,
		apiSixConfStore: apiSixConfStore,
		resourceSvc:     service.NewResourceService(registry, committer, apiSixConfStore),
	}
}

// GetLeader ...
func (r *ResourceApi) GetLeader(ctx *gin.Context) {
	_, span := tracer.NewTracer("grpcServer").Start(ctx, "grpcServer/getLeader")
	defer span.End()
	if r.LeaderElector == nil {
		span.AddEvent("LeaderElector not found")
		utils.CommonErrorJSONResponse(ctx, utils.NotFoundError, "LeaderElector not found")
		return
	}
	utils.SuccessJSONResponse(ctx, r.LeaderElector.Leader())
}

// Sync ...
func (r *ResourceApi) Sync(ctx *gin.Context) {
	var req protocol.SyncReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestErrorJSONResponse(ctx, utils.ValidationErrorMessage(err))
		return
	}
	err := r.resourceSvc.Sync(ctx, req)
	if err != nil {
		utils.CommonErrorJSONResponse(ctx, utils.SystemError, err.Error())
		return
	}
	utils.SuccessJSONResponse(ctx, "ok")
}

// Diff ...
func (r *ResourceApi) Diff(ctx *gin.Context) {
	var req protocol.DiffReq
	if err := ctx.ShouldBind(&req); err != nil {
		utils.BadRequestErrorJSONResponse(ctx, utils.ValidationErrorMessage(err))
		return
	}
	diff, err := r.resourceSvc.Diff(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"data":    diff,
			"message": err.Error(),
		})
		return
	}
	utils.SuccessJSONResponse(ctx, diff)
}

// List ...
func (r *ResourceApi) List(ctx *gin.Context) {
	var req protocol.ListReq
	if err := ctx.ShouldBind(&req); err != nil {
		utils.BadRequestErrorJSONResponse(ctx, utils.ValidationErrorMessage(err))
		return
	}
	list, err := r.resourceSvc.List(ctx, &req)
	if err != nil {
		utils.CommonErrorJSONResponse(ctx, utils.SystemError, err.Error())
		return
	}
	utils.SuccessJSONResponse(ctx, list)
}
