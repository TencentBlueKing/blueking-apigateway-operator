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

package registry

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
)

type mockKV struct {
	mock.Mock
	txn *mockTxn
	ch  chan interface{}
}

func (m *mockKV) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return &clientv3.PutResponse{}, nil
}

func (m *mockKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	args := m.Called(key)
	err := args.Get(1)
	if err != nil {
		return args.Get(0).(*clientv3.GetResponse), err.(error)
	}
	return args.Get(0).(*clientv3.GetResponse), nil
}

func (m *mockKV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	return &clientv3.DeleteResponse{}, nil
}

func (m *mockKV) Compact(
	ctx context.Context,
	rev int64,
	opts ...clientv3.CompactOption,
) (*clientv3.CompactResponse, error) {
	return nil, nil
}

func (m *mockKV) Do(ctx context.Context, op clientv3.Op) (clientv3.OpResponse, error) {
	return clientv3.OpResponse{}, nil
}

func (m *mockKV) Txn(ctx context.Context) clientv3.Txn { return m.txn }

type mockTxn struct {
	mock.Mock

	result *clientv3.TxnResponse
	err    error
}

func (m *mockTxn) If(cs ...clientv3.Cmp) clientv3.Txn {
	args := m.Called(cs)
	return args.Get(0).(*mockTxn)
}

func (m *mockTxn) Then(ops ...clientv3.Op) clientv3.Txn {
	return m
}

func (m *mockTxn) Else(ops ...clientv3.Op) clientv3.Txn {
	return m
}

func (m *mockTxn) Commit() (*clientv3.TxnResponse, error) {
	return m.result, m.err
}

type mockWatcher struct {
	mock.Mock
	ch *chan clientv3.WatchResponse
}

func (m *mockWatcher) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return *m.ch
}

func (m *mockWatcher) RequestProgress(ctx context.Context) error {
	return nil
}

func (m *mockWatcher) Close() error {
	return nil
}

func buildGetResponse(key, value string) *clientv3.GetResponse {
	return &clientv3.GetResponse{
		Count: 1,
		Kvs: []*mvccpb.KeyValue{
			{
				Key:   []byte(key),
				Value: []byte(value),
			},
		},
	}
}

