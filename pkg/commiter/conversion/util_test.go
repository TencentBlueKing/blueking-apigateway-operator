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

package conversion

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
)

func resetVars() {
	useUriGatewayPrefix = false
	gatewayResourceBasePrefix = ""
}

var _ = Describe("Util", func() {
	Context("getOptionalUriGatewayPrefix", func() {
		var cvt *Converter

		BeforeEach(func() {
			var err error
			cvt, err = NewConverter("", "gateway", &v1beta1.BkGatewayStage{
				Spec: v1beta1.BkGatewayStageSpec{
					Name: "stage",
				},
			}, nil, nil)
			Expect(err).To(BeNil())
			resetVars()
		})

		AfterEach(func() {
			resetVars()
		})

		It("does not have prefix", func() {
			Expect(cvt.getOptionalUriGatewayPrefix()).To(Equal(""))
		})

		It("agent mode has prefix", func() {
			Init(&config.Config{Operator: config.Operator{AgentMode: true}})
			Expect(cvt.getOptionalUriGatewayPrefix()).To(Equal("/gateway/stage"))
		})

		It("set gateway prefix should be prepend", func() {
			Init(&config.Config{Dashboard: config.Dashboard{PrefixPrepend: true}})
			Expect(cvt.getOptionalUriGatewayPrefix()).To(Equal("/gateway/stage"))
		})

		It("base prefix should be prepended before gateway prefix", func() {
			Init(&config.Config{Dashboard: config.Dashboard{PrefixPrepend: true, ResourceBasePrefix: "/api"}})
			Expect(cvt.getOptionalUriGatewayPrefix()).To(Equal("/api/gateway/stage"))
		})
	})
})

var _ = Describe("dependencies test", func() {
	Context("url", func() {
		It("should succ", func() {
			host, port, err := net.SplitHostPort("127.0.0.1:8080")
			Expect(err).NotTo(HaveOccurred())
			Expect(host).To(Equal("127.0.0.1"))
			Expect(port).To(Equal("8080"))
		})

		It("should succ", func() {
			_, _, err := net.SplitHostPort("127.0.0.1")
			Expect(err).To(HaveOccurred())
		})
	})
})
