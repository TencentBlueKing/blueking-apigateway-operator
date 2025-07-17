package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"strings"
)

func genResourceIDKey(gatewayName, stageName string, resourceID int64) string {
	return fmt.Sprintf("%s.%s.%d", gatewayName, stageName, resourceID)
}

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
		resourceIDKey := genResourceIDKey(gatewayName, stageName, -1)
		if _, ok := apiSixResources.Routes[resourceIDKey]; ok {
			delete(apiSixResources.Routes, resourceIDKey)
		}
	}
	return apiSixResources, nil
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
	apiSixResources, err := GetApigwResourcesByStage(ctx, commiter, gatewayName, stageName, true)
	if err != nil {
		return nil, err
	}
	resourceNameKey := fmt.Sprintf(
		"%s-%s-%s",
		gatewayName,
		stageName,
		strings.Replace(resourceName, "_", "-", -1),
	)
	resourceIDKey := genResourceIDKey(gatewayName, stageName, resourceID)
	for _, route := range apiSixResources.Routes {
		if resourceName != "" && route.Name == resourceNameKey {
			apiSixResources.Routes = map[string]*apisix.Route{route.ID: route}
			configMap[stageKey] = apiSixResources
			return configMap, nil
		}
		if resourceID != 0 && route.ID == resourceIDKey {
			apiSixResources.Routes = map[string]*apisix.Route{route.ID: route}
			configMap[stageKey] = apiSixResources
			return configMap, nil
		}
	}
	return configMap, nil

}

// GetApigwStageCurrentVersion 获取 apigw 指定环境的发布版本
func GetApigwStageCurrentVersion(
	ctx context.Context,
	commiter *commiter.Commiter,
	gatewayName string,
	stageName string,
) (int64, error) {
	si := registry.StageInfo{
		GatewayName: gatewayName,
		StageName:   stageName,
	}
	apiSixResources, _, err := commiter.ConvertEtcdKVToApisixConfiguration(ctx, si)
	if err != nil {
		return 0, err
	}
	var exampleData struct {
		PublishID int64  `json:"publish_id"`
		StartTime string `json:"start_time"`
	}
	resourceIDKey := genResourceIDKey(gatewayName, stageName, -1)
	for _, route := range apiSixResources.Routes {
		if route.ID == resourceIDKey {
			for _, plugin := range route.Plugins {
				pluginData := plugin.(map[string]interface{})
				ResponseExample, ok := pluginData["response_example"].(string)
				if !ok {
					return 0, errors.New("response_example is empty")
				}
				if err := json.Unmarshal([]byte(ResponseExample), &exampleData); err != nil {
					return 0, err
				}
				return exampleData.PublishID, nil
			}
		}
	}
	return 0, errors.New("current-version not found")
}
