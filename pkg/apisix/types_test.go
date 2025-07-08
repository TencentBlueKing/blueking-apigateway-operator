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

package apisix

import (
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	json "github.com/json-iterator/go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
)

var _ = Describe("ApisixConfiguration", func() {
	var conf1 *ApisixConfiguration

	Describe("configuration modification", func() {
		BeforeEach(func() {
			conf1 = NewEmptyApisixConfiguration()
			pluginmetaConfig := map[string]interface{}{
				"log-format": map[string]interface{}{
					"remote_addr": "$remote_addr",
				},
			}
			conf1.Routes["test-route"] = &Route{
				Route: apisixv1.Route{
					Metadata: apisixv1.Metadata{
						ID: "test-route",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
						},
					},
				},
			}
			conf1.Services["test-services"] = &Service{
				Metadata: apisixv1.Metadata{
					ID: "test-services",
					Labels: map[string]string{
						config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
						config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
					},
				},
			}
			conf1.SSLs["test-ssl"] = &SSL{
				Ssl: v1.Ssl{
					ID: "test-ssl",
					Labels: map[string]string{
						config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
						config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
					},
				},
			}
			conf1.PluginMetadatas["file-logger"] = NewPluginMetadata("file-logger", pluginmetaConfig)
		})

		Context("Test configuration merge", func() {
			var (
				conf2      *ApisixConfiguration
				confMerged *ApisixConfiguration
			)

			BeforeEach(func() {
				conf2 = NewEmptyApisixConfiguration()
				pluginmetaConfig := map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}
				conf2.Routes["test-route2"] = &Route{
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID: "test-route2",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
							},
						},
					},
				}
				conf2.Services["test-services2"] = &Service{
					Metadata: apisixv1.Metadata{
						ID: "test-services2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
						},
					},
				}
				conf2.SSLs["test-ssl2"] = &SSL{
					Ssl: v1.Ssl{
						ID: "test-ssl2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
						},
					},
				}
				conf2.PluginMetadatas["file-logger"] = NewPluginMetadata("file-logger", pluginmetaConfig)
			})

			BeforeEach(func() {
				confMerged = NewEmptyApisixConfiguration()
				pluginmetaConfig := map[string]interface{}{
					"log-format": map[string]interface{}{
						"remote_addr": "$remote_addr",
					},
				}
				confMerged.Routes["test-route2"] = &Route{
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID: "test-route2",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
							},
						},
					},
				}
				confMerged.Services["test-services2"] = &Service{
					Metadata: apisixv1.Metadata{
						ID: "test-services2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
						},
					},
				}
				confMerged.SSLs["test-ssl2"] = &SSL{
					Ssl: v1.Ssl{
						ID: "test-ssl2",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage2",
						},
					},
				}
				confMerged.Routes["test-route"] = &Route{
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID: "test-route",
							Labels: map[string]string{
								config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
								config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
							},
						},
					},
				}
				confMerged.Services["test-services"] = &Service{
					Metadata: apisixv1.Metadata{
						ID: "test-services",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
						},
					},
				}
				confMerged.SSLs["test-ssl"] = &SSL{
					Ssl: v1.Ssl{
						ID: "test-ssl",
						Labels: map[string]string{
							config.BKAPIGatewayLabelKeyGatewayName:  "test-gateway",
							config.BKAPIGatewayLabelKeyGatewayStage: "test-stage",
						},
					},
				}
				confMerged.PluginMetadatas["file-logger"] = NewPluginMetadata("file-logger", pluginmetaConfig)
			})

			It("merge from conf2 into conf1", func() {
				conf1.MergeFrom(conf2)
				Expect(conf1).To(Equal(confMerged))
			})
			It("merge from conf2 with conf1 into another conf", func() {
				originConf := conf1.DeepCopy()
				anotherConf := conf1.MergeCopy(conf2)
				Expect(anotherConf).To(Equal(confMerged))
				Expect(originConf).To(Equal(conf1))
			})
		})

		Context("Extract staged configuration", func() {
			It("extract test-gateway/test-stage from configuration", func() {
				extracted := conf1.ExtractStagedConfiguration("test-gateway/test-stage")
				Expect(extracted).NotTo(BeNil())
				Expect(extracted).NotTo(Equal(NewEmptyApisixConfiguration()))
				Expect(extracted.PluginMetadatas).To(Equal(make(map[string]*PluginMetadata)))
			})

			It("extract all staged configurations", func() {
				extracted := conf1.ToStagedConfiguration()
				staged, ok := extracted["test-gateway/test-stage"]
				Expect(ok).NotTo(BeFalse())
				Expect(staged).NotTo(BeNil())

				staged, ok = extracted[config.DefaultStageKey]
				Expect(ok).NotTo(BeFalse())
				Expect(staged).NotTo(BeNil())
				Expect(len(staged.PluginMetadatas)).NotTo(BeZero())
			})
		})
	})
})

var _ = Describe("PluginMetadata", func() {
	var (
		jsonStr string
		pm      *PluginMetadata
		err     error
	)

	Describe("Load from create handler", func() {
		BeforeEach(func() {
			cfg := map[string]interface{}{
				"log-format": map[string]interface{}{
					"remote_addr": "$remote_addr",
				},
			}
			pm = NewPluginMetadata("file-logger", cfg)
		})

		It("Should be non-nil", func() {
			Expect(pm).NotTo(Equal(nil))
		})

		It("Should have id file-logger", func() {
			Expect(pm.GetID()).To(Equal("file-logger"))
		})

		It("Should have stage key config.BKAPIGatewayStagedResourceKeyUnknown", func() {
			Expect(pm.GetStageFromLabel()).To(Equal(config.DefaultStageKey))
		})
	})

	Describe("Load from json object", func() {
		BeforeEach(func() {
			pm = &PluginMetadata{}
			jsonStr = `{"id":"file-logger", "log-format":{"remote_addr": "$remote_addr"}}`
			err = json.Unmarshal([]byte(jsonStr), pm)
		})

		It("Should not have error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should have id file-logger", func() {
			Expect(pm.GetID()).To(Equal("file-logger"))
		})

		It("Should have stage key config.BKAPIGatewayStagedResourceKeyUnknown", func() {
			Expect(pm.GetStageFromLabel()).To(Equal(config.DefaultStageKey))
		})
	})
})
