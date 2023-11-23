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
	"os"
	"time"

	"github.com/rotisserie/eris"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/TencentBlueKing/blueking-apigateway-operator/api/v1beta1"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/radixtree"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
)

const (
	retryLimit = 5
)

var retryDelaySeconds = time.Second * 5

// EventAgent ...
type EventAgent struct {
	resourceRegistry registry.Registry
	commitChan       chan []registry.StageInfo
	synchronizer     synchronizer.ApisixConfigSynchronizer

	// for upstream tls cert
	radixTreeGetter radixtree.RadixTreeGetter
	stageTimer      *timer.StageTimer

	retryChan chan *registry.ResourceMetadata

	keepAliveChan <-chan struct{} // for leader election

	logger *zap.SugaredLogger
}

// NewEventAgent ...
func NewEventAgent(
	resourceRegistry registry.Registry,
	commitCh chan []registry.StageInfo,
	synchronizer synchronizer.ApisixConfigSynchronizer,
	radixTreeGetter radixtree.RadixTreeGetter,
	stageTimer *timer.StageTimer,
) *EventAgent {
	return &EventAgent{
		resourceRegistry: resourceRegistry,
		commitChan:       commitCh,
		synchronizer:     synchronizer,
		radixTreeGetter:  radixTreeGetter,
		stageTimer:       stageTimer,
		retryChan:        make(chan *registry.ResourceMetadata, 100),
		logger:           logging.GetLogger().Named("event-agent"),
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

func (w *EventAgent) createWatchChannel(ctx context.Context) (<-chan *registry.ResourceMetadata, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	retryCount := 0
	var watchCh <-chan *registry.ResourceMetadata
	for retryCount < retryLimit {
		w.logger.Debugw("Boostrap watch channel")
		watchCh = w.resourceRegistry.Watch(ctx)

		err := w.bootstrapSync(ctx)

		ReportBootstrapSyncingMetric(err)

		if err == nil {
			break
		}

		w.logger.Error(err, "Boostrap watch channel failed")
		retryCount++
		time.Sleep(retryDelaySeconds)
	}

	if retryCount >= retryLimit {
		w.logger.Error(nil, "Boostrap watch channel failed", "retryCount", retryCount)
		os.Exit(1)
	}

	return watchCh, cancel
}

func (w *EventAgent) bootstrapSync(ctx context.Context) error {
	// TODO 这里没有做radixtree的数据同步, 确认是否需要

	stageList, err := w.resourceRegistry.ListStages(ctx) // get all stages
	if err != nil {
		return err
	}

	// 避免启动全量同步,导致大量publish event上报
	for i := range stageList {
		stageList[i].PublishID = constant.NoNeedReportPublishID
	}

	keys := make([]string, 0, len(stageList))
	for _, stage := range stageList {
		keys = append(keys, config.GenStagePrimaryKey(stage.GatewayName, stage.StageName))
	}

	w.logger.Debugw("RenewStages gateway exists keys", "keys", keys)

	err = w.synchronizer.RemoveNotExistStage(ctx, keys)
	if err != nil {
		return err
	}
	w.radixTreeGetter.RemoveNotExistStage(stageList)

	w.commitChan <- stageList // 全量同步

	return nil
}

func (w *EventAgent) handleEvent(event *registry.ResourceMetadata) {
	// trace
	ctx, span := trace.StartTrace(event.Ctx, "agent.handleEvent")
	event.Ctx = ctx
	defer span.End()

	if event.Kind == v1beta1.BkGatewayInstanceTypeName {
		w.logger.Debugw("skip BkInstance event")

		span.AddEvent("skip BkInstance event")
		return
	}

	if event.IsEmpty() || event.Kind == "Secret" ||
		event.Kind == v1beta1.BkGatewayTLSTypeName {
		err := w.handleSecret(event)
		if err != nil {
			event.RetryCount += 1

			// retry max limit 5 times
			time.AfterFunc(retryDelaySeconds, func() {
				w.retryChan <- event
			})
		}

		return
	}

	w.logger.Debugw("Receive event", "gatewayName", event.StageInfo.GatewayName, "stageName", event.StageInfo.StageName)
	// 更新时间窗口
	w.stageTimer.Update(event.StageInfo)
}

func (w *EventAgent) handleSecret(event *registry.ResourceMetadata) error {
	if event == nil {
		w.logger.Error(nil, "Receive nil event, ignore it")
		return nil
	}

	// trace
	ctx, span := trace.StartTrace(event.Ctx, "agent.handleSecret")
	event.Ctx = ctx
	defer span.End()

	if event.RetryCount > retryLimit {
		w.logger.Error(nil, "Receive retry event, retry count exceeded, ignore it", "event", event)
		return nil
	}

	switch event.Kind {
	case "Secret":
		var radixTree radixtree.RadixTree = w.radixTreeGetter.Get(event.StageInfo)
		shouldProcess, err := w.secretEventCallback(
			ctx,
			registry.ResourceKey{StageInfo: event.StageInfo, ResourceName: event.Name},
			radixTree,
		)
		if err != nil {
			w.logger.Error(err, "Reconcile secret failed", "event", event)
			return err
		}
		if shouldProcess {
			w.logger.Debugw(
				"Receive secret event",
				"gatewayName",
				event.StageInfo.GatewayName,
				"stageName",
				event.StageInfo.StageName,
			)

			w.stageTimer.Update(event.StageInfo)
		}
	case v1beta1.BkGatewayTLSTypeName:
		var radixTree radixtree.RadixTree = w.radixTreeGetter.Get(event.StageInfo)
		tlsObj := &v1beta1.BkGatewayTLS{}
		err := w.resourceRegistry.Get(
			ctx,
			registry.ResourceKey{StageInfo: event.StageInfo, ResourceName: event.Name},
			tlsObj,
		)
		if err != nil {
			w.logger.Error(err, "Get BkGaetwayTLS failed", "event", event)
			return err
		}
		// ignore
		var shouldProcess bool
		if len(tlsObj.Spec.SNIs) != 0 {
			shouldProcess, err = w.gatewayTLSCallback(
				ctx,
				registry.ResourceKey{ResourceName: tlsObj.Spec.GatewayTLSSecretRef, StageInfo: event.StageInfo},
				radixTree,
				tlsObj,
			)
		}
		if err != nil {
			w.logger.Error(err, "Reconcile secret failed", "event", event)
			return err
		}
		if shouldProcess {
			w.logger.Debugw(
				"Receive tls event",
				"gatewayName",
				event.StageInfo.GatewayName,
				"stageName",
				event.StageInfo.StageName,
			)
			w.stageTimer.Update(event.StageInfo)
		}
	default:
		if event.IsEmpty() {
			siList, err := w.resourceRegistry.ListStages(ctx)
			if err != nil {
				w.logger.Error(err, "ListStage failed")
				return err
			}
			w.commitChan <- siList
		} else {
			w.stageTimer.Update(event.StageInfo)
		}
	}

	return nil
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
	if includeAllStage {
		allStages, err := w.resourceRegistry.ListStages(ctx)
		if err != nil {
			w.logger.Error(err, "List stage failed when all stage event triggered")
			w.stageTimer.Update(registry.StageInfo{})

			// 避免全量同步,导致大量publish event上报
			for i := range stageList {
				stageList[i].PublishID = constant.NoNeedReportPublishID
			}

			w.commitChan <- stageList
			return
		}

		w.commitChan <- allStages
		return
	}

	if len(stageList) != 0 {
		w.commitChan <- stageList
	}
}

func (w *EventAgent) secretEventCallback(
	ctx context.Context,
	obj registry.ResourceKey,
	radixTree radixtree.RadixTree,
) (bool, error) {
	secret := &v1.Secret{}
	err := w.resourceRegistry.Get(ctx, obj, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			radixTree.Delete(obj)
			return true, nil
		}
		return false, eris.Wrapf(err, "Get secret (%s) failed", obj)
	}
	tlsCert, err := v1beta1.GetTLSCertFromSecret(secret)
	if err != nil {
		w.logger.Debugw("extract cert from secret failed, skip", "error", err, "secret", obj)
		return false, nil
	}

	shouldProcess, err := radixTree.Insert(obj, tlsCert)
	if err != nil {
		return false, err
	}
	return shouldProcess, nil
}

func (w *EventAgent) gatewayTLSCallback(
	ctx context.Context,
	secretObj registry.ResourceKey,
	radixTree radixtree.RadixTree,
	tlsObj *v1beta1.BkGatewayTLS,
) (bool, error) {
	if len(tlsObj.Spec.SNIs) == 0 {
		return false, nil
	}
	tlsObjKey := registry.ResourceKey{
		StageInfo:    secretObj.StageInfo,
		ResourceName: "virtual-key-" + tlsObj.GetName(),
	}
	secret := &v1.Secret{}
	err := w.resourceRegistry.Get(ctx, secretObj, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			radixTree.Delete(tlsObjKey)
			return true, nil
		}
		return false, eris.Wrapf(err, "Get secret (%s) failed", secretObj)
	}

	tlsCert, err := v1beta1.GetTLSCertFromSecret(secret)
	if err != nil {
		w.logger.Infow(
			"extract cert from secret failed, skip",
			"error",
			err,
			"bkgatewaytls",
			tlsObj.Spec,
			"stageinfo",
			secretObj.StageInfo,
		)
		return false, nil
	}
	tlsCert.SNIs = tlsObj.Spec.SNIs

	shouldProcess, err := radixTree.Insert(tlsObjKey, tlsCert)
	if err != nil {
		return false, err
	}
	return shouldProcess, err
}
