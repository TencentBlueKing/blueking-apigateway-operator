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

package render

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Urirender", func() {
	var render Render

	BeforeEach(func() {
		render = GetURIRender()
	})

	Context("Normal Case", func() {
		var source string

		BeforeEach(func() {
			source = "/uri/with/{param1}/and/{env.key1}/{env.key_2}/"
		})

		It("does not have key_2", func() {
			Expect(
				render.Render(source, map[string]string{"key1": "value1"}),
			).To(Equal("/uri/with/:param1/and/value1/{env.key_2}/"))
		})

		It("does have key_2", func() {
			Expect(
				render.Render(source, map[string]string{"key1": "value1", "key_2": "value2"}),
			).To(Equal("/uri/with/:param1/and/value1/value2/"))
		})
	})

	Context("Empty Case", func() {
		It("does not have env and param", func() {
			Expect(
				render.Render("/uri/without/anything", map[string]string{"key1": "value1"}),
			).To(Equal("/uri/without/anything"))
		})

		It("is empty", func() {
			Expect(render.Render("", map[string]string{"key1": "value1"})).To(Equal(""))
		})

		It("is without env key", func() {
			Expect(render.Render("/uri/with/{env.}", map[string]string{"key1": "value1"})).To(Equal("/uri/with/{env.}"))
		})

		It("does not have env prefix (env.)", func() {
			Expect(
				render.Render("/uri/with/{boo.key1}", map[string]string{"key1": "value1"}),
			).To(Equal("/uri/with/{boo.key1}"))
		})
	})
})

var _ = Describe("UpstreamUrirender", func() {
	var render Render

	BeforeEach(func() {
		render = GetUpstreamURIRender()
	})

	Context("Normal Case", func() {
		var source string

		BeforeEach(func() {
			source = "/uri/with/{param1}/and/{env.key1}/{env.key_2}/"
		})

		It("does not have key_2", func() {
			Expect(
				render.Render(source, map[string]string{"key1": "value1"}),
			).To(Equal("/uri/with/${param1}/and/value1/{env.key_2}/"))
		})

		It("does have key_2", func() {
			Expect(
				render.Render(source, map[string]string{"key1": "value1", "key_2": "value2"}),
			).To(Equal("/uri/with/${param1}/and/value1/value2/"))
		})
	})

	Context("Empty Case", func() {
		It("does not have env and param", func() {
			Expect(
				render.Render("/uri/without/anything", map[string]string{"key1": "value1"}),
			).To(Equal("/uri/without/anything"))
		})

		It("is empty", func() {
			Expect(render.Render("", map[string]string{"key1": "value1"})).To(Equal(""))
		})

		It("is without env key", func() {
			Expect(render.Render("/uri/with/{env.}", map[string]string{"key1": "value1"})).To(Equal("/uri/with/{env.}"))
		})

		It("does not have env prefix (env.)", func() {
			Expect(
				render.Render("/uri/with/{boo.key1}", map[string]string{"key1": "value1"}),
			).To(Equal("/uri/with/{boo.key1}"))
		})
	})
})

func BenchmarkURIRenderer(b *testing.B) {
	render := GetURIRender()
	source := "/uri/with/{param1}/and/{env.key1}/{env.key_2}/{param2}/{env.3key}"
	env := map[string]string{
		"key1":  "value1",
		"key_2": "value2",
		"3key":  "value3",
	}
	for i := 0; i < b.N; i++ {
		render.Render(source, env)
	}
}
