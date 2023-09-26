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

package conversion

import (
	"context"

	json "github.com/json-iterator/go"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter/cert"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/commiter/service"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
)

// UpstreamConfig ...
type UpstreamConfig struct {
	CertDetectTree radixtree.RadixTree

	InternalDiscoveryPlugins []string
	NodeDiscoverer           service.NodeDiscoverer
	ExternalNodeDiscoverer   service.ExternalNodeDiscoverer
}

// SSLConfig ...
type SSLConfig struct {
	CertFetcher cert.TLSCertFetcher
}

// Converter converter for apisix configuration
type Converter struct {
	namespace   string
	gatewayName string
	stageName   string
	stage       *v1beta1.BkGatewayStage

	upstreamConfig *UpstreamConfig
	sslConfig      *SSLConfig

	logger *zap.SugaredLogger
}

// NewConverter create converter
func NewConverter(
	namespace, gatewayName string,
	stage *v1beta1.BkGatewayStage,
	upstreamConfig *UpstreamConfig,
	sslConfig *SSLConfig,
) (*Converter, error) {
	stageName := stage.Spec.Name
	if len(stageName) == 0 {
		stageName = stage.Labels[config.BKAPIGatewayLabelKeyGatewayStage]
	}
	if len(stageName) == 0 {
		return nil, eris.New("Build converter failed: no stage name")
	}
	return &Converter{
		namespace:      namespace,
		gatewayName:    gatewayName,
		stageName:      stageName,
		stage:          stage,
		upstreamConfig: upstreamConfig,
		sslConfig:      sslConfig,
		logger:         logging.GetLogger().Named("converter"),
	}, nil
}

// Convert ...
func (c *Converter) Convert(
	ctx context.Context,
	resources []*v1beta1.BkGatewayResource,
	services []*v1beta1.BkGatewayService,
	ssls []*v1beta1.BkGatewayTLS,
	pluginMetadatas []*v1beta1.BkGatewayPluginMetadata,
) (*apisix.ApisixConfiguration, error) {
	if c.stage == nil {
		return nil, eris.New("no stage defined")
	}

	apisixConfig := apisix.NewEmptyApisixConfiguration()
	for _, res := range resources {
		// 如果publish_id为-1，则跳过版本探测路由(id=-1)的写入
		if res.Spec.ID.String() == constant.ApisixVersionRouteID &&
			c.stage.Labels[config.BKAPIGatewayLabelKeyGatewayPublishID] == constant.NoNeedReportPublishID {
			continue
		}
		route, err := c.convertResource(res, services)
		if err != nil {
			return nil, eris.Wrapf(err, "convert resource failed")
		}
		apisixConfig.Routes[route.GetID()] = route
	}

	for _, svc := range services {
		svc, err := c.convertService(svc)
		if err != nil {
			return nil, eris.Wrapf(err, "convert service failed")
		}
		apisixConfig.Services[svc.GetID()] = svc
	}

	for _, ssl := range ssls {
		apisixSSL, err := c.convertSSL(ctx, ssl)
		if err != nil {
			return nil, eris.Wrapf(err, "convert service failed")
		}
		apisixConfig.SSLs[apisixSSL.GetID()] = apisixSSL
	}

	for _, metadata := range pluginMetadatas {
		var obj map[string]interface{}
		err := json.Unmarshal(metadata.Spec.Config.Raw, &obj)
		if err != nil {
			return nil, eris.Wrapf(err, "convert plugin metadata failed")
		}
		name, err := c.getResourceName(metadata.Spec.Name, metadata.Labels)
		if err != nil {
			c.logger.Errorw("Get name from BkGatewayPluginMetadata failed", "err", err, "metadata", metadata.ObjectMeta)
			continue
		}
		apisixConfig.PluginMetadatas[name] = apisix.NewPluginMetadata(name, obj)
	}

	return apisixConfig, nil
}

func (c *Converter) getResourceName(specName string, labels map[string]string) (string, error) {
	if len(specName) != 0 {
		return specName, nil
	}
	name, ok := labels[config.BKAPIGatewayLabelKeyResourceName]
	if !ok {
		return name, eris.New("Neither spec.name nor metadata.labels.\"gateway.bk.tencent.com/name\" is provided")
	}
	return name, nil
}
