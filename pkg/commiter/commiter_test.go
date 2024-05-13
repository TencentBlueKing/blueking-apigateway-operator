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

package commiter

import (
	"context"
	"time"

	sm "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	synchronizerMock "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer/mock"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/eventreporter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	registryMock "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry/mock"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Commiter", func() {
	var ctl *gomock.Controller

	var mockRegister *registryMock.MockRegistry
	var mockSynchronizer *synchronizerMock.MockApisixConfigSynchronizer
	var stageTimer *timer.StageTimer

	var commiter *Commiter

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())

		mockRegister = registryMock.NewMockRegistry(ctl)
		mockSynchronizer = synchronizerMock.NewMockApisixConfigSynchronizer(ctl)
		stageTimer = timer.NewStageTimer()

		commiter = NewCommiter(mockRegister, mockSynchronizer, radixtree.NewSingleRadixTreeGetter(), stageTimer, nil)

		eventreporter.InitReporter(&config.Config{
			EventReporter: config.EventReporter{
				VersionProbe: config.VersionProbe{
					BufferSize: 100,
					Retry: config.Retry{
						Count:    60,
						Interval: time.Second,
					},
					Timeout:  time.Minute * 2,
					WaitTime: time.Second * 15,
				},
				EventBufferSize:    300,
				ReporterBufferSize: 100,
			},
		})
	})

	It("ConvertEtcdKVToApisixConfiguration", func() {
		patchGuard := sm.Patch((*Commiter).listResources, func(
			_ *Commiter, ctx context.Context,
			stageInfo registry.StageInfo,
		) ([]*v1beta1.BkGatewayResource, error) {
			return []*v1beta1.BkGatewayResource{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-resource",
						Namespace: "test",
					},
					Spec: v1beta1.BkGatewayResourceSpec{
						Desc:    "test resource",
						Methods: []string{"GET"},
						Timeout: &v1beta1.UpstreamTimeout{
							Connect: v1beta1.FormatDuration(time.Second),
							Send:    v1beta1.FormatDuration(time.Second),
							Read:    v1beta1.FormatDuration(time.Second),
						},
						URI:          "/test-resource",
						MatchSubPath: false,
						Upstream: &v1beta1.BkGatewayUpstreamConfig{
							Type: "roundrobin",
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "127.0.0.1:9090",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
						},
					},
				},
			}, nil
		})
		defer patchGuard.Unpatch()

		patchGuard = sm.Patch((*Commiter).listStreamResources, func(
			_ *Commiter, ctx context.Context,
			stageInfo registry.StageInfo,
		) ([]*v1beta1.BkGatewayStreamResource, error) {
			return []*v1beta1.BkGatewayStreamResource{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-stream-resource",
						Namespace: "test",
					},
					Spec: v1beta1.BkGatewayStreamResourceSpec{
						Desc: "test stream resource",
						Upstream: &v1beta1.BkGatewayUpstreamConfig{
							Type: "roundrobin",
							Nodes: []v1beta1.BkGatewayNode{
								{
									Host:     "127.0.0.1:9090",
									Port:     8080,
									Weight:   10,
									Priority: utils.IntPtr(-1),
								},
							},
						},
					},
				},
			}, nil
		})
		defer patchGuard.Unpatch()

		patchGuard = sm.Patch((*Commiter).getStage, func(
			_ *Commiter, ctx context.Context,
			stageInfo registry.StageInfo,
		) (*v1beta1.BkGatewayStage, error) {
			return &v1beta1.BkGatewayStage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "stage",
				},
				Spec: v1beta1.BkGatewayStageSpec{
					Name:       "stage",
					Domain:     "test.exmaple.com",
					PathPrefix: "/",
					Desc:       "test desc",
					Vars: map[string]string{
						"runMode": "prod",
					},
					Rewrite: &v1beta1.BkGatewayRewrite{
						Enabled: true,
						Headers: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
					Plugins: []*v1beta1.BkGatewayPlugin{
						{
							Name: "limit-req",
							Config: runtime.RawExtension{
								Raw: []byte("{\"rate\":1,\"burst\":2,\"rejected_code\":429,\"key\":\"consumer_name\"}"),
							},
						},
					},
				},
			}, nil
		})
		defer patchGuard.Unpatch()

		patchGuard = sm.Patch((*Commiter).listServices, func(
			_ *Commiter, ctx context.Context,
			stageInfo registry.StageInfo,
		) ([]*v1beta1.BkGatewayService, error) {
			return []*v1beta1.BkGatewayService{}, nil
		})
		defer patchGuard.Unpatch()

		patchGuard = sm.Patch((*Commiter).listPluginMetadatas, func(
			_ *Commiter, ctx context.Context,
			stageInfo registry.StageInfo,
		) ([]*v1beta1.BkGatewayPluginMetadata, error) {
			return []*v1beta1.BkGatewayPluginMetadata{}, nil
		})
		defer patchGuard.Unpatch()

		patchGuard = sm.Patch((*Commiter).listSSLs, func(
			_ *Commiter,
			ctx context.Context,
			stageInfo registry.StageInfo,
		) ([]*v1beta1.BkGatewayTLS, error) {
			return []*v1beta1.BkGatewayTLS{}, nil
		})
		defer patchGuard.Unpatch()

		conf, stage, err := commiter.ConvertEtcdKVToApisixConfiguration(
			context.TODO(),
			registry.StageInfo{GatewayName: "gateway", StageName: "stage"},
		)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(conf).Should(gomega.Equal(&apisix.ApisixConfiguration{
			Routes: map[string]*apisix.Route{
				"gateway.stage.test-resource": {
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID:   "gateway.stage.test-resource",
							Name: "test-resource",
							Desc: "test resource",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "stage",
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
						Name: "test-stream-resource",
						Desc: "test stream resource",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "stage",
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
			Services:        make(map[string]*apisix.Service),
			PluginMetadatas: make(map[string]*apisix.PluginMetadata),
			SSLs:            make(map[string]*apisix.SSL),
		}))
		gomega.Expect(stage).Should(gomega.Equal(&v1beta1.BkGatewayStage{
			TypeMeta: metav1.TypeMeta{Kind: "", APIVersion: ""},
			ObjectMeta: metav1.ObjectMeta{
				Name: "stage",
			},
			Spec: v1beta1.BkGatewayStageSpec{
				Name:       "stage",
				Domain:     "test.exmaple.com",
				PathPrefix: "/",
				Desc:       "test desc",
				Vars:       map[string]string{"runMode": "prod"},
				Rewrite: &v1beta1.BkGatewayRewrite{
					Enabled: true,
					Headers: map[string]string{"key1": "value1", "key2": "value2"},
				},
				Plugins: []*v1beta1.BkGatewayPlugin{
					{
						Name: "limit-req",
						Config: runtime.RawExtension{
							Raw: []byte("{\"rate\":1,\"burst\":2,\"rejected_code\":429,\"key\":\"consumer_name\"}"),
						},
					},
				},
			},
			Status: v1beta1.BkGatewayStageStatus{},
		}))
	})
})
