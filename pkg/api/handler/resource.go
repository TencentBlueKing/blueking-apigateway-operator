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
	"fmt"
	"reflect"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/leaderelection"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
	"github.com/google/go-cmp/cmp"
	"github.com/rotisserie/eris"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	LeaderElector   leaderelection.LeaderElector
	registry        registry.Registry
	committer       *commiter.Commiter
	apiSixConfStore synchronizer.ApisixConfigStore
}

func NewResourceApi(
	leaderElector leaderelection.LeaderElector,
	registry registry.Registry,
	committer *commiter.Commiter,
	apiSixConfStore synchronizer.ApisixConfigStore) *ResourceHandler {
	return &ResourceHandler{
		LeaderElector:   leaderElector,
		registry:        registry,
		committer:       committer,
		apiSixConfStore: apiSixConfStore,
	}
}

// GetLeader ...
func (r *ResourceHandler) GetLeader(c *gin.Context) {
	if r.LeaderElector == nil {
		utils.CommonErrorJSONResponse(c, utils.NotFoundError, "LeaderElector not found")
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
			},
		})
	}
	stageList, err := r.registry.ListStages(c)
	if err != nil {
		utils.CommonErrorJSONResponse(c, utils.SystemError, fmt.Errorf("registry list stages err:%w", err).Error())
		return
	}
	r.committer.ForceCommit(c, stageList)
	utils.SuccessJSONResponse(c, "ok")
}

// Diff ...
func (r *ResourceHandler) Diff(c *gin.Context) {
	var req DiffReq
	if err := c.ShouldBind(&req); err != nil {
		utils.BadRequestErrorJSONResponse(c, utils.ValidationErrorMessage(err))
		return
	}
	diff, err := r.DiffHandler(c, &req)
	if err != nil {
		utils.CommonErrorJSONResponse(c, utils.SystemError, fmt.Sprintf("diff fail: %+v", err))
		return
	}
	utils.SuccessJSONResponse(c, diff)
}

// List ...
func (r *ResourceHandler) List(c *gin.Context) {
	var req ListReq
	if err := c.ShouldBind(&req); err != nil {
		utils.BadRequestErrorJSONResponse(c, utils.ValidationErrorMessage(err))
		return
	}
	list, err := r.ListHandler(c, &req)
	if err != nil {
		utils.CommonErrorJSONResponse(c, utils.SystemError, fmt.Sprintf("list err:%+v", err.Error()))
		return
	}
	utils.SuccessJSONResponse(c, list)
}

// SyncHandler handle sys resource between gateway and apiSix
func (r *ResourceHandler) SyncHandler(ctx context.Context, req SyncReq) error {
	if !req.All {
		r.committer.ForceCommit(ctx, []registry.StageInfo{
			{
				GatewayName: req.Gateway,
				StageName:   req.Stage,
			},
		})
		return nil
	}
	stageList, err := r.registry.ListStages(ctx)
	if err != nil {
		return err
	}
	r.committer.ForceCommit(ctx, stageList)
	return nil
}

// DiffHandler handle diff resource between gateway and apiSix
func (r *ResourceHandler) DiffHandler(ctx context.Context, req *DiffReq) (DiffInfo, error) {
	resp := make(DiffInfo)
	var err error
	if !req.All {
		si := registry.StageInfo{
			GatewayName: req.Gateway,
			StageName:   req.Stage,
		}
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		originalApiSixResources := r.apiSixConfStore.Get(stageKey)
		apiSixResources, err := r.committer.ConvertEtcdKVToApisixConfiguration(ctx, si)
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
	allApiSixResources := r.apiSixConfStore.GetAll()
	for _, stage := range stageList {
		stageKey := config.GenStagePrimaryKey(stage.GatewayName, stage.StageName)
		apiSixResources, itemErr := r.committer.ConvertEtcdKVToApisixConfiguration(ctx, stage)
		if itemErr != nil {
			err = fmt.Errorf("%s [stage %s failed: %s]", stage.StageName, stageKey, eris.ToString(err, true))
			continue
		}
		diffResult := r.diff(allApiSixResources[stageKey], apiSixResources)
		if !r.isDiffResultEmpty(diffResult) {
			resp[stageKey] = diffResult
		}
	}
	return resp, nil
}

// ListHandler handle list resource from gateway and k8s
func (r *ResourceHandler) ListHandler(ctx context.Context, req *ListReq) (ListInfo, error) {
	resp := make(ListInfo)
	if !req.All {
		stageKey := config.GenStagePrimaryKey(req.Gateway, req.Stage)
		apiSixRes := r.apiSixConfStore.Get(stageKey)
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
	stagedApiSixRes := r.apiSixConfStore.GetAll()
	by, err := json.Marshal(stagedApiSixRes)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(by, &resp)
	return resp, nil
}

func (r *ResourceHandler) diffWithRouteID(
	lhs, rhs *apisix.ApisixConfiguration,
	routeID string) *StageScopedApiSixResources {
	ret := &StageScopedApiSixResources{
		Routes:         r.diffMap(lhs.Routes, rhs.Routes, routeID),
		Services:       r.diffMap(lhs.Services, rhs.Services, ""),
		PluginMetadata: r.diffMap(lhs.PluginMetadatas, rhs.PluginMetadatas, ""),
		Ssl:            r.diffMap(lhs.SSLs, rhs.SSLs, ""),
	}
	return ret
}

func (r *ResourceHandler) diff(lhs, rhs *apisix.ApisixConfiguration) *StageScopedApiSixResources {
	ret := &StageScopedApiSixResources{
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
		return "", eris.New("resource id not found")
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
	return "", eris.New("resource id not found")
}

func (r *ResourceHandler) isDiffResultEmpty(result *StageScopedApiSixResources) bool {
	if result == nil {
		return true
	}
	return len(result.Routes) == 0 && len(result.Services) == 0 && len(result.Ssl) == 0 &&
		len(result.PluginMetadata) == 0
}
