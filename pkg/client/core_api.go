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
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	gentleman "gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugins/body"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
)

const (
	reportPublishEventURL = "/api/v1/micro-gateway/release/%s/events/"
)

var coreAPIClient *CoreAPIClient

var coreOnce sync.Once

type CoreAPIClient struct {
	baseClient
}

// InitCoreAPIClient init core api client
func InitCoreAPIClient(cfg *config.Config) {
	coreOnce.Do(func() {
		coreAPIClient = newCoreAPIClient(cfg.EventReporter.CoreAPIHost, cfg.Instance.ID, cfg.Instance.Secret)
	})
}

// GetCoreAPIClient get core api client
func GetCoreAPIClient() *CoreAPIClient {
	return coreAPIClient
}

// NewCoreAPIClient New core_api client with instance_id and instance_secret
func newCoreAPIClient(endpoints string, instanceID string, instanceSecret string) *CoreAPIClient {
	cli := gentleman.New()
	cli.URL(endpoints)

	// set instance
	cli.SetHeader("X-Bk-Micro-Gateway-Instance-Id", instanceID)
	cli.SetHeader("X-Bk-Micro-Gateway-Instance-Secret", instanceSecret)

	return &CoreAPIClient{
		baseClient: baseClient{
			client: cli,
		},
	}
}

// AddPublishEvent report event to core_api
func (c *CoreAPIClient) AddPublishEvent(ctx context.Context, req *AddEventReq) error {
	if req.PublishID == "" {
		return errors.New("publish_id is empty")
	}
	request := c.client.Request()
	request.Path(fmt.Sprintf(reportPublishEventURL, req.PublishID))
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	return c.doHttpRequest(request, sendAndDecodeResp(nil))
}
