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

package utils

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	json "github.com/json-iterator/go"
)

type HTTPBaseClient struct {
	host   string
	client *http.Client
}

type HttpOption func(client *HTTPBaseClient)

func WithTimeout(timeoutSecond int) HttpOption {
	return func(client *HTTPBaseClient) {
		client.client.Timeout = time.Duration(timeoutSecond) * time.Second
	}
}

type DoHttpOption func(req *http.Request)

func WithHttpHeader(headers map[string]string) DoHttpOption {
	return func(req *http.Request) {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}
}

func WithParam(params map[string]string) DoHttpOption {
	return func(req *http.Request) {
		q := req.URL.Query()
		for key, val := range params {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}
}

// NewClient ...
func NewClient(host string, options ...HttpOption) *HTTPBaseClient {
	client := &HTTPBaseClient{
		host:   host,
		client: &http.Client{},
	}
	for _, op := range options {
		op(client)
	}
	return client
}

// Get http get method
func (h *HTTPBaseClient) Get(url string, doOptions ...DoHttpOption) (*http.Response, error) {
	// new request
	req, err := http.NewRequest("GET", h.host+url, nil)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("new request is fail:%w", err)
	}
	for _, op := range doOptions {
		op(req)
	}
	return h.client.Do(req)
}

// Post http post method
func (h *HTTPBaseClient) Post(url string, body interface{}, doOptions ...DoHttpOption) (*http.Response, error) {
	// add post body
	var bodyJson []byte
	var req *http.Request
	if body != nil {
		var err error
		bodyJson, err = json.Marshal(body)
		if err != nil {
			log.Println(err)
			return nil, fmt.Errorf("http post body to json failed:%w", err)
		}
	}
	req, err := http.NewRequest("POST", h.host+url, bytes.NewBuffer(bodyJson))
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("new request is fail: %w", err)
	}
	req.Header.Set("Content-type", "application/json")
	for _, op := range doOptions {
		op(req)
	}
	return h.client.Do(req)
}
