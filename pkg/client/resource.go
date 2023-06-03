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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/api/handler"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugins/body"
)

const (
	GetLeaderURL    = "/v1/leader"
	DiffResourceURL = "/v1/resources/diff"
	ListResourceURL = "/v1/resources/list"
	SyncResourceURL = "/v1/resources/sync"
)

type ResourceClient struct {
	client *gentleman.Client
	Apikey string
}

var (
	serverHost     string
	serverBindPort = 6004
)

// Init client
func Init(cfg *config.Config) {
	switch {
	case cfg.HttpServer.BindAddress != "":
		serverHost = fmt.Sprintf(
			"http://%s:%d",
			cfg.HttpServer.BindAddress,
			cfg.HttpServer.BindPort,
		)
	case cfg.HttpServer.BindAddressV6 != "":
		serverHost = fmt.Sprintf(
			"http://%s:%d",
			cfg.HttpServer.BindAddressV6,
			cfg.HttpServer.BindPort,
		)
	default:
		serverHost = fmt.Sprintf("http://127.0.0.1:%d", cfg.HttpServer.BindPort)
	}

	serverBindPort = cfg.HttpServer.BindPort
}

// NewResourceClient New resource client with host and apiKey
func NewResourceClient(host string, apiKey string) *ResourceClient {
	cli := gentleman.New()
	cli.URL(host)
	return &ResourceClient{
		client: cli,
		Apikey: apiKey,
	}
}

// GetLeaderResourceClient get leader resource client
func GetLeaderResourceClient(apiKey string) (*ResourceClient, error) {
	client := NewResourceClient(serverHost, apiKey)
	leader, err := client.GetLeader()
	if err != nil {
		return nil, err
	}
	leaderHost := getHostFromLeaderName(leader)
	if leaderHost == "" {
		return nil, errors.New("empty leader host")
	}
	return NewResourceClient(leaderHost, apiKey), nil
}

// GetLeader Resource leader instance
func (r *ResourceClient) GetLeader() (string, error) {
	request := r.client.Request()
	request.Path(GetLeaderURL)
	request.Method(http.MethodGet)
	var leader string
	return leader, r.DoHttpRequest(request, SetAuth(r.Apikey), SendAndDecodeResp(&leader))
}

// Diff resource both gateway and apiSix
func (r *ResourceClient) Diff(req *handler.DiffReq) (*handler.DiffInfo, error) {
	request := r.client.Request()
	request.Path(DiffResourceURL)
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	var res handler.DiffInfo
	return &res, r.DoHttpRequest(request, SetAuth(r.Apikey), SendAndDecodeResp(&res))
}

// List Resource
func (r *ResourceClient) List(req *handler.ListReq) (handler.ListInfo, error) {
	request := r.client.Request()
	request.Path(ListResourceURL)
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	var res handler.ListInfo
	return res, r.DoHttpRequest(request, SetAuth(r.Apikey), SendAndDecodeResp(&res))
}

// Sync Resource between gateway and apiSix
func (r *ResourceClient) Sync(req *handler.SyncReq) error {
	request := r.client.Request()
	request.Path(SyncResourceURL)
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	return r.DoHttpRequest(request, SetAuth(r.Apikey), SendAndDecodeResp(nil))
}

// DoHttpRequest do http request with opt
func (r *ResourceClient) DoHttpRequest(request *gentleman.Request, options ...RequestOption) error {
	for _, opt := range options {
		err := opt(request)
		if err != nil {
			return err
		}
	}
	return nil
}

// RequestOption http option
type RequestOption func(request *gentleman.Request) error

// SetAuth set basic auth
func SetAuth(apiKey string) RequestOption {
	return func(request *gentleman.Request) error {
		request.SetHeader("Authorization",
			fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(
				[]byte(fmt.Sprintf("%s:%s", constant.ApiAuthAccount, apiKey)))))
		return nil
	}
}

// SendAndDecodeResp do http request and decode resp
func SendAndDecodeResp(result interface{}) RequestOption {
	return func(request *gentleman.Request) error {
		resp, err := request.Send()
		if err != nil {
			return fmt.Errorf("send http fail:%w", err)
		}
		var res utils.CommonResp
		err = json.Unmarshal(resp.Bytes(), &res)
		if err != nil {
			return fmt.Errorf("unmarshal http resp err:%w", err)
		}
		if res.Error.Code != "" {
			return fmt.Errorf("code:%s,msg:%s", res.Error.Code, res.Error.Message)
		}
		if result != nil {
			resultByte, err := json.Marshal(res.Data)
			if err != nil {
				return fmt.Errorf("marshal http result data err:%w", err)
			}
			return json.Unmarshal(resultByte, &result)
		}
		return nil
	}
}

// getHostFromLeaderName eg: in:somename-ip1,ip2 out: http://ip1:port
func getHostFromLeaderName(leader string) string {
	// format somename-ip1,ip2,ip3
	splitRes := strings.Split(leader, "_")
	addrAll := splitRes[len(splitRes)-1]
	if len(addrAll) == 0 {
		return ""
	}
	addrList := strings.Split(addrAll, ",")
	if ip := net.ParseIP(addrList[0]); ip == nil {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", addrList[0], serverBindPort)
}
