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

package integration_test

import (
	"context"
	"strings"
	"time"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer/etcd"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/client"
	util "github.com/TencentBlueKing/blueking-apigateway-operator/tests/integration"
)

const (
	testGateway             = "integration-test"
	testStage               = "prod"
	testDataStageAmount     = 1
	testDataServiceAmount   = 1
	testDataRoutesAmount    = 75
	versionRouteKey         = "integration-test-prod-apigw-builtin-mock-release-version"
	updateVersionRouteValue = "metadata:\n  name: integration-test-prod-apigw-builtin-mock-release-version\n  labels:\n    gateway.bk.tencent.com/gateway: integration-test\n    gateway.bk.tencent.com/stage: prod\n  annotations: {}\nspec:\n  name: apigw_builtin__mock_release_version\n  desc: \u83b7\u53d6\u53d1\u5e03\u4fe1\u606f\uff0c\u7528\u4e8e\u68c0\u67e5\u7248\u672c\u53d1\u5e03\u7ed3\u679c\n  id: -1\n  plugins:\n  - name: bk-mock\n    config:\n      response_status: 200\n      response_example: '{\"publish_id\": 14, \"start_time\": \"2023-11-08 15:11:09+0800\"}'\n      response_headers:\n        Content-Type: application/json\n  service: ''\n  protocol: http\n  methods:\n  - GET\n  timeout:\n    connect: 60\n    read: 60\n    send: 60\n  uri: /__apigw_version\n  matchSubPath: false\n  upstream:\n    type: roundrobin\n    hashOn:\n    key:\n    checks:\n    scheme: http\n    retries:\n    retryTimeout:\n    passHost: node\n    upstreamHost:\n    tlsEnable: false\n    externalDiscoveryType:\n    externalDiscoveryConfig:\n    discoveryType:\n    serviceName:\n    nodes: []\n    timeout:\n  rewrite:\n    enabled: false\n    method:\n    path:\n    headers: {}\n    stageHeaders: append\n    serviceHeaders: append\n"
)

