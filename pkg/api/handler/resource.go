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

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

// ResourceHandler resource api handler
type ResourceHandler struct {
	LeaderElector   leaderelection.LeaderElector
	registry        registry.Registry
	committer       *commiter.Commiter
	apisixConfStore synchronizer.ApisixConfigStore
}

// NewResourceApi constructor of resource handler
func NewResourceApi(
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore,
) *ResourceHandler {
	return &ResourceHandler{
		LeaderElector:   leaderElector,
		registry:        registry,
		committer:       committer,
		apisixConfStore: apiSixConfStore,
	}
}

// GetLeader get leader pod host
func (r *ResourceHandler) GetLeader(c *gin.Context) {
	if r.LeaderElector == nil {
		utils.BaseErrorJSONResponse(c, utils.NotFoundError, "LeaderElector not found", http.StatusOK)
		return
	}
	utils.SuccessJSONResponse(c, r.LeaderElector.Leader())
}

// Sync ...
func (r *ResourceHandler) Sync(c *gin.Context) {
	var req SyncReq
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestErrorJSONResponse(c, utils.ValidationErrorMessage(err))
		return
	}

	if !req.All {
		r.committer.ForceCommit(c, []registry.StageInfo{
			{
				GatewayName: req.Gateway,
				StageName:   req.Stage,
				Ctx:         context.Background(),
			},
		})
	}

	stageList, err := r.registry.ListStages(c)
	if err != nil {
		utils.BaseErrorJSONResponse(c, utils.SystemError,
			fmt.Errorf("registry list stages err:%w", err).Error(), http.StatusOK)
		return
	}
	r.committer.ForceCommit(c, stageList)
	utils.SuccessJSONResponse(c, "ok")
}

// Diff between bkgateway resources and apisix storage
func (r *ResourceHandler) Diff(c *gin.Context) {
	var req DiffReq
	if err := c.ShouldBind(&req); err != nil {
		utils.BadRequestErrorJSONResponse(c, utils.ValidationErrorMessage(err))
		return
	}
	diff, err := r.diffHandler(c, &req)
	if err != nil {
		utils.BaseErrorJSONResponse(c, utils.SystemError, fmt.Sprintf("diff fail: %+v", err), http.StatusOK)
		return
	}
	utils.SuccessJSONResponse(c, diff)
}

// List resources in apisix
func (r *ResourceHandler) List(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBind(&req); err != nil {
		utils.BadRequestErrorJSONResponse(c, utils.ValidationErrorMessage(err))
		return
	}
	list, err := r.listHandler(c, &req)
	if err != nil {
		utils.BaseErrorJSONResponse(c, utils.SystemError, fmt.Sprintf("list err:%+v", err.Error()), http.StatusOK)
		return
	}
	utils.SuccessJSONResponse(c, list)
}

// diffHandler handle diff resource between gateway and apiSix
func (r *ResourceHandler) diffHandler(ctx context.Context, req *DiffReq) (DiffInfo, error) {
	resp := make(DiffInfo)
	var err error
	if !req.All {
		si := registry.StageInfo{
			GatewayName: req.Gateway,
			StageName:   req.Stage,
			Ctx:         context.Background(),
		}
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		originalApiSixResources := r.apisixConfStore.Get(stageKey)
		apiSixResources, _, err := r.committer.ConvertEtcdKVToApisixConfiguration(ctx, si)
		if err != nil {
			return nil, err
		}
		resourceKey, err := r.getRouteIDByResourceIdentity(
			originalApiSixResources,
			req.Gateway,
			req.Stage,
			req.Resource,
		)
		if err != nil {
			resourceKey, err = r.getRouteIDByResourceIdentity(apiSixResources, req.Gateway, req.Stage, req.Resource)
			if err != nil {
				return nil, err
			}
		}
		resp[stageKey] = r.diffWithRouteID(originalApiSixResources, apiSixResources, resourceKey)
		return resp, nil
	}

	stageList, err := r.registry.ListStages(ctx)
	if err != nil {
		return nil, err
	}
	allApiSixResources := r.apisixConfStore.GetAll()
	for _, stage := range stageList {
		stageKey := config.GenStagePrimaryKey(stage.GatewayName, stage.StageName)
		apiSixResources, _, itemErr := r.committer.ConvertEtcdKVToApisixConfiguration(ctx, stage)
		if itemErr != nil {
			err = fmt.Errorf("%s [stage %s failed: %w]", stage.StageName, stageKey, err)
			continue
		}
		diffResult := r.diff(allApiSixResources[stageKey], apiSixResources)
		if !r.isDiffResultEmpty(diffResult) {
			resp[stageKey] = diffResult
		}
	}
	return resp, nil
}

