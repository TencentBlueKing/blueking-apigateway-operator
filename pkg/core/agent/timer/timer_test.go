// Package timer ...
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

package timer

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
)

var _ = Describe("Timer", func() {
	var (
		stageTimer *ResourceTimer
		stageInfo  entity.ReleaseInfo
	)

	BeforeEach(func() {
		stageTimer = NewResourceTimer()
		stageInfo = entity.ReleaseInfo{
			ResourceMetadata: entity.ResourceMetadata{
				Labels: entity.LabelInfo{
					Gateway:       "gateway",
					Stage:         "stage",
					PublishId:     "1",
					ApisixVersion: "2.13.1",
				},
				ID: "gateway-stage",
			},

			PublishId:       1,
			PublishTime:     "2023-01-01T00:00:00Z",
			ApisixVersion:   "2.13.1",
			ResourceVersion: "1",
		}

		eventsWaitingTimeWindow = 100 * time.Millisecond
	})

	AfterEach(func() {
		eventsWaitingTimeWindow = 2 * time.Second
	})

	It("should update the stage timer correctly", func() {
		stageTimer.Update(&stageInfo)
		stageTimer.Update(&stageInfo)
		stageList := stageTimer.ListResourcesForCommit()
		// no sleep for exceeding 100ms (eventsWaitingTimeWindow)
		gomega.Expect(stageList).To(gomega.HaveLen(0))
	})

	It("should list stages for commit correctly", func() {
		stageTimer.Update(&stageInfo)
		stageTimer.Update(&stageInfo)

		time.Sleep(200 * time.Millisecond)

		stageList := stageTimer.ListResourcesForCommit()
		gomega.Expect(stageList).To(gomega.HaveLen(1))
		gomega.Expect(stageList[0].ID).To(gomega.Equal(stageInfo.ID))
	})
})
