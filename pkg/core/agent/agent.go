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

// Package agent handle the event from resource registry
package agent

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/watcher"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
)

const (
	retryLimit = 5
)

var retryDelaySeconds = time.Second * 5

// EventAgent ...
type EventAgent struct {
	apigwWatcher *watcher.APIGEtcdWatcher
	commitChan   chan []*entity.ReleaseStageInfo
	synchronizer *synchronizer.ApisixConfigSynchronizer

	stageTimer *timer.StageTimer

	retryChan chan *entity.ResourceMetadata

	keepAliveChan <-chan struct{} // for leader election

	logger *zap.SugaredLogger
}

// NewEventAgent ...
func NewEventAgent(
resourceRegistry *watcher.APIGEtcdWatcher,
commitCh chan []*entity.ReleaseStageInfo,
synchronizer *synchronizer.ApisixConfigSynchronizer,
stageTimer *timer.StageTimer,
) *EventAgent {
	return &EventAgent{
		apigwWatcher: resourceRegistry,
		commitChan:   commitCh,
		synchronizer: synchronizer,
		stageTimer:   stageTimer,
		retryChan:    make(chan *entity.ResourceMetadata, 100),
		logger:       logging.GetLogger().Named("event-agent"),
	}
}

// SetKeepAliveChan ...
func (w *EventAgent) SetKeepAliveChan(keepAliveChan <-chan struct{}) {
	w.keepAliveChan = keepAliveChan
}

// Run ...
func (w *EventAgent) Run(ctx context.Context) {
	watchCh, watchCancel := w.createWatchChannel(ctx)

	ticker := time.NewTicker(commitTimeWindow) // 窗口定时器
	for {
		select {
		// event receive
		case event, ok := <-watchCh:
			w.logger.Debugw("resource registry event trigger", "event", event)

			if !ok {
				w.logger.Error("Watch resources failed: channel break")

				// stop last watch loop
				watchCancel()

				// reset watch channel
				watchCh, watchCancel = w.createWatchChannel(ctx)

				break
			}

			ReportEventTriggeredMetric(event)

			// 更新stage的事件窗口, 发送特殊事件到innerLoopChan
			// NOTE: 事件实际只是记录有哪个stage需要更新, 更新的单位为stage, 而不是细粒度的资源本身
			w.handleEvent(event)

		case event := <-w.retryChan:
			w.logger.Debugw("retry channel event trigger", "event", event)

			w.handleEvent(event)
		// events commit
		case <-ticker.C:
			w.logger.Debugw("commit ticker trigger")

			// 定时处理时间窗口已经超时的stage
			w.handleTicker(ctx)

		case <-w.keepAliveChan:
			w.logger.Debugw("keep alive trigger")
			return

		case <-ctx.Done():
			w.logger.Infow("gateway agent stopped, stop watching etcd")
			return
		}
	}
}

func (w *EventAgent) createWatchChannel(ctx context.Context) (<-chan *entity.ResourceMetadata, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	watchCh := w.apigwWatcher.Watch(ctx)

	return watchCh, cancel
}

func (w *EventAgent) handleEvent(event *entity.ResourceMetadata) {
	// trace
	ctx, span := trace.StartTrace(event.Ctx, "agent.handleEvent")
	event.Ctx = ctx
	defer span.End()

	if event.IsEmpty() {
		w.logger.Debugw("skip empty event")

		span.AddEvent("skip empty event")
		return
	}

	switch event.Kind {
	// 全局配置单独处理
	case constant.GlobalRule, constant.PluginMetadata:
		w.logger.Debugw("skip BkInstance event")

		span.AddEvent("skip BkInstance event")
		return
	default:
		w.logger.Debugw("Receive event", "gatewayName",
			event.Labels.Gateway, "stageName", event.Labels.Stage)
		// 更新时间窗口
		w.stageTimer.Update(event.GetReleaseStageInfo())

	}

}

func (w *EventAgent) handleTicker(ctx context.Context) {
	stageList := w.stageTimer.ListStagesForCommit()
	var includeAllStage bool
	for _, stage := range stageList {
		if stage.IsEmpty() {
			includeAllStage = true
		}
	}
	w.logger.Debugw("stages to be committed", "stageList", stageList, "includeAllStage", includeAllStage)
	if len(stageList) != 0 {
		w.commitChan <- stageList
	}
}
