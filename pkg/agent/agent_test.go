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

package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	sm "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/agent/timer"
	synchronizerMock "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer/mock"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
	radixMock "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree/mock"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	registryMock "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry/mock"
)

var _ = Describe("Agent", func() {
	var ctl *gomock.Controller

	var mockRegister *registryMock.MockRegistry
	var commitCh chan []registry.StageInfo
	var mockSynchronizer *synchronizerMock.MockApisixConfigSynchronizer
	var mockRadixTreeGetter *radixMock.MockRadixTreeGetter
	var stageTimer *timer.StageTimer

	var agent *EventAgent

	BeforeEach(func() {
		ctl = gomock.NewController(GinkgoT())

		mockRegister = registryMock.NewMockRegistry(ctl)
		commitCh = make(chan []registry.StageInfo, 100)
		mockSynchronizer = synchronizerMock.NewMockApisixConfigSynchronizer(ctl)
		mockRadixTreeGetter = radixMock.NewMockRadixTreeGetter(ctl)
		stageTimer = timer.NewStageTimer()

		agent = NewEventAgent(
			mockRegister,
			commitCh,
			mockSynchronizer,
			mockRadixTreeGetter,
			stageTimer,
		)
		agent.SetKeepAliveChan(make(chan struct{}))

		timer.Init(&config.Config{
			Operator: config.Operator{
				AgentEventsWaitingTimeWindow: 100 * time.Microsecond,
				AgentForceUpdateTimeWindow:   10 * time.Second,
			},
		})
	})

	AfterEach(func() {
		timer.Init(&config.Config{
			Operator: config.Operator{
				AgentEventsWaitingTimeWindow: 2 * time.Second,
				AgentForceUpdateTimeWindow:   10 * time.Second,
			},
		})
	})

	It("bootstrapSync two stages", func() {
		mockRegister.EXPECT().ListStages(gomock.Any()).Return([]registry.StageInfo{
			{
				GatewayName: "gateway",
				StageName:   "stage1",
				Ctx:         nil,
			},
			{
				GatewayName: "gateway",
				StageName:   "stage2",
				Ctx:         nil,
			},
		}, nil)
		mockSynchronizer.EXPECT().RemoveNotExistStage(gomock.Any(), gomock.Any()).Return(nil)
		mockRadixTreeGetter.EXPECT().RemoveNotExistStage(gomock.Any()).Return()

		err := agent.bootstrapSync(context.Background())
		gomega.Expect(err).To(gomega.BeNil())

		stageList := <-commitCh
		gomega.Expect(stageList).To(gomega.HaveLen(2))
		gomega.Expect(stageList[0].Key()).To(gomega.Equal("gateway/stage1"))
	})

	It("createWatchChannel two stages", func() {
		watchCh := make(chan *registry.ResourceMetadata, 100)
		mockRegister.EXPECT().Watch(gomock.Any()).Return(watchCh)
		mockRegister.EXPECT().ListStages(gomock.Any()).Return([]registry.StageInfo{
			{
				GatewayName: "gateway",
				StageName:   "stage1",
				Ctx:         nil,
			},
			{
				GatewayName: "gateway",
				StageName:   "stage2",
				Ctx:         nil,
			},
		}, nil)
		mockSynchronizer.EXPECT().RemoveNotExistStage(gomock.Any(), gomock.Any()).Return(nil)
		mockRadixTreeGetter.EXPECT().RemoveNotExistStage(gomock.Any()).Return()

		agent.createWatchChannel(context.Background())

		stageList := <-commitCh
		gomega.Expect(stageList).To(gomega.HaveLen(2))
		gomega.Expect(stageList[0].Key()).To(gomega.Equal("gateway/stage1"))
	})

	Describe("handleEvent", func() {
		It("normal event", func() {
			agent.handleEvent(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       v1beta1.BkGatewayResourceTypeName,
			})

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			gomega.Expect(stageList).To(gomega.HaveLen(1))
			gomega.Expect(stageList[0].Key()).To(gomega.Equal("gateway/stage1"))
		})

		It("instance event", func() {
			agent.handleEvent(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       v1beta1.BkGatewayInstanceTypeName,
			})

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			// Instance event should be ignored
			gomega.Expect(stageList).To(gomega.HaveLen(0))
		})

		It("Secret event", func() {
			patchGuard := sm.Patch((*EventAgent).handleSecret, func(_ *EventAgent, event *registry.ResourceMetadata) error {
				fmt.Println("monkey patch")
				return nil
			})
			defer patchGuard.Unpatch()

			agent.handleEvent(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       "Secret",
			})

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			// Secret event will handle by handleSecret
			// 这里只测试handleEvent, 不测试handleSecret, 所以monkey patch后不会执行handleSecret, 不会set timer
			gomega.Expect(stageList).To(gomega.HaveLen(0))
		})

		It("tls event", func() {
			patchGuard := sm.Patch((*EventAgent).handleSecret, func(_ *EventAgent, event *registry.ResourceMetadata) error {
				fmt.Println("monkey patch")
				return nil
			})
			defer patchGuard.Unpatch()

			agent.handleEvent(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       v1beta1.BkGatewayTLSTypeName,
			})

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			// tls event will handle by handleSecret
			gomega.Expect(stageList).To(gomega.HaveLen(0))
		})

		It("empty event", func() {
			patchGuard := sm.Patch((*EventAgent).handleSecret, func(_ *EventAgent, event *registry.ResourceMetadata) error {
				fmt.Println("monkey patch")
				return nil
			})
			defer patchGuard.Unpatch()

			agent.handleEvent(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "",
					StageName:   "",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       "",
			})

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			// empty event will handle by handleSecret
			gomega.Expect(stageList).To(gomega.HaveLen(0))
		})

		It("retry event", func() {
			patchGuard := sm.Patch((*EventAgent).handleSecret, func(_ *EventAgent, event *registry.ResourceMetadata) error {
				fmt.Println("monkey patch")
				return errors.New("retry")
			})
			defer patchGuard.Unpatch()

			agent.handleEvent(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "",
					StageName:   "",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       "",
			})

			event := <-agent.retryChan
			// empty event will handle by handleSecret
			// if handleSecret failed, retryChan will be pushed
			gomega.Expect(event.IsEmpty()).To(gomega.BeTrue())
		})
	})

	Describe("handleSecret", func() {
		It("retry over limit", func() {
			err := agent.handleSecret(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "",
					StageName:   "",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       "",
				RetryCount: 6,
			})
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			gomega.Expect(stageList).To(gomega.HaveLen(0))
		})

		It("empty event", func() {
			mockRegister.EXPECT().ListStages(gomock.Any()).Return([]registry.StageInfo{
				{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				{
					GatewayName: "gateway",
					StageName:   "stage2",
					Ctx:         nil,
				},
			}, nil)

			err := agent.handleSecret(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "",
					StageName:   "",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       "",
			})
			gomega.Expect(err).To(gomega.BeNil())

			stageList := <-commitCh
			gomega.Expect(stageList).To(gomega.HaveLen(2))
			gomega.Expect(stageList[0].Key()).To(gomega.Equal("gateway/stage1"))
		})

		It("secret event", func() {
			mockRadixTreeGetter.EXPECT().Get(gomock.Any()).Return(nil)
			patchGuard := sm.Patch((*EventAgent).secretEventCallback, func(
				_ *EventAgent,
				ctx context.Context,
				obj registry.ResourceKey,
				radixTree radixtree.RadixTree,
			) (bool, error) {
				fmt.Println("monkey patch")
				return true, nil
			})
			defer patchGuard.Unpatch()

			err := agent.handleSecret(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       "Secret",
			})
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			// handleSecret will set stage to timer
			gomega.Expect(stageList).To(gomega.HaveLen(1))
		})

		It("BkGatewayTLSTypeName event", func() {
			mockRadixTreeGetter.EXPECT().Get(gomock.Any()).Return(nil)
			mockRegister.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			err := agent.handleSecret(&registry.ResourceMetadata{
				StageInfo: registry.StageInfo{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				APIVersion: "v1beta1",
				Kind:       v1beta1.BkGatewayTLSTypeName,
			})
			gomega.Expect(err).To(gomega.BeNil())

			time.Sleep(200 * time.Millisecond)

			stageList := stageTimer.ListStagesForCommit()
			// tls sni is empty, so stage will be not pushed
			gomega.Expect(stageList).To(gomega.HaveLen(0))
		})
	})

	Describe("handleTicker", func() {
		It("empty stage", func() {
			mockRegister.EXPECT().ListStages(gomock.Any()).Return([]registry.StageInfo{
				{
					GatewayName: "gateway",
					StageName:   "stage1",
					Ctx:         nil,
				},
				{
					GatewayName: "gateway",
					StageName:   "stage2",
					Ctx:         nil,
				},
			}, nil)

			stageTimer.Update(registry.StageInfo{
				GatewayName: "",
				StageName:   "",
				Ctx:         nil,
			})

			time.Sleep(200 * time.Millisecond)

			agent.handleTicker(context.Background())

			stageList := <-commitCh
			gomega.Expect(stageList).To(gomega.HaveLen(2))
			gomega.Expect(stageList[0].Key()).To(gomega.Equal("gateway/stage1"))
		})

		It("normal stage", func() {
			stageTimer.Update(registry.StageInfo{
				GatewayName: "gateway",
				StageName:   "stage1",
				Ctx:         nil,
			})

			time.Sleep(200 * time.Millisecond)

			agent.handleTicker(context.Background())

			stageList := <-commitCh
			gomega.Expect(stageList).To(gomega.HaveLen(1))
			gomega.Expect(stageList[0].Key()).To(gomega.Equal("gateway/stage1"))
		})
	})
})
