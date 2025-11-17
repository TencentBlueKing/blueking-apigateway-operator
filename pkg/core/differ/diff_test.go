/*
 *  TencentBlueKing is pleased to support the open source community by making
 *  蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 *  Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 *  Licensed under the MIT License (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at
 *
 *      http://opensource.org/licenses/MIT
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under
 *  the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 *  either express or implied. See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 *   We undertake not to change the open source license (MIT license) applicable
 *   to the current version of the project delivered to anyone in the future.
 */

// Package etcd ...
package differ

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
)

func TestDiffer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Differ Suite")
}

var _ = Describe("configDiffer", func() {
	var differ *ConfigDiffer
	metric.InitMetric(prometheus.DefaultRegisterer)
	Describe("diffSSLs", func() {
		var (
			newSSLs map[string]*entity.SSL
			oldSSLs map[string]*entity.SSL
		)
		BeforeEach(func() {
			differ = NewConfigDiffer()
			newSSLs = map[string]*entity.SSL{
				"test-ssl2": { // put
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-ssl2",
						Kind: constant.SSL,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage2",
						},
					},
				},
				"test-ssl3": { // put
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-ssl3",
						Kind: constant.SSL,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage3",
						},
					},
				},
				"test-ssl4": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-ssl3",
						Kind: constant.SSL,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage4",
						},
					},
				},
			}
			oldSSLs = map[string]*entity.SSL{
				"test-ssl1": { // delete
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-ssl1",
						Kind: constant.SSL,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage1",
						},
					},
				},
				"test-ssl2": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-ssl2",
						Kind: constant.SSL,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stagexx",
						},
					},
				},
				"test-ssl4": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-ssl3",
						Kind: constant.SSL,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage4",
						},
					},
				},
			}
		})

		Context("Test diff ssl", func() {
			It("diff ssl", func() {
				put, del := differ.DiffSSLs(oldSSLs, newSSLs)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffPluginMetadatas", func() {
		var (
			newPms map[string]*entity.PluginMetadata
			oldPms map[string]*entity.PluginMetadata
		)
		BeforeEach(func() {
			differ = NewConfigDiffer()
			newPms = map[string]*entity.PluginMetadata{
				"test-plugin1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-plugin1",
						Kind: constant.PluginMetadata,
					},
					PluginMetadataConf: entity.PluginMetadataConf{
						"log-format": json.RawMessage(`{"remote_addr":"$remote_addr"}`),
					},
				},
				"test-plugin2": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-plugin2",
						Kind: constant.PluginMetadata,
					},
					PluginMetadataConf: entity.PluginMetadataConf{
						"log-format": json.RawMessage(`{"remote_addr":"$remote_addr"}`),
					},
				},
				"test-plugin4": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-plugin4",
						Kind: constant.PluginMetadata,
					},
					PluginMetadataConf: entity.PluginMetadataConf{
						"log-format": json.RawMessage(`{"remote_addr":"$remote_addrxx"}`),
					},
				},
			}

			oldPms = map[string]*entity.PluginMetadata{
				"test-plugin1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-plugin1",
						Kind: constant.PluginMetadata,
					},
					PluginMetadataConf: entity.PluginMetadataConf{
						"log-format": json.RawMessage(`{"remote_addr":"$remote_addr"}`),
					},
				},
				"test-plugin3": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-plugin3",
						Kind: constant.PluginMetadata,
					},
					PluginMetadataConf: entity.PluginMetadataConf{
						"log-format": json.RawMessage(`{"remote_addr":"$remote_addr"}`),
					},
				},
				"test-plugin4": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-plugin4",
						Kind: constant.PluginMetadata,
					},
					PluginMetadataConf: entity.PluginMetadataConf{
						"log-format": json.RawMessage(`{"remote_addr":"$remote_addr"}`),
					},
				},
			}
		})

		Context("Test diff PluginMetadata", func() {
			It("diff PluginMetadata", func() {
				put, del := differ.DiffPluginMetadatas(newPms, oldPms)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffServices", func() {
		var (
			newServices map[string]*entity.Service
			oldServices map[string]*entity.Service
		)
		BeforeEach(func() {
			differ = NewConfigDiffer()
			newServices = map[string]*entity.Service{
				"test-svc1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-svc1",
						Kind: constant.Service,
						Name: "test-stage1",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage1",
						},
					},
				},
				"test-svc2": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-svc2",
						Kind: constant.Service,
						Name: "test-stage2",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage2",
						},
					},
				},
				"test-svc4": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-svc4",
						Kind: constant.Service,
						Name: "test-stage4",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage4",
						},
					},
				},
			}

			oldServices = map[string]*entity.Service{
				"test-svc1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-svc1",
						Kind: constant.Service,
						Name: "test-stage1",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage1",
						},
					},
				},
				"test-svc2": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-svc2",
						Kind: constant.Service,
						Name: "test-stagexx",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stagexx",
						},
					},
				},
				"test-svc3": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-svc3",
						Kind: constant.Service,
						Name: "test-stage3",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage3",
						},
					},
				},
			}
		})

		Context("Test diff Services", func() {
			It("diff Services", func() {
				put, del := differ.DiffServices(newServices, oldServices)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffRoutes", func() {
		var (
			newRoutes map[string]*entity.Route
			oldRoutes map[string]*entity.Route
		)
		BeforeEach(func() {
			differ = NewConfigDiffer()
			newRoutes = map[string]*entity.Route{
				"test-route1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Name: "test-stage",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
						},
					},
				},
				"test-route2": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Name: "test-stagexx",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stagexx",
						},
					},
				},
				"test-route4": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Name: "test-stage",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
						},
					},
				},
			}

			oldRoutes = map[string]*entity.Route{
				"test-route1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Name: "test-stage",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
						},
					},
				},
				"test-route2": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Name: "test-stage",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
						},
					},
				},
				"test-route3": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Name: "test-stage",
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
						},
					},
				},
			}
		})

		Context("Test diff Routes", func() {
			It("diff Routes", func() {
				put, del := differ.DiffRoutes(newRoutes, oldRoutes)
				Expect(len(put)).To(Equal(2))
				Expect(len(del)).To(Equal(1))
			})
		})
	})

	Describe("diffRoutes with different type plugin value", func() {
		var (
			newRoutes map[string]*entity.Route
			oldRoutes map[string]*entity.Route
		)
		BeforeEach(func() {
			differ = NewConfigDiffer()
			newRoutes = map[string]*entity.Route{
				"test-route1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
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
			}

			oldRoutes = map[string]*entity.Route{
				"test-route1": {
					ResourceMetadata: entity.ResourceMetadata{
						ID:   "test-route",
						Kind: constant.Route,
						Labels: entity.LabelInfo{
							Gateway: "test-gateway",
							Stage:   "test-stage",
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
			}
		})

		Context("Test diff Routes", func() {
			It("diff Routes", func() {
				put, del := differ.DiffRoutes(newRoutes, oldRoutes)
				Expect(len(put)).To(Equal(0))
				Expect(len(del)).To(Equal(0))
			})
		})
	})

	Describe("diff", func() {
		var (
			newConf *entity.ApisixStageResource
			oldConf *entity.ApisixStageResource
		)
		BeforeEach(func() {
			differ = NewConfigDiffer()
			newConf = &entity.ApisixStageResource{
				Routes: map[string]*entity.Route{
					"test-route1": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-route1",
							Kind: constant.Route,
							Name: "test-stage",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage",
							},
						},
					},
					"test-route2": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-route2",
							Kind: constant.Route,
							Name: "test-stage",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage",
							},
						},
					},
				},
				Services: map[string]*entity.Service{
					"test-svc1": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-svc1",
							Kind: constant.Service,
							Name: "test-stage1",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage1",
							},
						},
					},
					"test-svc2": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-svc2",
							Kind: constant.Service,
							Name: "test-stagexx",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stagexx",
							},
						},
					},
				},
				SSLs: map[string]*entity.SSL{
					"test-ssl1": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-ssl1",
							Kind: constant.SSL,
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage1",
							},
						},
					},
					"test-ssl2": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-ssl2",
							Kind: constant.SSL,
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage2",
							},
						},
					},
				},
			}

			oldConf = &entity.ApisixStageResource{
				Routes: map[string]*entity.Route{
					"test-route1": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-route1",
							Kind: constant.Route,
							Name: "test-stage",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage",
							},
						},
					},
					"test-route3": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-route3",
							Kind: constant.Route,
							Name: "test-stage",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage",
							},
						},
					},
				},
				Services: map[string]*entity.Service{
					"test-svc1": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-svc1",
							Kind: constant.Service,
							Name: "test-stage1",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage1",
							},
						},
					},
					"test-svc3": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-svc3",
							Kind: constant.Service,
							Name: "test-stagexx",
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stagexx",
							},
						},
					},
				},
				SSLs: map[string]*entity.SSL{
					"test-ssl1": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-ssl1",
							Kind: constant.SSL,
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage1",
							},
						},
					},
					"test-ssl3": {
						ResourceMetadata: entity.ResourceMetadata{
							ID:   "test-ssl3",
							Kind: constant.SSL,
							Labels: entity.LabelInfo{
								Gateway: "test-gateway",
								Stage:   "test-stage2",
							},
						},
					},
				},
			}
		})

		Context("Test diff", func() {
			It("diff", func() {
				put, del := differ.Diff(oldConf, newConf)

				Expect(len(put.Routes)).To(Equal(1))

				Expect(len(put.Services)).To(Equal(1))

				Expect(len(put.SSLs)).To(Equal(1))

				Expect(len(del.Routes)).To(Equal(1))

				Expect(len(del.Services)).To(Equal(1))

				Expect(len(del.SSLs)).To(Equal(1))
			})
		})
	})
})
