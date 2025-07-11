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

// Package registry ...
package registry

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test StageInfo", func() {
	It("Test IsEmpty", func() {
		info := StageInfo{}
		Expect(info.IsEmpty()).To(BeTrue())

		info.GatewayName = "gw"
		info.StageName = "stage"
		Expect(info.IsEmpty()).To(BeFalse())
	})
})

var _ = Describe("Test ResourceMetadata.IsEmpty", func() {
	It("Test nil object", func() {
		var mdata *ResourceMetadata
		Expect(mdata.IsEmpty()).To(BeTrue())
	})

	It("Test object with empty values", func() {
		mdata := &ResourceMetadata{}
		Expect(mdata.IsEmpty()).To(BeTrue())
	})

	It("Test normal", func() {
		mdata := &ResourceMetadata{
			StageInfo: StageInfo{
				GatewayName: "gw",
				StageName:   "stage",
			},
		}
		Expect(mdata.IsEmpty()).To(BeFalse())
	})
})