var _ = Describe("Operator Integration", func() {
	time.Sleep(time.Second * 15)
	var etcdCli *clientv3.Client
	var resourceCli *client.ResourceClient
	BeforeEach(func() {
		cfg := clientv3.Config{
			Endpoints:   []string{"localhost:2479"},
			DialTimeout: 5 * time.Second,
		}
		var err error
		etcdCli, err = clientv3.New(cfg)
		Expect(err).NotTo(HaveOccurred())

		resourceCli = client.NewResourceClient("http://127.0.0.1:6004", "DebugModel@bk")
	})

	AfterEach(func() {
		_, err := etcdCli.Delete(context.Background(), "", clientv3.WithPrefix())
		Expect(err).NotTo(HaveOccurred())
		_ = etcdCli.Close()
	})

	Describe("test publish httpbin resource", func() {
		Context("test new agteway publish", func() {
			It("should not error and the value should be equal to what was put", func() {
				//load resources
				resources := util.GetHttpBinGatewayResource()
				//put httpbin resources
				for _, resource := range resources {
					_, err := etcdCli.Put(context.Background(), resource.Key, resource.Value)
					Expect(err).NotTo(HaveOccurred())
				}

				time.Sleep(time.Second * 10)

				metrics, err := util.GetAllMetrics()

				Expect(err).NotTo(HaveOccurred())

				// assert bootstrap syncing count
				Expect(int(util.GetBootstrapSyncingSuccessCountMetric(metrics))).To(Equal(1))

				// assert resource_event_triggered_count
				Expect(int(util.GetResourceEventTriggeredCountMetric(
					metrics, testGateway, testStage, v1beta1.BkGatewayResourceTypeName),
				)).To(Equal(testDataRoutesAmount))

				Expect(int(util.GetResourceEventTriggeredCountMetric(
					metrics, testGateway, testStage, v1beta1.BkGatewayServiceTypeName),
				)).To(Equal(testDataServiceAmount))

				Expect(int(util.GetResourceEventTriggeredCountMetric(
					metrics, testGateway, testStage, v1beta1.BkGatewayStageTypeName),
				)).To(Equal(testDataStageAmount))

				// assert resource convert
				Expect(int(util.GetResourceConvertedCountMetric(
					metrics, testGateway, testStage, v1beta1.BkGatewayResourceTypeName),
				)).To(Equal(testDataRoutesAmount))

				Expect(int(util.GetResourceConvertedCountMetric(
					metrics, testGateway, testStage, v1beta1.BkGatewayServiceTypeName),
				)).To(Equal(testDataServiceAmount))

				// assert apisix operation count
				Expect(int(util.GetApisixOperationCountMetric(
					metrics, metric.ActionPut, metric.ResultSuccess, etcd.ApisixResourceTypeRoutes),
				// 2 micro-gateway-not-found-handling and healthz-outer
				)).To(Equal(testDataRoutesAmount + 2))

				Expect(int(util.GetApisixOperationCountMetric(
					metrics, metric.ActionPut, metric.ResultSuccess, etcd.ApisixResourceTypeServices),
				)).To(Equal(testDataServiceAmount))

				// assert apigw resource and apisix resource
				gatewayResourcesMap, err := resourceCli.List(&client.ListReq{
					Gateway: testGateway,
					Stage:   testStage,
				})
				Expect(err).NotTo(HaveOccurred())

				resourceInfo, ok := gatewayResourcesMap[testGateway+"/"+testStage]

				Expect(ok).To(BeTrue())

				Expect(len(resourceInfo.Routes)).To(Equal(testDataRoutesAmount))
				Expect(len(resourceInfo.Services)).To(Equal(testDataServiceAmount))

				// assert apigw resource diff apisix resource
				diffResourceResult, err := resourceCli.Diff(&client.DiffReq{
					Gateway: testGateway,
					Stage:   testStage,
					All:     true,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(*diffResourceResult)).To(Equal(0))

			})
		})

		Context("test update publish", func() {
			It("should not error and the value should be equal to what was put", func() {
				//load base resources
				resources := util.GetHttpBinGatewayResource()
				//put httpbin resources
				var versionRoute util.EtcdConfig
				for _, resource := range resources {
					if strings.Contains(resource.Key, versionRouteKey) {
						versionRoute = resource
					}
					_, err := etcdCli.Put(context.Background(), resource.Key, resource.Value)
					Expect(err).NotTo(HaveOccurred())
				}

				time.Sleep(time.Second * 10)

				// update version routes
				versionRoute.Value = updateVersionRouteValue
				_, err := etcdCli.Put(context.Background(), versionRoute.Key, versionRoute.Value)
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(time.Second * 10)

				metrics, err := util.GetAllMetrics()

				Expect(err).NotTo(HaveOccurred())

				// assert sync
				Expect(int(util.GetResourceSyncCmpCountMetrics(
					metrics,
					testGateway,
					testStage,
					etcd.ApisixResourceTypeRoutes),
				)).To(Equal(testDataRoutesAmount))

				Expect(int(util.GetResourceSyncCmpCountMetrics(
					metrics,
					testGateway,
					testStage,
					etcd.ApisixResourceTypeServices),
				)).To(Equal(testDataServiceAmount))

				// diff
				Expect(int(util.GetResourceSyncCmpDiffCountMetrics(
					metrics,
					testGateway,
					testStage,
					etcd.ApisixResourceTypeRoutes),
				)).To(Equal(1))

				Expect(int(util.GetResourceSyncCmpDiffCountMetrics(
					metrics,
					testGateway,
					testStage,
					etcd.ApisixResourceTypeServices),
				)).To(Equal(0))

				// assert apigw resource and apisix resource
				gatewayResourcesMap, err := resourceCli.List(&client.ListReq{
					Gateway: testGateway,
					Stage:   testStage,
				})
				Expect(err).NotTo(HaveOccurred())

				resourceInfo, ok := gatewayResourcesMap[testGateway+"/"+testStage]

				Expect(ok).To(BeTrue())

				Expect(len(resourceInfo.Routes)).To(Equal(testDataRoutesAmount))
				Expect(len(resourceInfo.Services)).To(Equal(testDataServiceAmount))

				// assert apigw resource diff apisix resource
				diffResourceResult, err := resourceCli.Diff(&client.DiffReq{
					Gateway: testGateway,
					Stage:   testStage,
					All:     true,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(*diffResourceResult)).To(Equal(0))

			})
		})

	})
})