// listHandler handle list resource from apisix
func (r *ResourceHandler) listHandler(ctx context.Context, req *ListReq) (ListInfo, error) {
	resp := make(ListInfo)
	if !req.All {
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		apiSixRes := r.apisixConfStore.Get(stageKey)
		if req.Resource != nil {
			resourceKey, err := r.getRouteIDByResourceIdentity(apiSixRes, req.Gateway, req.Stage, req.Resource)
			if err != nil {
				return nil, err
			}
			apiSixRes.Routes = map[string]*apisix.Route{
				resourceKey: apiSixRes.Routes[resourceKey],
			}
		}
		stagedApiSixRes := map[string]interface{}{
			stageKey: apiSixRes,
		}
		by, err := json.Marshal(stagedApiSixRes)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(by, &resp)
		return resp, nil
	}
	stagedApiSixRes := r.apisixConfStore.GetAll()
	by, err := json.Marshal(stagedApiSixRes)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(by, &resp)
	return resp, nil
}

func (r *ResourceHandler) diffWithRouteID(
	lhs, rhs *apisix.ApisixConfiguration,
	routeID string,
) *StageScopedApisixResources {
	ret := &StageScopedApisixResources{
		Routes:         r.diffMap(lhs.Routes, rhs.Routes, routeID),
		Services:       r.diffMap(lhs.Services, rhs.Services, ""),
		PluginMetadata: r.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, ""),
		Ssl:            r.diffMap(lhs.SSLs, rhs.SSLs, ""),
	}
	return ret
}

func (r *ResourceHandler) diff(lhs, rhs *apisix.ApisixConfiguration) *StageScopedApisixResources {
	ret := &StageScopedApisixResources{
		Routes:         r.diffMap(lhs.Routes, rhs.Routes, ""),
		Services:       r.diffMap(lhs.Services, rhs.Services, ""),
		PluginMetadata: r.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, ""),
		Ssl:            r.diffMap(lhs.SSLs, rhs.SSLs, ""),
	}
	return ret
}

func (r *ResourceHandler) diffMap(lhs, rhs interface{}, id string) map[string]interface{} {
	lhsValue := reflect.ValueOf(lhs)
	rhsValue := reflect.ValueOf(rhs)
	diff := make(map[string]interface{})
	if lhsValue.Kind() != reflect.Map || rhsValue.Kind() != reflect.Map {
		return diff
	}
	lhsIter := lhsValue.MapRange()
	for lhsIter.Next() {
		key := lhsIter.Key()
		if len(id) != 0 && key.String() != id {
			rhsValue.SetMapIndex(key, reflect.Value{})
			continue
		}
		val := lhsIter.Value()
		rhsItemValue := rhsValue.MapIndex(key)
		if !rhsItemValue.IsValid() {
			diff[key.String()] = cmp.Diff(val.Interface(), nil)
		} else {
			diffStr := cmp.Diff(val.Interface(), rhsItemValue.Interface())
			if len(diffStr) != 0 {
				diff[key.String()] = diffStr
			}
			rhsValue.SetMapIndex(key, reflect.Value{})
		}
	}
	rhsIter := rhsValue.MapRange()
	for rhsIter.Next() {
		key := rhsIter.Key()
		if len(id) != 0 && key.String() != id {
			continue
		}
		val := rhsIter.Value()
		diff[key.String()] = cmp.Diff(nil, val.Interface())
	}
	return diff
}

func (r *ResourceHandler) getRouteIDByResourceIdentity(
	conf *apisix.ApisixConfiguration,
	gateway, stage string,
	resource *ResourceInfo,
) (string, error) {
	if conf == nil || resource == nil {
		return "", errors.New("resource id not found")
	}
	if resource.ResourceId != 0 {
		return fmt.Sprintf("%s.%s.%d", gateway, stage, resource.ResourceId), nil
	}
	// fixme: labels的resource name 超过64不能被apisix从etcd读取, 从源头没有写入 pkg/conversion/resource_slz.go:62
	for _, route := range conf.Routes {
		if route.Metadata.Labels[config.BKAPIGatewayLabelKeyResourceName] == resource.ResourceName {
			return route.ID, nil
		}
	}
	return "", errors.New("resource id not found")
}

func (r *ResourceHandler) isDiffResultEmpty(result *StageScopedApisixResources) bool {
	if result == nil {
		return true
	}
	return len(result.Routes) == 0 && len(result.Services) == 0 && len(result.Ssl) == 0 &&
		len(result.PluginMetadata) == 0
}
