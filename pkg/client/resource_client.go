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
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/protocol"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/middleware"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"

	json "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

const (
	GetLeaderURL    = "/v1/leader"
	DiffResourceURL = "/v1/resources/diff"
	ListResourceURL = "/v1/resources/list"
	SyncResourceURL = "/v1/resources/sync"
)

type ResourceClient struct {
	client *utils.HTTPBaseClient
	Apikey string
}

func NewResourceClient(host string, apiKey string) *ResourceClient {
	return &ResourceClient{
		client: utils.NewClient(host),
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
	resp, err := r.client.Get(GetLeaderURL, utils.WithHttpHeader(
		map[string]string{
			middleware.ApiKeyHeader: r.Apikey,
		}))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var res protocol.CommonResp
	err = json.Unmarshal(result, &res)
	if err != nil {
		return "", err
	}
	if res.Code != "" {
		return "", fmt.Errorf("code:%s,msg:%s", res.Code, res.Message)
	}
	return cast.ToString(res.Data), nil
}

// Diff Resource
func (r *ResourceClient) Diff(req *protocol.DiffReq) (*protocol.DiffResp, error) {
	resp, err := r.client.Post(DiffResourceURL, req, utils.WithHttpHeader(
		map[string]string{
			middleware.ApiKeyHeader: r.Apikey,
		}))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res protocol.DiffResp
	err = json.Unmarshal(result, &res)
	if err != nil {
		return nil, err
	}
	if res.Code != "" {
		return nil, fmt.Errorf("code:%s,msg:%s", res.Code, res.Message)
	}
	return &res, nil
}

// List Resource
func (r *ResourceClient) List(req *protocol.ListReq) (*protocol.ListResp, error) {
	resp, err := r.client.Post(ListResourceURL, req, utils.WithHttpHeader(
		map[string]string{
			middleware.ApiKeyHeader: r.Apikey,
		}))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res protocol.ListResp
	err = json.Unmarshal(result, &res)
	if err != nil {
		return nil, err
	}
	if res.Code != "" {
		return nil, fmt.Errorf("code:%s,msg:%s", res.Code, res.Message)
	}
	return &res, nil
}

// Sync Resource
func (r *ResourceClient) Sync(req *protocol.SyncReq) error {
	resp, err := r.client.Post(SyncResourceURL, req, utils.WithHttpHeader(
		map[string]string{
			middleware.ApiKeyHeader: r.Apikey,
		}))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var res protocol.CommonResp
	err = json.Unmarshal(result, &res)
	if err != nil {
		return err
	}
	if res.Code != "" {
		return fmt.Errorf("code:%s,msg:%s", res.Code, res.Message)
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