func buildMockKV() *mockKV {
	kv := &mockKV{}
	yaml := `metadata:
  name: resource
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
spec:
  name: user_verified_unrequired
  desc: ''
  id: 0
  plugins:
  - name: bk-resource-context
    config:
      bk_resource_id: 0
      bk_resource_name: user_verified_unrequired
      verified_app_required: false
      verified_user_required: false
      resource_perm_required: false
      skip_user_verification: false
  service: stage-stag
  protocol: http
  methods:
  - GET
  timeout:
    connect: 30
    read: 30
    send: 30
  uri: /api/bkuser/prod/user-verified-unrequired/
  matchSubPath: false
  upstream:
  rewrite:
    enabled: true
    method: GET
    path: /echo/
    headers: {}
    stageHeaders: append
    serviceHeaders: append`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayResource/resource").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayResource/resource", yaml), nil)
	yaml = `metadata:
  name: service
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
spec:
  name: stage-stag
  desc: ''
  id: stage-183
  upstream:
    type: roundrobin
    hashOn:
    key:
    checks:
      active:
        type: http
        timeout:
        concurrency: 3
        httpPath: /healthz
        healthy:
        unhealthy:
      passive:
    scheme: http
    retries:
    retryTimeout:
    passHost: node
    upstreamHost:
    tlsEnable: false
    externalDiscoveryType:
    externalDiscoveryConfig:
    discoveryType:
    serviceName:
    nodes:
    - host: 127.0.0.1
      port: 2333
      weight: 100
      priority: 1
    timeout:
      connect: 30
      read: 30
      send: 30
  rewrite:
    enabled: false
    headers: {}
  plugins: []`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayService/service").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayService/service", yaml), nil)
	yaml = `metadata:
  name: endpoints
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
spec:
  nodes:
  - host: 127.0.0.1
    port: 8080
    priority: 0
    weight: 100`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayEndpoints/endpoints").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayEndpoints/endpoints", yaml), nil)
	yaml = `metadata:
  name: tls
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
spec:
  gatewayTLSSecretRef: secret`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayTLS/tls").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayTLS/tls", yaml), nil)
	yaml = `metadata:
  name: stage
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
    gateway.bk.tencent.com/publish_id: publish_id
  annotations: {}
spec:
  name: stag
  desc: ''
  vars:
    x: '4'
  rewrite:
    enabled: true
    headers: {}
  plugins:
  - name: prometheus
    config: {}
  - name: bk-request-id
    config: {}
  - name: bk-permission
    config: {}
  - name: file-logger
    config:
    path: logs/access.log
  - name: bk-auth-verify
    config: {}
  - name: bk-auth-validate
    config: {}
  - name: bk-jwt
    config: {}
  - name: bk-delete-sensitive
    config: {}
  - name: bk-stage-context
    config:
      instance_id: xxxxxxxxxxxxxxxx
      bk_gateway_name: micro-gateway
      bk_gateway_id: 0
      bk_stage_name: stag
      jwt_private_key: xxxxxxxxxxxxxxxxxxxxx
      controller:
        endpoints:
        - https://bk-apigateway.apigw.example.com/
        base_path: /stag/api/v1/edge-controller
        jwt_auth:
          secret: xxxxxxxxxxxxxxxxxxxxxxxx
  domain: ''
  pathPrefix: /stag`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayStage/stage").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayStage/stage", yaml), nil)
	yaml = `metadata:
  name: config
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
spec:
  name: micro-gateway
  desc: ''
  instanceID: xxxxx
  controller:
    basePath: /stag/api/v1/edge-controller
    endpoints:
    - https://xxx/
    jwtAuth:
      secret: xxxxx`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayConfig/config").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayConfig/config", yaml), nil)
	yaml = `metadata:
  name: file-logger
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
spec:
  name: file-logger
  desc: ''
  config:
    log_format:
      host: $host
      '@timestamp': $time_iso8601
      client_ip: $remote_addr
      request_id: $bk_request_id
      api_id: $bk_gateway_id
      api_name: $bk_gateway_name
      resource_name: $resource_name
      app_code: $bk_app_code
      stage: $bk_stage_name`
	kv.On("Get", "/test/gateway/stage/v1beta1/BkGatewayPluginMetadata/file-logger").
		Return(buildGetResponse("/test/gateway/stage/v1beta1/BkGatewayPluginMetadata/file-logger", yaml), nil)
	yaml = `metadata:
  name: secret
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: "-"
  annotations: {}
data:
  ca.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpVENDQVM2Z0F3SUJBZ0lRREpiU2UwVUQwL0g1YkNxL0l0RkNRekFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxpWTNNdFkyRXRZbU56TFhObGNuWnBZMlZ6TFhOMFlXTnJNQjRYRFRJeU1EVXhNekUxTlRJeQpNRm9YRFRJeU1EZ3hNVEUxTlRJeU1Gb3dKREVpTUNBR0ExVUVBeE1aWW1OekxXTmhMV0pqY3kxelpYSjJhV05sCmN5MXpkR0ZqYXpCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQk5qK09qMVBVS0YvSjQ1YUhOUUUKWnVCSlpDUTRobCt4SmhHQ2NCWjRzRnQvVE9CSU5xcEFFREIxS0JXVWw3QkdRVC9ienp5Ry9jTkhaeVl4RENQRwp5aFdqUWpCQU1BNEdBMVVkRHdFQi93UUVBd0lDcERBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXCkJCUmZVRjhxWkJ3b3BJSnNnNXlUb0NKaFBoZDJkekFLQmdncWhrak9QUVFEQWdOSkFEQkdBaUVBdEZET0pmRW0KYlRmeHpiT0N4VzlzSFZLNnQ4bFZBd2RMTTFmUHVVN2NCUFVDSVFDUkQyQm5DcExHdERDL1pYOHRkY015M28vYQo1S3E2TnhWMGVVbjFBWjhqaVE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM4akNDQXBpZ0F3SUJBZ0lRSDZxWC80NWFGK0RjQVUyNHJhaGtwREFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxpWTNNdFkyRXRZbU56TFhObGNuWnBZMlZ6TFhOMFlXTnJNQjRYRFRJeU1EY3dOekE1TWpVegpOVm9YRFRNd01Ea3lNekE1TWpVek5Wb3dnWU14Q3pBSkJnTlZCQVlUQWtOT01Rc3dDUVlEVlFRSUV3SkhSREVMCk1Ba0dBMVVFQnhNQ1Uxb3hEVEFMQmdOVkJBa1RCRmxJU2tReEVEQU9CZ05WQkFvVEIxUmxibU5sYm5ReEVUQVAKQmdOVkJBc1RDRUpzZFdWTGFXNW5NU1l3SkFZRFZRUURFeDFpWTNNdFkyeHBaVzUwTFdKamN5MXpaWEoyYVdObApjeTF6ZEdGamF6Q0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQUs0SmRCNFpObnA5CmRUMVFVdFR2Mk4yUTQ2NjRQczJDZnFZV25VQWZUR054TVFBRE9JWXgraURqYUp6T3EybUtYU0trYWNPRTFHb0cKU0dRMG51ZVJrcnRWSmt0Nkt5WVNjNUg3SFo0Q21BVU96NDVTUHBwWlF2ZmRKRFZqQ25sLzVZS05FMy81cHNLawoyYmo2aWZ6L1NhQzNzU0V3N1lTNExNWmV5cmwzZWk2Sy9LYmxTNzFKai9iOGJlWUdRVnN4eFE4MStLWkxLMWFjCjB3dUpuQ3gxOG4vK3Ixa2hlWCt5QTFndE55eVhQVG0vdWlqalVuRmEzTHlQUmNsZmtzelBuSFBYV3pGNVVxekMKa3pBZHhGWlRtS2I4R25peExuU2thdGJOOUt5Tk5TdXFDVWNhT1V3dTgzL0ozY1FucG9kdkpjUlg1cE5zcSt4agp6czNZUDhjbW02VUNBd0VBQWFPQmdEQitNQTRHQTFVZER3RUIvd1FFQXdJRm9EQWRCZ05WSFNVRUZqQVVCZ2dyCkJnRUZCUWNEQVFZSUt3WUJCUVVIQXdJd0RBWURWUjBUQVFIL0JBSXdBREFmQmdOVkhTTUVHREFXZ0JSZlVGOHEKWkJ3b3BJSnNnNXlUb0NKaFBoZDJkekFlQmdOVkhSRUVGekFWZ2hNcUxtSnJZbU56TG5SbGJtTmxiblF1WTI5dApNQW9HQ0NxR1NNNDlCQU1DQTBnQU1FVUNJUUR1WWdIQmxxdWRIU2FtcTJIU1F4RUt4dGhZTmtIYi9PUk9URnh5CkF4bk5qQUlnYkVZeGU3bWY4d3hzSkc3dmdwcjVoZHptT3JPSmpKUElGbzhjSzNPamhJUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBcmdsMEhoazJlbjExUFZCUzFPL1kzWkRqcnJnK3pZSitwaGFkUUI5TVkzRXhBQU00CmhqSDZJT05vbk02cmFZcGRJcVJwdzRUVWFnWklaRFNlNTVHU3UxVW1TM29ySmhKemtmc2RuZ0tZQlE3UGpsSSsKbWxsQzk5MGtOV01LZVgvbGdvMFRmL21td3FUWnVQcUovUDlKb0xleElURHRoTGdzeGw3S3VYZDZMb3I4cHVWTAp2VW1QOXZ4dDVnWkJXekhGRHpYNHBrc3JWcHpUQzRtY0xIWHlmLzZ2V1NGNWY3SURXQzAzTEpjOU9iKzZLT05TCmNWcmN2STlGeVYrU3pNK2NjOWRiTVhsU3JNS1RNQjNFVmxPWXB2d2FlTEV1ZEtScTFzMzBySTAxSzZvSlJ4bzUKVEM3emY4bmR4Q2VtaDI4bHhGZm1rMnlyN0dQT3pkZy94eWFicFFJREFRQUJBb0lCQUhVMVJxK1NtVjhMS1M4ZQo3bm9jQWZqT1FKaUYyejM2dWFMUHJoM21Oa0x1azJxSHdNU1gyZlhXVWJqeGN2M0VRbzgzSFVlaEtKRXpKQVBnCmNIaFNVUGk3RXV4WUhjRXBRZzQ1aWF2RjRXM2VtS2duK2FObnBETmNDcXV0eFBzb3lJQVExT1ltVTBuWlRneEgKSnpGdEdNQVZsa1JkT0VsZTVFREF6RlQyQXlKZU9kZGw4cHFvN09BajdKZ2dGaGNuNnB3UTdzcGNqRXZYYVdZMgoxNE1GUllFbURNQ2xSeEFXbjhsdVJQKytWNWtGU0RuUGhPUmYwWjQrRklsd1ZwTFI3eHpsSVd6bTEvS3RMSTQ1CkZMQ09RWDN4WS9YQVJ1bVlPd3U5K3ZNakpMN2Q0ZGZ1UnpkWlAwdkJjdWx2K1JJQWVtcUJySFgxSitLNTROeFUKS25HMUFhRUNnWUVBMjl6MkQ2cVRVR1JwUzZ0eFNzTjZGaXhzN2NMSkd5U2x6bVlyMjJIZGtvSDRGaXBjWDI1Mwphb09saEt5L2s4ZkpPY055dkJFUWZIN3k4VHB2ZXlDMy9mcnBNTHJRT09ORnhhcFlPU3hBeVpJZGNMQ3QrTW94Ckw4ZU4yQTJXQnpHWldzL2Yra1cyK2J0Z0Jza1U2ejUyZnd0YXF0aG8xK1A1alRGT01wTnd0YzBDZ1lFQXlxUksKL2huZk9waDk5WUQ1V2J1VVlvenlOK3ROTEliSm4vK2t5Mm1HajB1dnFxMTVNb00vNERaRWdaTkZWeGxHSDh0dgp1WlBqM1ZmWXE1WVBwcDhDa2g1K1VJQWk2bkFKalpSbTAvK2ozU2hDb1A1SWJLaFRrZVl2ZDV6VEkyaXY3K2piClpWeEY0YzZTS0lsMGlyb0FPc2JROEI5TklaQlVxTGFYMytSbHBUa0NnWUI4OTFDY2d2V01ZaVkvTGtrTWw2TFMKNjVsV1lycHZ4UnJBLysyNW1oeVlZMnNoSGg2MjExRGtwOEx5Y0VYTHQyaTJmbEsrZG15S2RwV2JhdjFtWEtoMwpvWi9kWkxGcFJEU3FMekpKL004dVF2Q2MxcTlyazNEMW1WVVVFbFRON2ZFZVhyME53WVpJMTZteThhUUVPZEtjClQxWFBlWVhOLy9RZHZvS1YySnZkbVFLQmdBM2Z0NzZ6K01PalF5UjI0eHVRcXpVZ0gwbFMwK0xUaTZSbnRWbXoKN09HTXRnZENmMFRGRmE5OUo2MlRickRxNnhFc05ZY0lLQmEwZUFJdmNQemdjQ1dlN3RrR0hOM2VNOWs4cXRtaQo3QTR0UG5xVlRsSWFLRGFhQXUvMmpjSWozYi9ZT29VekR4bkpzZG9TcHljRVd4V3JIUTBEcDUwL3Ezd3RuREpaCkNkaUpBb0dBZExNSmN3TnBqbmp1N09LWG14cDlUYWYvTjFVdzdmSGhKelRKTUFBWHZ0QXVEWkVWaDNBR1UrUjMKakgxbW0rSzJXQmZHSzNJZExZRVRuczlnMUZiVkl0ZHc3RldvQ2QvdXNSQXgxRXNpbjU2bGlOYUNTbjVUdFN0cApSZi9RVWFEZ21FbDJ3KzBtTlFRMTVWQXFWMHRDbGRPYllVSG9tRW51SU5nWVNETEtmSFk9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==`
	kv.On("Get", "/test/gateway/-/v1/Secret/secret").
		Return(buildGetResponse("/test/gateway/-/v1/Secret/secret", yaml), nil)
	yaml = `metadata:
  name: secret
  labels:
    gateway.bk.tencent.com/gateway: gateway
    gateway.bk.tencent.com/stage: stage
  annotations: {}
data:
  ca.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpVENDQVM2Z0F3SUJBZ0lRREpiU2UwVUQwL0g1YkNxL0l0RkNRekFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxpWTNNdFkyRXRZbU56TFhObGNuWnBZMlZ6TFhOMFlXTnJNQjRYRFRJeU1EVXhNekUxTlRJeQpNRm9YRFRJeU1EZ3hNVEUxTlRJeU1Gb3dKREVpTUNBR0ExVUVBeE1aWW1OekxXTmhMV0pqY3kxelpYSjJhV05sCmN5MXpkR0ZqYXpCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQk5qK09qMVBVS0YvSjQ1YUhOUUUKWnVCSlpDUTRobCt4SmhHQ2NCWjRzRnQvVE9CSU5xcEFFREIxS0JXVWw3QkdRVC9ienp5Ry9jTkhaeVl4RENQRwp5aFdqUWpCQU1BNEdBMVVkRHdFQi93UUVBd0lDcERBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXCkJCUmZVRjhxWkJ3b3BJSnNnNXlUb0NKaFBoZDJkekFLQmdncWhrak9QUVFEQWdOSkFEQkdBaUVBdEZET0pmRW0KYlRmeHpiT0N4VzlzSFZLNnQ4bFZBd2RMTTFmUHVVN2NCUFVDSVFDUkQyQm5DcExHdERDL1pYOHRkY015M28vYQo1S3E2TnhWMGVVbjFBWjhqaVE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM4akNDQXBpZ0F3SUJBZ0lRSDZxWC80NWFGK0RjQVUyNHJhaGtwREFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxpWTNNdFkyRXRZbU56TFhObGNuWnBZMlZ6TFhOMFlXTnJNQjRYRFRJeU1EY3dOekE1TWpVegpOVm9YRFRNd01Ea3lNekE1TWpVek5Wb3dnWU14Q3pBSkJnTlZCQVlUQWtOT01Rc3dDUVlEVlFRSUV3SkhSREVMCk1Ba0dBMVVFQnhNQ1Uxb3hEVEFMQmdOVkJBa1RCRmxJU2tReEVEQU9CZ05WQkFvVEIxUmxibU5sYm5ReEVUQVAKQmdOVkJBc1RDRUpzZFdWTGFXNW5NU1l3SkFZRFZRUURFeDFpWTNNdFkyeHBaVzUwTFdKamN5MXpaWEoyYVdObApjeTF6ZEdGamF6Q0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQUs0SmRCNFpObnA5CmRUMVFVdFR2Mk4yUTQ2NjRQczJDZnFZV25VQWZUR054TVFBRE9JWXgraURqYUp6T3EybUtYU0trYWNPRTFHb0cKU0dRMG51ZVJrcnRWSmt0Nkt5WVNjNUg3SFo0Q21BVU96NDVTUHBwWlF2ZmRKRFZqQ25sLzVZS05FMy81cHNLawoyYmo2aWZ6L1NhQzNzU0V3N1lTNExNWmV5cmwzZWk2Sy9LYmxTNzFKai9iOGJlWUdRVnN4eFE4MStLWkxLMWFjCjB3dUpuQ3gxOG4vK3Ixa2hlWCt5QTFndE55eVhQVG0vdWlqalVuRmEzTHlQUmNsZmtzelBuSFBYV3pGNVVxekMKa3pBZHhGWlRtS2I4R25peExuU2thdGJOOUt5Tk5TdXFDVWNhT1V3dTgzL0ozY1FucG9kdkpjUlg1cE5zcSt4agp6czNZUDhjbW02VUNBd0VBQWFPQmdEQitNQTRHQTFVZER3RUIvd1FFQXdJRm9EQWRCZ05WSFNVRUZqQVVCZ2dyCkJnRUZCUWNEQVFZSUt3WUJCUVVIQXdJd0RBWURWUjBUQVFIL0JBSXdBREFmQmdOVkhTTUVHREFXZ0JSZlVGOHEKWkJ3b3BJSnNnNXlUb0NKaFBoZDJkekFlQmdOVkhSRUVGekFWZ2hNcUxtSnJZbU56TG5SbGJtTmxiblF1WTI5dApNQW9HQ0NxR1NNNDlCQU1DQTBnQU1FVUNJUUR1WWdIQmxxdWRIU2FtcTJIU1F4RUt4dGhZTmtIYi9PUk9URnh5CkF4bk5qQUlnYkVZeGU3bWY4d3hzSkc3dmdwcjVoZHptT3JPSmpKUElGbzhjSzNPamhJUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBcmdsMEhoazJlbjExUFZCUzFPL1kzWkRqcnJnK3pZSitwaGFkUUI5TVkzRXhBQU00CmhqSDZJT05vbk02cmFZcGRJcVJwdzRUVWFnWklaRFNlNTVHU3UxVW1TM29ySmhKemtmc2RuZ0tZQlE3UGpsSSsKbWxsQzk5MGtOV01LZVgvbGdvMFRmL21td3FUWnVQcUovUDlKb0xleElURHRoTGdzeGw3S3VYZDZMb3I4cHVWTAp2VW1QOXZ4dDVnWkJXekhGRHpYNHBrc3JWcHpUQzRtY0xIWHlmLzZ2V1NGNWY3SURXQzAzTEpjOU9iKzZLT05TCmNWcmN2STlGeVYrU3pNK2NjOWRiTVhsU3JNS1RNQjNFVmxPWXB2d2FlTEV1ZEtScTFzMzBySTAxSzZvSlJ4bzUKVEM3emY4bmR4Q2VtaDI4bHhGZm1rMnlyN0dQT3pkZy94eWFicFFJREFRQUJBb0lCQUhVMVJxK1NtVjhMS1M4ZQo3bm9jQWZqT1FKaUYyejM2dWFMUHJoM21Oa0x1azJxSHdNU1gyZlhXVWJqeGN2M0VRbzgzSFVlaEtKRXpKQVBnCmNIaFNVUGk3RXV4WUhjRXBRZzQ1aWF2RjRXM2VtS2duK2FObnBETmNDcXV0eFBzb3lJQVExT1ltVTBuWlRneEgKSnpGdEdNQVZsa1JkT0VsZTVFREF6RlQyQXlKZU9kZGw4cHFvN09BajdKZ2dGaGNuNnB3UTdzcGNqRXZYYVdZMgoxNE1GUllFbURNQ2xSeEFXbjhsdVJQKytWNWtGU0RuUGhPUmYwWjQrRklsd1ZwTFI3eHpsSVd6bTEvS3RMSTQ1CkZMQ09RWDN4WS9YQVJ1bVlPd3U5K3ZNakpMN2Q0ZGZ1UnpkWlAwdkJjdWx2K1JJQWVtcUJySFgxSitLNTROeFUKS25HMUFhRUNnWUVBMjl6MkQ2cVRVR1JwUzZ0eFNzTjZGaXhzN2NMSkd5U2x6bVlyMjJIZGtvSDRGaXBjWDI1Mwphb09saEt5L2s4ZkpPY055dkJFUWZIN3k4VHB2ZXlDMy9mcnBNTHJRT09ORnhhcFlPU3hBeVpJZGNMQ3QrTW94Ckw4ZU4yQTJXQnpHWldzL2Yra1cyK2J0Z0Jza1U2ejUyZnd0YXF0aG8xK1A1alRGT01wTnd0YzBDZ1lFQXlxUksKL2huZk9waDk5WUQ1V2J1VVlvenlOK3ROTEliSm4vK2t5Mm1HajB1dnFxMTVNb00vNERaRWdaTkZWeGxHSDh0dgp1WlBqM1ZmWXE1WVBwcDhDa2g1K1VJQWk2bkFKalpSbTAvK2ozU2hDb1A1SWJLaFRrZVl2ZDV6VEkyaXY3K2piClpWeEY0YzZTS0lsMGlyb0FPc2JROEI5TklaQlVxTGFYMytSbHBUa0NnWUI4OTFDY2d2V01ZaVkvTGtrTWw2TFMKNjVsV1lycHZ4UnJBLysyNW1oeVlZMnNoSGg2MjExRGtwOEx5Y0VYTHQyaTJmbEsrZG15S2RwV2JhdjFtWEtoMwpvWi9kWkxGcFJEU3FMekpKL004dVF2Q2MxcTlyazNEMW1WVVVFbFRON2ZFZVhyME53WVpJMTZteThhUUVPZEtjClQxWFBlWVhOLy9RZHZvS1YySnZkbVFLQmdBM2Z0NzZ6K01PalF5UjI0eHVRcXpVZ0gwbFMwK0xUaTZSbnRWbXoKN09HTXRnZENmMFRGRmE5OUo2MlRickRxNnhFc05ZY0lLQmEwZUFJdmNQemdjQ1dlN3RrR0hOM2VNOWs4cXRtaQo3QTR0UG5xVlRsSWFLRGFhQXUvMmpjSWozYi9ZT29VekR4bkpzZG9TcHljRVd4V3JIUTBEcDUwL3Ezd3RuREpaCkNkaUpBb0dBZExNSmN3TnBqbmp1N09LWG14cDlUYWYvTjFVdzdmSGhKelRKTUFBWHZ0QXVEWkVWaDNBR1UrUjMKakgxbW0rSzJXQmZHSzNJZExZRVRuczlnMUZiVkl0ZHc3RldvQ2QvdXNSQXgxRXNpbjU2bGlOYUNTbjVUdFN0cApSZi9RVWFEZ21FbDJ3KzBtTlFRMTVWQXFWMHRDbGRPYllVSG9tRW51SU5nWVNETEtmSFk9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==`
	kv.On("Get", "/test/gateway/stage/v1/Secret/secret").
		Return(buildGetResponse("/test/gateway/stage/v1/Secret/secret", yaml), nil)
	kv.On("Put", mock.Anything, mock.Anything).Return(&clientv3.PutResponse{}, nil)
	kv.On("Delete", mock.Anything).Return(&clientv3.DeleteResponse{}, nil)
	return kv
}

