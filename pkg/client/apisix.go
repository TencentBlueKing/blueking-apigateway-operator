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

package client

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"gopkg.in/eapache/go-resiliency.v1/retrier"
	retry "gopkg.in/h2non/gentleman-retry.v2"
	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugins/url"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

const (
	getPublishVersionURL = "/api/:gateway/:stage/__apigw_version"
)

var apisixClient *ApisixClient

var apisxiOnce sync.Once

type ApisixClient struct {
	baseClient
	// apisix版本探测次数
	versionProbeCount int
	// apisix版本探测间隔
	versionProbeInterval time.Duration
}

// InitApisixClient init apisix cli
func InitApisixClient(cfg *config.Config) {
	apisxiOnce.Do(func() {
		cli := gentleman.New()
		cli.URL(cfg.EventReporter.ApisixHost)
		apisixClient = &ApisixClient{
			baseClient:           baseClient{client: cli},
			versionProbeCount:    cfg.EventReporter.VersionProbe.Retry.Count,
			versionProbeInterval: cfg.EventReporter.VersionProbe.Retry.Interval,
		}
	})
}

func GetApisixClient() *ApisixClient {
	return apisixClient
}

// GetReleaseVersion get apisix release info
func (a *ApisixClient) GetReleaseVersion(gatewayName string, stageName string,
	publishID string) (*VersionRouteResp, error) {
	request := a.client.Request()
	request.Path(getPublishVersionURL)
	request.Method(http.MethodGet)
	request.Use(url.Param("gateway", gatewayName))
	request.Use(url.Param("stage", stageName))
	retryStrategy := retrier.New(retrier.ConstantBackoff(
		a.versionProbeCount, a.versionProbeInterval), nil)
	var resp VersionRouteResp
	retryError := new(error)
	// set retry strategy
	retry.Evaluator = retryEvaluator(gatewayName, stageName, cast.ToInt64(publishID), retryError, &resp)
	retryPlugin := retry.New(retryStrategy)
	request.Use(retryPlugin)
	_, err := request.Send()
	if err != nil {
		return nil, err
	}
	if *retryError != nil {
		return &resp, *retryError
	}

	return &resp, nil
}

// retryEvaluator retry strategy
func retryEvaluator(gateway string, stage string, publishID int64, retryError *error,
	resp *VersionRouteResp) retry.EvalFunc {
	return func(err error, res *http.Response, req *http.Request) error {
		if err != nil {
			return err
		}
		if res.StatusCode >= http.StatusInternalServerError || res.StatusCode == http.StatusTooManyRequests {
			return retry.ErrServer
		}
		// 虚拟路由不存在,继续重试
		if res.StatusCode == http.StatusNotFound {
			*retryError = fmt.Errorf(
				"configuration [gateway: %s,stage: %s] version route not found", gateway, stage)
			logging.GetLogger().Info(*retryError)
			return *retryError
		}
		if res.StatusCode == http.StatusOK {
			// 解析返回结果
			defer res.Body.Close()
			result, readErr := io.ReadAll(res.Body)
			if readErr != nil {
				*retryError = fmt.Errorf("read configuration [gateway: %s,state: %s] version route body err: %w",
					gateway, stage, readErr)
				logging.GetLogger().Error(*retryError)
				return *retryError
			}
			unmarshalErr := json.Unmarshal(result, &resp)
			if unmarshalErr != nil {
				*retryError = fmt.Errorf(
					"unmarshal configuration [gateway: %s,stage: %s] version route body err: %w",
					gateway, stage, unmarshalErr)
				logging.GetLogger().Error(*retryError)
				return *retryError
			}

			// 判断版本号
			if resp.PublishID < publishID {
				// 如果获取到的版本号比当前小，说明当前的版本还未加载完成
				*retryError = fmt.Errorf(
					"configuration [gateway: %s,stage: %s]  [current: %d, expected: %d] is not latest",
					gateway, stage, resp.PublishID, publishID)
				logging.GetLogger().Info(*retryError)
				return *retryError
			}

			// 如果发布的版本号比当前大，说明已经覆盖加载完成
			if resp.PublishID >= publishID {
				*retryError = nil
				return nil
			}
		}
		return nil
	}
}
