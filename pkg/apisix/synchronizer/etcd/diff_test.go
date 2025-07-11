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

// Package etcd ...
package etcd

import (
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
)

var _ = Describe("configDiffer", func() {
	var differ *configDiffer
	metric.InitMetric(prometheus.DefaultRegisterer)
	Describe("diffSSLs", func() {
		var (
			newSSLs map[string]*apisix.SSL
			oldSSLs map[string]*apisix.SSL
		)
		BeforeEach(func() {
			differ = newConfigDiffer()
			newSSLs = map[string]*apisix.SSL{
				"test-ssl2": { // put
					Ssl: v1.Ssl{
						ID: "test-ssl2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
						},
					},
				},
				"test-ssl3": { // put
					Ssl: v1.Ssl{
						ID: "test-ssl3",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage3",
						},
					},
				},
				"test-ssl4": {
					Ssl: v1.Ssl{
						ID: "test-ssl3",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage4",
						},
					},
					CreateTime: 1,
				},
			}
			oldSSLs = map[string]*apisix.SSL{
				"test-ssl1": { // delete
					Ssl: v1.Ssl{
						ID: "test-ssl1",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
						},
					},
				},
				"test-ssl2": {
					Ssl: v1.Ssl{
						ID: "test-ssl2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stagexx",
						},
					},
				},
				"test-ssl4": {
					Ssl: v1.Ssl{
						ID: "test-ssl3",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage4",
						},
					},
					CreateTime: 2,
				},
			}
		})

		Context("Test diff ssl", func() {
			It("diff ssl", func() {
				put, del := differ.diffSSLs(oldSSLs, newSSLs)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffPluginMetadatas", func() {
		var (
			newPms map[string]*apisix.PluginMetadata
			oldPms map[string]*apisix.PluginMetadata
		)
		BeforeEach(func() {
			differ = newConfigDiffer()
			newPms = map[string]*apisix.PluginMetadata{
				"test-plugin1": apisix.NewPluginMetadata("test-plugin1", map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}),
				"test-plugin2": apisix.NewPluginMetadata("test-plugin2", map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}),
				"test-plugin4": apisix.NewPluginMetadata("test-plugin4", map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addrxx",
					},
				}),
			}

			oldPms = map[string]*apisix.PluginMetadata{
				"test-plugin1": apisix.NewPluginMetadata("test-plugin1", map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}),
				"test-plugin3": apisix.NewPluginMetadata("test-plugin3", map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}),
				"test-plugin4": apisix.NewPluginMetadata("test-plugin4", map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}),
			}
		})

		Context("Test diff PluginMetadata", func() {
			It("diff PluginMetadata", func() {
				put, del := differ.diffPluginMetadatas(newPms, oldPms)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffServices", func() {
		var (
			newServices map[string]*apisix.Service
			oldServices map[string]*apisix.Service
		)
		BeforeEach(func() {
			differ = newConfigDiffer()
			newServices = map[string]*apisix.Service{
				"test-svc1": {
					Metadata: v1.Metadata{
						ID: "test-svc1",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
						},
						Name: "test-stage1",
					},
					CreateTime: 1,
				},
				"test-svc2": {
					Metadata: v1.Metadata{
						ID: "test-svc2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
						},
						Name: "test-stage2",
					},
				},
				"test-svc4": {
					Metadata: v1.Metadata{
						ID: "test-svc4",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage4",
						},
						Name: "test-stage4",
					},
				},
			}

			oldServices = map[string]*apisix.Service{
				"test-svc1": {
					Metadata: v1.Metadata{
						ID: "test-svc1",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
						},
						Name: "test-stage1",
						Desc: "test-svc1",
					},
					CreateTime: 2,
				},
				"test-svc2": {
					Metadata: v1.Metadata{
						ID: "test-svc2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stagexx",
						},
						Name: "test-stagexx",
						Desc: "test-svc2",
					},
				},
				"test-svc3": {
					Metadata: v1.Metadata{
						ID: "test-svc3",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage3",
						},
						Name: "test-stage3",
						Desc: "test-svc3",
					},
				},
			}
		})

		Context("Test diff Services", func() {
			It("diff Services", func() {
				put, del := differ.diffServices(newServices, oldServices)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffRoutes", func() {
		var (
			newRoutes map[string]*apisix.Route
			oldRoutes map[string]*apisix.Route
		)
		BeforeEach(func() {
			differ = newConfigDiffer()
			newRoutes = map[string]*apisix.Route{
				"test-route1": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
							Name: "test-stage",
						},
					},
					CreateTime: 1,
				},
				"test-route2": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stagexx",
							},
							Name: "test-stagexx",
						},
					},
				},
				"test-route4": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
							Name: "test-stage",
						},
					},
				},
			}

			oldRoutes = map[string]*apisix.Route{
				"test-route1": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
							Name: "test-stage",
							Desc: "test-route1",
						},
					},
					CreateTime: 2,
				},
				"test-route2": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
							Name: "test-stage",
							Desc: "test-route2",
						},
					},
				},
				"test-route3": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
							Name: "test-stage",
						},
					},
				},
			}
		})

		Context("Test diff Routes", func() {
			It("diff Routes", func() {
				put, del := differ.diffRoutes(newRoutes, oldRoutes)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffRoutes with different type plugin value", func() {
		var (
			newRoutes map[string]*apisix.Route
			oldRoutes map[string]*apisix.Route
		)
		BeforeEach(func() {
			differ = newConfigDiffer()
			newRoutes = map[string]*apisix.Route{
				"test-route1": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
						},
						Plugins: map[string]interface{}{
							"proxy-rewrite": map[any]any{
								"uri": "/test/v1",
							},
							"response-rewrite": map[interface{}]any{
								"uri": "/test/v1",
							},
						},
					},
					CreateTime: 1,
				},
			}

			oldRoutes = map[string]*apisix.Route{
				"test-route1": {
					Route: v1.Route{
						Metadata: v1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
						},
						Plugins: map[string]interface{}{
							"proxy-rewrite": map[string]any{
								"uri": "/test/v1",
							},
							"response-rewrite": map[any]any{
								"uri": "/test/v1",
							},
						},
					},
					CreateTime: 2,
				},
			}
		})

		Context("Test diff Routes", func() {
			It("diff Routes", func() {
				put, del := differ.diffRoutes(newRoutes, oldRoutes)
				Expect(len(put)).To(Equal(0))
				Expect(len(del)).To(Equal(0))
			})
		})
	})

	Describe("diff", func() {
		var (
			newConf *apisix.ApisixConfiguration
			oldConf *apisix.ApisixConfiguration
		)
		BeforeEach(func() {
			differ = newConfigDiffer()
			newConf = &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"test-route1": {
						Route: v1.Route{
							Metadata: v1.Metadata{
								ID: "test-route1",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
								},
								Name: "test-stage",
								Desc: "test-route1",
							},
						},
					},
					"test-route2": {
						Route: v1.Route{
							Metadata: v1.Metadata{
								ID: "test-route2",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
								},
								Name: "test-stage",
								Desc: "test-route2",
							},
						},
					},
				},
				Services: map[string]*apisix.Service{
					"test-svc1": {
						Metadata: v1.Metadata{
							ID: "test-svc1",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
							},
							Name: "test-stage1",
						},
					},
					"test-svc2": {
						Metadata: v1.Metadata{
							ID: "test-svc2",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stagexx",
							},
							Name: "test-stagexx",
						},
					},
				},
				SSLs: map[string]*apisix.SSL{
					"test-ssl1": {
						Ssl: v1.Ssl{
							ID: "test-ssl1",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
							},
						},
					},
					"test-ssl2": {
						Ssl: v1.Ssl{
							ID: "test-ssl2",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
							},
						},
					},
				},
				PluginMetadatas: map[string]*apisix.PluginMetadata{
					"test-plugin1": apisix.NewPluginMetadata("test-plugin1", map[string]interface{}{
						"log-format": map[string]interface{}{
							"remote_addr": "$remote_addr",
						},
					}),
				},
			}

			oldConf = &apisix.ApisixConfiguration{
				Routes: map[string]*apisix.Route{
					"test-route1": {
						Route: v1.Route{
							Metadata: v1.Metadata{
								ID: "test-route1",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
								},
								Name: "test-stage",
							},
						},
					},
					"test-route3": {
						Route: v1.Route{
							Metadata: v1.Metadata{
								ID: "test-route3",
								Labels: map[string]string{
									config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
									config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
								},
								Name: "test-stage",
							},
						},
					},
				},
				Services: map[string]*apisix.Service{
					"test-svc1": {
						Metadata: v1.Metadata{
							ID: "test-svc1",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
							},
							Name: "test-stage1",
						},
					},
					"test-svc3": {
						Metadata: v1.Metadata{
							ID: "test-svc3",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stagexx",
							},
							Name: "test-stagexx",
						},
					},
				},
				SSLs: map[string]*apisix.SSL{
					"test-ssl1": {
						Ssl: v1.Ssl{
							ID: "test-ssl1",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage1",
							},
						},
					},
					"test-ssl3": {
						Ssl: v1.Ssl{
							ID: "test-ssl3",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
							},
						},
					},
				},
				PluginMetadatas: map[string]*apisix.PluginMetadata{
					"test-plugin1": apisix.NewPluginMetadata("test-plugin1", map[string]interface{}{
						"log-format": map[string]interface{}{
							"remote_addr": "$remote_addr",
						},
					}),
					"test-plugin2": apisix.NewPluginMetadata("test-plugin2", map[string]interface{}{
						"log-format": map[string]interface{}{
							"remote_addr": "$remote_addr",
						},
					}),
				},
			}
		})

		Context("Test diff", func() {
			It("diff", func() {
				put, del := differ.diff(oldConf, newConf)

				Expect(len(put.Routes)).To(Equal(1))

				Expect(len(put.Services)).To(Equal(1))

				Expect(len(put.SSLs)).To(Equal(1))

				Expect(len(put.PluginMetadatas)).To(Equal(0))

				Expect(len(del.Routes)).To(Equal(1))

				Expect(len(del.Services)).To(Equal(1))

				Expect(len(del.SSLs)).To(Equal(1))

				Expect(len(del.PluginMetadatas)).To(Equal(1))
			})
		})
	})
})
