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

// Package biz ...
package biz

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

// GetApigwResourcesByStage 根据网关查询资源
func GetApigwResourcesByStage(
	ctx context.Context,
	commiter *commiter.Commiter,
	gatewayName string,
	stageName string,
	isExcludeReleaseVersion bool,
) (*apisix.ApisixConfiguration, error) {
	si := registry.StageInfo{
		GatewayName: gatewayName,
		StageName:   stageName,
	}
	apiSixResources, _, err := commiter.ConvertEtcdKVToApisixConfiguration(ctx, si)
	if err != nil {
		return nil, err
	}
	if isExcludeReleaseVersion {
		// 资源列表中排除 apigw-builtin-mock-release-version
		resourceIDKey := genResourceIDKey(gatewayName, stageName, config.ReleaseVersionResourceID)
		if _, ok := apiSixResources.Routes[resourceIDKey]; ok {
			delete(apiSixResources.Routes, resourceIDKey)
		}
	}
	return apiSixResources, nil
}

// GetApigwResourceCount 获取 apigw 指定环境的资源数量
func GetApigwResourceCount(
	ctx context.Context,
	commiter *commiter.Commiter,
	gatewayName string,
	stageName string,
) (int64, error) {
	si := registry.StageInfo{
		GatewayName: gatewayName,
		StageName:   stageName,
	}
	count, err := commiter.CliGetResourceCount(ctx, si)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ListApigwResources 获取 apigw 指定环境的资源列表
func ListApigwResources(
	ctx context.Context,
	commiter *commiter.Commiter,
	gatewayName string,
	stageName string,
) (map[string]*apisix.ApisixConfiguration, error) {
	configMap := make(map[string]*apisix.ApisixConfiguration)
	stageKey := config.GenStagePrimaryKey(gatewayName, stageName)
	apiSixResources, err := GetApigwResourcesByStage(ctx, commiter, gatewayName, stageName, true)
	if err != nil {
		return nil, err
	}
	configMap[stageKey] = apiSixResources
	return configMap, nil
}

// GetApigwResource 获取 apigw 指定环境下的资源信息
func GetApigwResource(
	ctx context.Context,
	commiter *commiter.Commiter,
	gatewayName string,
	stageName string,
	resourceName string,
	resourceID int64,
) (map[string]*apisix.ApisixConfiguration, error) {
	configMap := make(map[string]*apisix.ApisixConfiguration)
	stageKey := config.GenStagePrimaryKey(gatewayName, stageName)

	// by resourceName
	if resourceName != "" {
		si := registry.StageInfo{
			GatewayName: gatewayName,
			StageName:   stageName,
		}
		resourceNameKey := genResourceNameKey(gatewayName, stageName, resourceName)
		apiSixResources, _, err := commiter.CliConvertEtcdResourceToApisixConfiguration(ctx, si, resourceNameKey)
		if err != nil {
			return nil, err
		}
		configMap[stageKey] = apiSixResources
		return configMap, nil
	}

	// by resourceID
	apiSixResources, err := GetApigwResourcesByStage(ctx, commiter, gatewayName, stageName, true)
	if err != nil {
		return nil, err
	}
	resourceIDKey := genResourceIDKey(gatewayName, stageName, resourceID)
	for _, route := range apiSixResources.Routes {
		if resourceID != 0 && route.ID == resourceIDKey {
			apiSixResources.Routes = map[string]*apisix.Route{route.ID: route}
			configMap[stageKey] = apiSixResources
			return configMap, nil
		}
	}
	return configMap, nil
}

// GetApigwStageCurrentVersionInfo 获取 apigw 指定环境的发布版本信息
func GetApigwStageCurrentVersionInfo(
	ctx context.Context,
	commiter *commiter.Commiter,
	gatewayName string,
	stageName string,
) (map[string]interface{}, error) {
	si := registry.StageInfo{
		GatewayName: gatewayName,
		StageName:   stageName,
	}

	resourceNameKey := genResourceNameKey(gatewayName, stageName, "apigw-builtin-mock-release-version")
	apiSixResources, _, err := commiter.CliConvertEtcdResourceToApisixConfiguration(ctx, si, resourceNameKey)
	if err != nil {
		return nil, err
	}

	if len(apiSixResources.Routes) == 0 {
		return nil, errors.New("current-version not found")
	}

	resourceIDKey := genResourceIDKey(gatewayName, stageName, config.ReleaseVersionResourceID)
	plugins := apiSixResources.Routes[resourceIDKey].Plugins

	for _, plugin := range plugins {
		pluginData := plugin.(map[string]interface{})
		responseExample := pluginData["response_example"].(string)
		if responseExample == "" {
			continue
		}
		versionInfo := make(map[string]interface{})
		err := json.Unmarshal([]byte(responseExample), &versionInfo)
		if err != nil {
			return nil, errors.New("current-version unmarshal error: " + err.Error())
		}

		return versionInfo, nil
	}

	return nil, errors.New("current-version not found")
}
