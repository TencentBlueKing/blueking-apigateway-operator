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

package integration_test

//
//const (
//	testGateway             = "integration-test"
//	testStage               = "prod"
//	testDataStageAmount     = 1
//	testDataServiceAmount   = 1
//	testDataRoutesAmount    = 75
//	testDataResourcesAmount = 74
//	versionRouteKey         = "integration-test-prod-apigw-builtin-mock-release-version"
//	operatorURL             = "http://127.0.0.1:6004"
//	updateVersionRouteValue = "metadata:\n  name: integration-test-prod-apigw-builtin-mock-release-version\n  labels:\n    gateway.bk.tencent.com/gateway: integration-test\n    gateway.bk.tencent.com/stage: prod\n  annotations: {}\nspec:\n  name: apigw_builtin__mock_release_version\n  desc: \u83b7\u53d6\u53d1\u5e03\u4fe1\u606f\uff0c\u7528\u4e8e\u68c0\u67e5\u7248\u672c\u53d1\u5e03\u7ed3\u679c\n  id: -1\n  plugins:\n  - name: bk-mock\n    config:\n      response_status: 200\n      response_example: '{\"publish_id\": 14, \"start_time\": \"2023-11-08 15:11:09+0800\"}'\n      response_headers:\n        Content-Type: application/json\n  service: ''\n  protocol: http\n  methods:\n  - GET\n  timeout:\n    connect: 60\n    read: 60\n    send: 60\n  uri: /__apigw_version\n  matchSubPath: false\n  upstream:\n    type: roundrobin\n    hashOn:\n    key:\n    checks:\n    scheme: http\n    retries:\n    retryTimeout:\n    passHost: node\n    upstreamHost:\n    tlsEnable: false\n    externalDiscoveryType:\n    externalDiscoveryConfig:\n    discoveryType:\n    serviceName:\n    nodes: []\n    timeout:\n  rewrite:\n    enabled: false\n    method:\n    path:\n    headers: {}\n    stageHeaders: append\n    serviceHeaders: append\n"
//	publishID               = 13
//)
////
////var _ = Describe("Operator Integration", func() {
////	time.Sleep(time.Second * 15)
////	var etcdCli *clientv3.Client
////	var resourceCli *client.ResourceClient
////	BeforeEach(func() {
////		cfg := clientv3.Config{
////			Endpoints:   []string{"localhost:2479"},
////			DialTimeout: 5 * time.Second,
////		}
////		var err error
////		etcdCli, err = clientv3.New(cfg)
////		Expect(err).NotTo(HaveOccurred())
////
////		resourceCli = client.NewResourceClient(operatorURL, "DebugModel@bk")
////	})
////
////	AfterEach(func() {
////		_, err := etcdCli.Delete(context.Background(), "", clientv3.WithPrefix())
////		Expect(err).NotTo(HaveOccurred())
////		_ = etcdCli.Close()
////	})
////
////	Describe("test publish httpbin resource", func() {
////		Context("test new agteway publish", func() {
////			It("should not error and the value should be equal to what was put", func() {
////				// load resources
////				resources := util.GetHTTPBinGatewayResource()
////				// put httpbin resources
////				for _, resource := range resources {
////					_, err := etcdCli.Put(context.Background(), resource.Key, resource.Value)
////					Expect(err).NotTo(HaveOccurred())
////				}
////
////				time.Sleep(time.Second * 10)
////
////				metricsAdapter, err := util.NewMetricsAdapter(operatorURL)
////
////				Expect(err).NotTo(HaveOccurred())
////
////				// assert bootstrap syncing count
////				Expect(metricsAdapter.GetBootstrapSyncingSuccessCountMetric(metric.ResultSuccess)).To(Equal(1))
////
////				// assert resource_event_triggered_count
////				Expect(metricsAdapter.GetResourceEventTriggeredCountMetric(
////					testGateway, testStage, constant.),
////				).To(Equal(testDataRoutesAmount))
////
////				Expect(metricsAdapter.GetResourceEventTriggeredCountMetric(
////					testGateway, testStage, v1beta1.BkGatewayServiceTypeName),
////				).To(Equal(testDataServiceAmount))
////
////				Expect(metricsAdapter.GetResourceEventTriggeredCountMetric(
////					testGateway, testStage, v1beta1.BkGatewayStageTypeName),
////				).To(Equal(testDataStageAmount))
////
////				// assert resource convert
////				Expect(metricsAdapter.GetResourceConvertedCountMetric(
////					testGateway, testStage, watcher.ApisixResourceTypeRoutes),
////				).To(Equal(testDataRoutesAmount))
////
////				Expect(metricsAdapter.GetResourceConvertedCountMetric(
////					testGateway, testStage, watcher.ApisixResourceTypeServices),
////				).To(Equal(testDataServiceAmount))
////
////				// assert apisix operation count
////				Expect(metricsAdapter.GetApisixOperationCountMetric(
////					metric.ActionPut, metric.ResultSuccess, watcher.ApisixResourceTypeRoutes),
////					// 2 micro-gateway-not-found-handling and healthz-outer
////				).To(Equal(testDataRoutesAmount + 2))
////
////				Expect(metricsAdapter.GetApisixOperationCountMetric(
////					metric.ActionPut, metric.ResultSuccess, watcher.ApisixResourceTypeServices),
////				).To(Equal(testDataServiceAmount))
////
////				// assert apigw resource
////				apigwGatewayResourcesMap, err := resourceCli.ApigwList(&client.ApigwListRequest{
////					GatewayName: testGateway,
////					StageName:   testStage,
////				})
////				Expect(err).NotTo(HaveOccurred())
////				resourceInfo, ok := apigwGatewayResourcesMap[testGateway+"/"+testStage]
////
////				Expect(ok).To(BeTrue())
////
////				Expect(len(resourceInfo.Routes)).To(Equal(testDataResourcesAmount))
////				Expect(len(resourceInfo.Services)).To(Equal(testDataServiceAmount))
////
////				// assert apigw resource count
////				apigwGatewayResourceCount, err := resourceCli.ApigwStageResourceCount(&client.ApigwListRequest{
////					GatewayName: testGateway,
////					StageName:   testStage,
////				})
////				Expect(err).NotTo(HaveOccurred())
////				Expect(apigwGatewayResourceCount.Count).To(Equal(int64(testDataRoutesAmount)))
////
////				// assert apigw current-version publish_id
////				apigwGatewayStageVersion, err := resourceCli.ApigwStageCurrentVersion(&client.ApigwListRequest{
////					GatewayName: testGateway,
////					StageName:   testStage,
////				})
////				Expect(err).NotTo(HaveOccurred())
////				res := client.ApigwListCurrentVersionInfoResponse{
////					"publish_id": float64(publishID),
////					"start_time": "2023-11-07 15:11:09+0800",
////				}
////				Expect(apigwGatewayStageVersion).To(Equal(res))
////			})
////		})
////	})
////})
