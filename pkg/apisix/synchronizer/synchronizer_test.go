// Package synchronizer_test ...
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

package synchronizer_test

import (
	"context"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	. "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer/etcd"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
	"github.com/TencentBlueKing/blueking-apigateway-operator/tests/util"
)

var _ = Describe("ApisixConfigSynchronizer", func() {
	var ac ApisixConfigSynchronizer
	var ctx context.Context
	var embedEtcd *embed.Etcd
	var err error
	var etcdClient *clientv3.Client
	var store *etcd.EtcdConfigStore
	var mockCtrl *gomock.Controller
	var aConfig *apisix.ApisixConfiguration

	metric.InitMetric(prometheus.DefaultRegisterer)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		etcdClient, embedEtcd, err = util.StartEmbedEtcdClient(ctx)
		Expect(err).NotTo(HaveOccurred())

		store, err = etcd.NewEtcdConfigStore(etcdClient, "test_prefix", 1*time.Second, 1*time.Second)

		Expect(err).NotTo(HaveOccurred())

		ac = NewSynchronizer(store, "test_url")

		ctx = context.Background()
		aConfig = &apisix.ApisixConfiguration{
			Routes: map[string]*apisix.Route{
				"gateway.prod.test-resource": {
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID:   "gateway.prod.test-resource",
							Name: "test-resource",
							Desc: "test resource",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "prod",
							},
						},
						Host:    "test.exmaple.com",
						Uris:    []string{"/test-resource", "/test-resource/"},
						Methods: []string{"GET"},
						Timeout: &apisixv1.UpstreamTimeout{
							Connect: 1,
							Read:    1,
							Send:    1,
						},
					},
					Status: utils.IntPtr(1),
					Upstream: &apisix.Upstream{
						Type: utils.StringPtr("roundrobin"),
						Nodes: []v1beta1.BkGatewayNode{
							{
								Host:     "127.0.0.1",
								Port:     9090,
								Weight:   10,
								Priority: utils.IntPtr(-1),
							},
						},
					},
				},
			},
			StreamRoutes: map[string]*apisix.StreamRoute{
				"gateway.stage.test-stream-resource": {
					Metadata: apisixv1.Metadata{
						ID:   "gateway.stage.test-stream-resource",
						Desc: "test resource",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "prod",
						},
					},
					Status:     utils.IntPtr(1),
					ServerPort: 8080,
					SNI:        "test.example.com",
					Upstream: &apisix.Upstream{
						Type: utils.StringPtr("roundrobin"),
						Nodes: []v1beta1.BkGatewayNode{
							{
								Host:     "127.0.0.1",
								Port:     9090,
								Weight:   10,
								Priority: utils.IntPtr(-1),
							},
						},
					},
				},
			},
			Services:        make(map[string]*apisix.Service),
			PluginMetadatas: make(map[string]*apisix.PluginMetadata),
			SSLs:            make(map[string]*apisix.SSL),
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
		etcdClient.Close()
		// Shutdown the embedded etcd server.
		embedEtcd.Close()
		// Remove the etcd data directory
		_ = os.RemoveAll(embedEtcd.Config().Dir)
	})

	Describe("Testing Sync method And Flush", func() {
		Context("with valid input", func() {
			It("should sync new staged apisix configuration correctly", func() {
				gatewayName := "gateway"
				stageName := "prod"
				err := ac.Sync(ctx, gatewayName, stageName, aConfig)
				Expect(err).Should(BeNil())
				time.Sleep(time.Second * 5)

				apisixConfig := store.Get("gateway/prod")

				result := cmp.Equal(
					apisixConfig, aConfig,
					cmpopts.IgnoreFields(apisixv1.Metadata{}, "Desc", "Labels"),
				)
				Expect(result).Should(BeTrue())
			})
		})
	})
})
