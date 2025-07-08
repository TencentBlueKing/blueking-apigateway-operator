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
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	. "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
)

var _ = Describe("VirtualStage", func() {
	var (
		stage            *VirtualStage
		apisixHealthzURI string
		gatewayName      string
		stageName        string
		logPath          string
	)

	JustAfterEach(viper.Reset)

	BeforeEach(func() {
		apisixHealthzURI = "/healthz"
		gatewayName = "virtual-gateway"
		stageName = "virtual-stage"
		logPath = "/logs/access.log"

		Init(&config.Config{
			Apisix: config.Apisix{
				VirtualStage: config.VirtualStage{
					VirtualGateway:    gatewayName,
					VirtualStage:      stageName,
					FileLoggerLogPath: logPath,
				},
			},
		})
	})

	JustBeforeEach(func() {
		stage = NewVirtualStage(apisixHealthzURI)
	})

	checkLabels := func(labels map[string]string) {
		Expect(labels).To(HaveKeyWithValue(config.BKAPIGatewayLabelKeyGatewayName, gatewayName))
		Expect(labels).To(HaveKeyWithValue(config.BKAPIGatewayLabelKeyGatewayStage, stageName))
	}

	checkMetadata := func(metadata apisixv1.Metadata) {
		Expect(metadata.Name).To(Equal(metadata.ID))
		checkLabels(metadata.Labels)
	}

	Context("MakeConfiguration", func() {
		var configuration *apisix.ApisixConfiguration

		JustBeforeEach(func() {
			configuration = stage.MakeConfiguration()
		})

		Context("Standard Configuration", func() {
			It("should create 404 default route", func() {
				route := configuration.Routes[NotFoundHandling]
				checkMetadata(route.Metadata)

				Expect(route.Uri).To(Equal("/*"))
				Expect(route.Priority).To(Equal(-100))
				Expect(*route.Status).To(Equal(1))

				plugins := route.Plugins
				Expect(plugins).To(HaveKey("bk-error-wrapper"))
				Expect(plugins).To(HaveKey("bk-not-found-handler"))
				Expect(plugins["file-logger"]).To(HaveKeyWithValue("path", logPath))
			})

			It("should create outter healthz route", func() {
				route := configuration.Routes[HealthZRouteIDOuter]
				checkMetadata(route.Metadata)

				Expect(route.Uri).To(Equal(apisixHealthzURI))
				Expect(route.Priority).To(Equal(-100))
				Expect(route.Methods).To(ContainElement("GET"))
				Expect(*route.Status).To(Equal(1))

				plugins := route.Plugins
				Expect(plugins["limit-req"]).To(HaveKeyWithValue("key", "server_addr"))
				Expect(plugins["mocking"]).To(HaveKeyWithValue("response_example", "ok"))
			})
		})

		Context("Extra Configuration", func() {
			var (
				extraPath          string
				extraConfiguration *apisix.ApisixConfigurationStandalone
			)

			BeforeEach(func() {
				extraPath = filepath.Join(os.TempDir(), "extra-config.yaml")
				Init(&config.Config{
					Apisix: config.Apisix{
						VirtualStage: config.VirtualStage{
							VirtualGateway:       gatewayName,
							VirtualStage:         stageName,
							FileLoggerLogPath:    logPath,
							ExtraApisixResources: extraPath,
						},
					},
				})

				extraConfiguration = &apisix.ApisixConfigurationStandalone{}
			})

			AfterEach(func() {
				_ = os.Remove(extraPath)
			})

			writeExtraConfiguration := func() {
				buf := &bytes.Buffer{}

				encoder := yaml.NewEncoder(buf)
				Expect(encoder.Encode(&extraConfiguration)).To(BeNil())

				Expect(ioutil.WriteFile(extraPath, buf.Bytes(), 0o644)).To(BeNil())
			}

			It("should skip a not exists file", func() {
				config := stage.MakeConfiguration()
				Expect(len(config.Routes)).To(Equal(len(configuration.Routes)))
			})

			It("should skip a valid yaml", func() {
				Expect(ioutil.WriteFile(extraPath, []byte("not:a:yaml"), 0o644)).To(BeNil())

				config := stage.MakeConfiguration()
				Expect(len(config.Routes)).To(Equal(len(configuration.Routes)))
			})

			It("should not include invalid extra route", func() {
				extraConfiguration.Routes = []*apisix.Route{{
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID: "",
						},
					},
				}}
				writeExtraConfiguration()

				config := stage.MakeConfiguration()
				Expect(len(config.Routes)).To(Equal(len(configuration.Routes)))
			})

			It("should include valid extra route", func() {
				extraConfiguration.Routes = []*apisix.Route{{
					Route: apisixv1.Route{
						Metadata: apisixv1.Metadata{
							ID: "not-empty",
						},
					},
				}}
				writeExtraConfiguration()

				config := stage.MakeConfiguration()
				Expect(len(config.Routes)).To(Equal(len(configuration.Routes) + 1))

				for _, route := range configuration.Routes {
					checkMetadata(route.Metadata)
				}
			})

			It("should not include invalid extra service", func() {
				extraConfiguration.Services = []*apisix.Service{{
					Metadata: apisixv1.Metadata{
						ID: "",
					},
				}}
				writeExtraConfiguration()

				config := stage.MakeConfiguration()
				Expect(len(config.Services)).To(Equal(len(configuration.Services)))
			})

			It("should include valid extra service", func() {
				extraConfiguration.Services = []*apisix.Service{{
					Metadata: apisixv1.Metadata{
						ID: "not-empty",
					},
				}}
				writeExtraConfiguration()

				config := stage.MakeConfiguration()
				Expect(len(config.Services)).To(Equal(len(configuration.Services) + 1))

				for _, service := range configuration.Services {
					checkMetadata(service.Metadata)
				}
			})

			It("should not include invalid extra ssl", func() {
				extraConfiguration.SSLs = []*apisix.SSL{{
					Ssl: apisixv1.Ssl{
						ID: "",
					},
				}}
				writeExtraConfiguration()

				config := stage.MakeConfiguration()
				Expect(len(config.SSLs)).To(Equal(len(configuration.SSLs)))
			})

			It("should include valid extra ssl", func() {
				extraConfiguration.SSLs = []*apisix.SSL{{
					Ssl: apisixv1.Ssl{
						ID: "not-empty",
					},
				}}
				writeExtraConfiguration()

				config := stage.MakeConfiguration()
				Expect(len(config.SSLs)).To(Equal(len(configuration.SSLs) + 1))

				for _, ssl := range configuration.SSLs {
					checkLabels(ssl.Labels)
				}
			})
		})
	})
})