var _ = Describe("Etcdadapter Operations", Ordered, func() {
	var (
		testRegistry = &EtcdRegistryAdapter{
			keyPrefix:       "/test",
			currentRevision: 0,
			logger:          logging.GetLogger(),
		}
		testStageInfo = StageInfo{
			GatewayName: "gateway",
			StageName:   "stage",
		}
		kv *mockKV
	)

	BeforeAll(func() {
		kv = buildMockKV()
		testRegistry.etcdClient = &clientv3.Client{KV: kv}
		metric.InitMetric(prometheus.DefaultRegisterer)
	})

	It("will create registry successfully", func() {
		reg := NewEtcdResourceRegistry(&clientv3.Client{KV: kv}, "/bk-gateway/default")
		Expect(reg).ShouldNot(BeNil())
	})

	Context("Get from ETCD", func() {
		It("Will get BkGatewaResource from etcd", func() {
			resource := v1beta1.BkGatewayResource{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "resource",
				},
				&resource,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resource.Name).Should(Equal("resource"))
			Expect(resource.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(resource.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(*resource.Spec.ID).Should(Equal(intstr.FromInt(0)))
			Expect(resource.Spec.Name).Should(Equal("user_verified_unrequired"))
		})

		It("Will get BkGatewayService from etcd", func() {
			svc := v1beta1.BkGatewayService{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "service",
				},
				&svc,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(svc.Name).Should(Equal("service"))
			Expect(svc.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(svc.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(svc.Spec.ID).Should(Equal("stage-183"))
			Expect(svc.Spec.Name).Should(Equal("stage-stag"))
		})

		It("Will get BkGatewayStage from etcd", func() {
			stage := v1beta1.BkGatewayStage{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "stage",
				},
				&stage,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(stage.Name).Should(Equal("stage"))
			Expect(stage.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(stage.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(stage.Labels[config.BKAPIGatewayLabelKeyGatewayPublishID]).Should(Equal("publish_id"))
			Expect(stage.Spec.Name).Should(Equal("stag"))
		})

		It("Will get BkGatewayConfig from etcd", func() {
			cfg := v1beta1.BkGatewayConfig{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "config",
				},
				&cfg,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(cfg.Name).Should(Equal("config"))
			Expect(cfg.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(cfg.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(cfg.Spec.Name).Should(Equal("micro-gateway"))
			Expect(cfg.Spec.InstanceID).Should(Equal("xxxxx"))
		})

		It("Will get BkGatewayEndpoints from etcd", func() {
			eps := v1beta1.BkGatewayEndpoints{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "endpoints",
				},
				&eps,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(eps.Name).Should(Equal("endpoints"))
			Expect(eps.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(eps.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(eps.Spec.Nodes[0].Host).Should(Equal("127.0.0.1"))
		})

		It("Will get BkGatewayTLS from etcd", func() {
			tls := v1beta1.BkGatewayTLS{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "tls",
				},
				&tls,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(tls.Name).Should(Equal("tls"))
			Expect(tls.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(tls.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(tls.Spec.GatewayTLSSecretRef).Should(Equal("secret"))
		})

		It("Will get PluginMetadata from etcd", func() {
			pm := v1beta1.BkGatewayPluginMetadata{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "file-logger",
				},
				&pm,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(pm.Name).Should(Equal("file-logger"))
			Expect(pm.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(pm.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(pm.Spec.Name).Should(Equal("file-logger"))
		})

		It("Will get Secret by stage scope from etcd", func() {
			secret := v1.Secret{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo:    testStageInfo,
					ResourceName: "secret",
				},
				&secret,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(secret.Name).Should(Equal("secret"))
			Expect(secret.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(secret.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("stage"))
			Expect(secret.Data["ca.crt"]).ShouldNot(BeEmpty())
		})

		It("Will get Secret by gateway scope from etcd", func() {
			secret := v1.Secret{}
			err := testRegistry.Get(
				context.Background(),
				ResourceKey{
					StageInfo: StageInfo{
						GatewayName: "gateway",
						StageName:   "-",
					},
					ResourceName: "secret",
				},
				&secret,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(secret.Name).Should(Equal("secret"))
			Expect(secret.Labels[config.BKAPIGatewayLabelKeyGatewayName]).Should(Equal("gateway"))
			Expect(secret.Labels[config.BKAPIGatewayLabelKeyGatewayStage]).Should(Equal("-"))
			Expect(secret.Data["ca.crt"]).ShouldNot(BeEmpty())
		})
	})

	Context("List from ETCD", Ordered, func() {
		BeforeAll(func() {
			// List 空
			kv.On("Get", "/test/gateway/empty/v1beta1/BkGatewayResource/").Return(&clientv3.GetResponse{
				Count: 1,
				Kvs:   []*mvccpb.KeyValue{},
			}, nil)

			// List 1 个资源
			res := v1beta1.BkGatewayResource{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
						config.BKAPIGatewayLabelKeyGatewayStage: "one",
					},
				},
			}
			res.Name = "resource"
			res.Spec.Name = "test_resource"
			by, err := yaml.Marshal(res)
			Expect(err).ShouldNot(HaveOccurred())
			kv.On("Get", "/test/gateway/one/v1beta1/BkGatewayResource/").Return(&clientv3.GetResponse{
				Count: 1,
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("/test/gateway/one/v1beta1/BkGatewayResource/resource"),
						Value: by,
					},
				},
			}, nil)

			// List 多个资源
			res2 := v1beta1.BkGatewayResource{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
						config.BKAPIGatewayLabelKeyGatewayStage: "two",
					},
				},
			}
			res.Labels[config.BKAPIGatewayLabelKeyGatewayStage] = "two"
			res2.Name = "resource2"
			res2.Spec.Name = "test_resource2"
			by, err = yaml.Marshal(res)
			Expect(err).ShouldNot(HaveOccurred())
			by2, err := yaml.Marshal(res2)
			Expect(err).ShouldNot(HaveOccurred())
			kv.On("Get", "/test/gateway/two/v1beta1/BkGatewayResource/").Return(&clientv3.GetResponse{
				Count: 2,
				Kvs: []*mvccpb.KeyValue{
					{
						Key:   []byte("/test/gateway/two/v1beta1/BkGatewayResource/resource"),
						Value: by,
					},
					{
						Key:   []byte("/test/gateway/two/v1beta1/BkGatewayResource/resource2"),
						Value: by2,
					},
				},
			}, nil)
		})

		It("Will List BkGatewayResources with 0 items", func() {
			list := &v1beta1.BkGatewayResourceList{}
			err := testRegistry.List(context.Background(), ResourceKey{
				StageInfo: StageInfo{
					GatewayName: "gateway",
					StageName:   "empty",
				},
			}, list)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(list.Items)).Should(Equal(0))
		})

		It("Will List BkGatewayResources with 1 items", func() {
			list := &v1beta1.BkGatewayResourceList{}
			err := testRegistry.List(context.Background(), ResourceKey{
				StageInfo: StageInfo{
					GatewayName: "gateway",
					StageName:   "one",
				},
			}, list)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(list.Items)).Should(Equal(1))
		})
		It("Will List BkGatewayResources with multiple items", func() {
			list := &v1beta1.BkGatewayResourceList{}
			err := testRegistry.List(context.Background(), ResourceKey{
				StageInfo: StageInfo{
					GatewayName: "gateway",
					StageName:   "two",
				},
			}, list)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(list.Items)).Should(Equal(2))
		})
	})

	Context("ListStage", Ordered, func() {
		BeforeAll(func() {
			kv.On("Get", "/test/").Return(
				&clientv3.GetResponse{
					Count: 1,
					Kvs: []*mvccpb.KeyValue{
						{
							Key: []byte("/test/gateway/stage/v1beta1/BkGatewayStage/stage"),
						},
						{
							Key: []byte("/test/gateway/empty/v1beta1/BkGatewayStage/stage"),
						},
						{
							Key: []byte("/test/gateway/one/v1beta1/BkGatewayStage/stage"),
						},
						{
							Key: []byte("/test/gateway/two/v1beta1/BkGatewayStage/stage"),
						},
						{
							Key: []byte("/test/gateway/two/v1beta1/BkGatewayResource/resource"),
						},
					},
				}, nil)
		})

		It("Will list stage info from etcd", func() {
			siList, _ := testRegistry.ListStages(context.Background())
			Expect(siList).ShouldNot(BeNil())
			Expect(len(siList)).Should(Equal(4))
		})
	})

	Context("convertStage", Ordered, func() {
		var kvs []*mvccpb.KeyValue

		BeforeEach(func() {
			kvs = []*mvccpb.KeyValue{
				{
					Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource1"),
				},
				{
					Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource2"),
				},
				{
					Key: []byte("/test/gateway/stage2/v1beta1/BkGatewayResource/resource3"),
				},
				{
					Key: []byte("/test/gateway/stage2/v1beta1/BkGatewayResource/resource4"),
				},
			}
		})

		It("should return a list of unique stages", func() {
			stages := testRegistry.convertStages(kvs)
			Expect(len(stages)).To(Equal(2))
			for _, stage := range stages {
				Expect(stage.Key()).To(BeElementOf([]string{"gateway/stage", "gateway/stage2"}))
			}
		})
	})

	Context("Watch", Ordered, func() {
		var ch chan clientv3.WatchResponse
		var metaCh <-chan *ResourceMetadata

		BeforeAll(func() {
			ch = make(chan clientv3.WatchResponse)
			testWatcher := &mockWatcher{
				ch: &ch,
			}
			testRegistry.etcdClient.Watcher = testWatcher

			metaCh = testRegistry.Watch(context.Background())
			Expect(metaCh).NotTo(BeNil())

			go func() {
				//
				ch <- clientv3.WatchResponse{
					Header: pb.ResponseHeader{
						Revision: 1,
					},
					Events: []*clientv3.Event{
						{
							Type: clientv3.EventTypePut,
							Kv: &mvccpb.KeyValue{
								Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource"),
							},
						},
					},
				}

				ch <- clientv3.WatchResponse{
					Header: pb.ResponseHeader{
						Revision: 2,
					},
					Events: []*clientv3.Event{
						{
							Type: clientv3.EventTypeDelete,
							PrevKv: &mvccpb.KeyValue{
								Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource"),
							},
						},
					},
				}

				ch <- clientv3.WatchResponse{
					Header: pb.ResponseHeader{
						Revision: 3,
					},
					Events: []*clientv3.Event{
						{
							Type: clientv3.EventTypePut,
							Kv: &mvccpb.KeyValue{
								Key: []byte("/test/some/other/key"),
							},
						},
					},
				}

				ch <- clientv3.WatchResponse{
					Header: pb.ResponseHeader{
						Revision: 4,
					},
					Events: []*clientv3.Event{
						{
							Type: clientv3.EventTypeDelete,
							PrevKv: &mvccpb.KeyValue{
								Key: []byte("/test/some/other/key"),
							},
						},
					},
				}

				ch <- clientv3.WatchResponse{
					Header: pb.ResponseHeader{
						Revision: 5,
					},
					Events: []*clientv3.Event{
						{
							Type: clientv3.EventTypePut,
							Kv: &mvccpb.KeyValue{
								Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource"),
							},
						},
					},
				}
			}()
		})

		It("will receive put event", func() {
			rm := <-metaCh
			Expect(rm.StageInfo.GatewayName).Should(Equal(testStageInfo.GatewayName))
			Expect(rm.StageInfo.StageName).Should(Equal(testStageInfo.StageName))
			Expect(rm.Kind).Should(Equal(v1beta1.BkGatewayResourceTypeName))
			Expect(rm.APIVersion).Should(Equal("v1beta1"))
			Expect(rm.Name).Should(Equal("resource"))
			Expect(testRegistry.currentRevision).Should(Equal(int64(1)))
		})

		It("will receive delete event", func() {
			rm := <-metaCh
			Expect(rm.StageInfo.GatewayName).Should(Equal(testStageInfo.GatewayName))
			Expect(rm.StageInfo.StageName).Should(Equal(testStageInfo.StageName))
			Expect(rm.Kind).Should(Equal(v1beta1.BkGatewayResourceTypeName))
			Expect(rm.APIVersion).Should(Equal("v1beta1"))
			Expect(rm.Name).Should(Equal("resource"))
			<-time.After(time.Second)
			Expect(testRegistry.currentRevision).Should(Equal(int64(2)))
		})

		It("will skip event", func() {
			rm := <-metaCh
			Expect(rm.StageInfo.GatewayName).Should(Equal(testStageInfo.GatewayName))
			Expect(rm.StageInfo.StageName).Should(Equal(testStageInfo.StageName))
			Expect(rm.Kind).Should(Equal(v1beta1.BkGatewayResourceTypeName))
			Expect(rm.APIVersion).Should(Equal("v1beta1"))
			Expect(rm.Name).Should(Equal("resource"))
			<-time.After(time.Second)
			Expect(testRegistry.currentRevision).Should(Equal(int64(5)))
		})

		It("will break watch by compact", func() {
			ch <- clientv3.WatchResponse{
				CompactRevision: 5,
			}
			_, ok := <-metaCh
			Expect(ok).To(BeFalse())
			Expect(testRegistry.currentRevision).Should(Equal(int64(0)))

			metaCh = testRegistry.Watch(context.Background())
			ch <- clientv3.WatchResponse{
				Header: pb.ResponseHeader{
					Revision: 6,
				},
				Events: []*clientv3.Event{
					{
						Type: clientv3.EventTypePut,
						Kv: &mvccpb.KeyValue{
							Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource"),
						},
					},
				},
			}
			<-metaCh
			time.Sleep(time.Second)
			Expect(testRegistry.currentRevision).Should(Equal(int64(6)))
		})

		It("will auto recover from etcd channel break", func() {
			close(ch)
			ch = make(chan clientv3.WatchResponse)
			ch <- clientv3.WatchResponse{
				Header: pb.ResponseHeader{
					Revision: 7,
				},
				Events: []*clientv3.Event{
					{
						Type: clientv3.EventTypePut,
						Kv: &mvccpb.KeyValue{
							Key: []byte("/test/gateway/stage/v1beta1/BkGatewayResource/resource"),
						},
					},
				},
			}
			<-metaCh
			time.Sleep(time.Second)
			Expect(testRegistry.currentRevision).Should(Equal(int64(7)))
		})
	})
})
