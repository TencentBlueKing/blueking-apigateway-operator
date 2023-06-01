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
	serverAddr     string
	serverBindPort = 6004
)

// Init ...
func Init(cfg *config.Config) {
	switch {
	case cfg.HttpServer.BindAddress != "":
		serverAddr = fmt.Sprintf(
			"%s:%d",
			cfg.HttpServer.BindAddress,
			cfg.HttpServer.BindPort,
		)
	case cfg.HttpServer.BindAddressV6 != "":
		serverAddr = fmt.Sprintf(
			"%s:%d",
			cfg.HttpServer.BindAddressV6,
			cfg.HttpServer.BindPort,
		)
	default:
		serverAddr = fmt.Sprintf("127.0.0.1:%d", cfg.HttpServer.BindPort)
	}

	serverBindPort = cfg.HttpServer.BindPort
}

func NewResourceClient(host string, apiKey string) *ResourceClient {
	cli := gentleman.New()
	cli.URL(host)
	return &ResourceClient{
		client: cli,
		Apikey: apiKey,
	}
}

// GetLeaderResourceClient ...
func GetLeaderResourceClient(apiKey string) (*ResourceClient, error) {
	client := NewResourceClient("http://"+serverAddr, apiKey)
	leader, err := client.GetLeader()
	if err != nil {
		return nil, err
	}
	leaderHost := getHostFromLeaderName(leader)
	if leaderHost == "" {
		return nil, errors.New("empty leader host")
	}
	return NewResourceClient("http://"+leaderHost, apiKey), nil
}

// GetLeader Resource
func (r *ResourceClient) GetLeader() (string, error) {
	request := r.client.Request()
	request.Path(GetLeaderURL)
	request.Method(http.MethodGet)
	r.SetAuth(request)
	result, err := request.Send()
	if err != nil {
		return "", err
	}
	var leader string
	return leader, r.DecodeResp(result, &leader)
}

// Diff Resource
func (r *ResourceClient) Diff(req *handler.DiffReq) (*handler.DiffInfo, error) {
	request := r.client.Request()
	request.Path(DiffResourceURL)
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	r.SetAuth(request)
	result, err := request.Send()
	if err != nil {
		return nil, err
	}
	var res handler.DiffInfo
	return &res, r.DecodeResp(result, &res)
}

// List Resource
func (r *ResourceClient) List(req *handler.ListReq) (handler.ListInfo, error) {
	request := r.client.Request()
	request.Path(ListResourceURL)
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	r.SetAuth(request)
	result, err := request.Send()
	if err != nil {
		return nil, err
	}
	var res handler.ListInfo
	return res, r.DecodeResp(result, &req)
}

// Sync Resource
func (r *ResourceClient) Sync(req *handler.SyncReq) error {
	request := r.client.Request()
	request.Path(SyncResourceURL)
	request.Method(http.MethodPost)
	request.Use(body.JSON(req))
	r.SetAuth(request)
	result, err := request.Send()
	if err != nil {
		return err
	}
	return r.DecodeResp(result, nil)
}

func (r *ResourceClient) SetAuth(request *gentleman.Request) {
	request.SetHeader("Authorization",
		fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("bk-operator:%s", r.Apikey)))))
}

func (r *ResourceClient) DecodeResp(response *gentleman.Response, result interface{}) error {
	var res utils.CommonResp
	err := json.Unmarshal(response.Bytes(), &res)
	if err != nil {
		return err
	}
	if res.Error.Code != "" {
		return fmt.Errorf("code:%s,msg:%s", res.Error.Code, res.Error.Message)
	}
	if result != nil {
		resultByte, err := json.Marshal(res.Data)
		if err != nil {
			return err
		}
		return json.Unmarshal(resultByte, &result)
	}
	return nil
}

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
	return fmt.Sprintf("%s:%d", addrList[0], serverBindPort)
}
