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

// Package commiter provides the functionality to commit changes
package commiter

import (
	"context"
	"sync"

	"github.com/rotisserie/eris"
	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/watcher"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

const maxStageRetryCount = 3

var errStageNotFound = eris.Errorf("no bk gateway stage found")

// Commiter ...
type Commiter struct {
	resourceRegistry *watcher.APIGEtcdWWatcher

	commitChan chan []*entity.ReleaseStageInfo

	synchronizer *synchronizer.ApisixConfigSynchronizer

	stageTimer *timer.StageTimer

	logger *zap.SugaredLogger

	// Gateway stage dimension
	gatewayStageChanMap map[string]chan struct{}
	gatewayStageMapLock *sync.RWMutex
}

// NewCommiter 创建Commiter
func NewCommiter(
resourceRegistry *watcher.APIGEtcdWWatcher,
synchronizer *synchronizer.ApisixConfigSynchronizer,
stageTimer *timer.StageTimer,
) *Commiter {
	return &Commiter{
		resourceRegistry:    resourceRegistry,                      // Registry for resource management
		commitChan:          make(chan []*entity.ReleaseStageInfo), // Channel for committing stage information
		synchronizer:        synchronizer,                          // Configuration synchronizer
		stageTimer:          stageTimer,                            // Timer for stage management
		logger:              logging.GetLogger().Named("commiter"), // Logger instance named "commiter"
		gatewayStageChanMap: make(map[string]chan struct{}),        // Map for storing gateway stage channels
		gatewayStageMapLock: &sync.RWMutex{},
	}
}

// Run ...
func (c *Commiter) Run(ctx context.Context) {
	// 分批次处理需要同步的stage
	for {
		c.logger.Debugw("commiter waiting for commit command")
		select {
		case stageList := <-c.commitChan:
			c.logger.Infow("received commit command", "stageList", stageList)

			// 分批处理stage，避免一次性处理过多stage
			segmentLength := 10
			for offset := 0; offset < len(stageList); offset += segmentLength {
				if offset+segmentLength > len(stageList) {
					c.commitGroup(ctx, stageList[offset:])
					break
				}

				c.commitGroup(ctx, stageList[offset:(offset+segmentLength)])

				c.logger.Infow("Commit stage keys done", "stageList", stageList)
			}

		case <-ctx.Done():
			c.logger.Info("gateway agent stopped, stop commit")
			return
		}
	}
}

// GetCommitChan 获取提交channel
func (c *Commiter) GetCommitChan() chan []*entity.ReleaseStageInfo {
	return c.commitChan
}

// ForceCommit ...
func (c *Commiter) ForceCommit(ctx context.Context, stageList []*entity.ReleaseStageInfo) {
	c.logger.Infow("force commit stage changes", "stageList", stageList)
	c.commitChan <- stageList
}

func (c *Commiter) commitGroup(ctx context.Context, stageInfoList []*entity.ReleaseStageInfo) {
	c.logger.Debugw("Commit stage group", "stageList", stageInfoList)

	// batch write apisix conf to buffer
	wg := &sync.WaitGroup{}
	for _, stageInfo := range stageInfoList {
		wg.Add(1)
		tempStageInfo := stageInfo
		utils.GoroutineWithRecovery(ctx, func() {
			c.logger.Infof("begin commit gateway channel: %s", tempStageInfo.Id)
			c.commitGatewayStage(ctx, tempStageInfo, wg)
			c.logger.Infof("end commit gateway channel: %s", tempStageInfo.Id)
		})
	}
	wg.Wait()
}

// 按照gateway的维度串行更新etcd
func (c *Commiter) commitGatewayStage(ctx context.Context, si *entity.ReleaseStageInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	c.gatewayStageMapLock.Lock()
	stageChan, ok := c.gatewayStageChanMap[si.GetGatewayName()]
	if !ok {
		stageChan = make(chan struct{}, 1)
		c.gatewayStageChanMap[si.GetGatewayName()] = stageChan
	}
	c.gatewayStageMapLock.Unlock()
	utils.GoroutineWithRecovery(ctx, func() {
		// Control stage writes for each gateway to be serial
		stageChan <- struct{}{}
		c.logger.Infof("begin commit stage channel: %s", si.GetStageID())
		c.commitStage(ctx, si, stageChan)
		c.logger.Infof("end commit stage channel: %s", si.GetStageID())
	})

}

func (c *Commiter) commitStage(ctx context.Context, si *entity.ReleaseStageInfo, stageChan chan struct{}) {
	defer func() {
		<-stageChan
	}()
	// trace
	_, span := trace.StartTrace(si.Ctx, "commiter.commitStage")
	defer span.End()

	span.AddEvent("commiter.GetNativeApisixConfiguration")
	// 直接从etcd获取原生apisix配置，无需转换
	apisixConf, err := c.GetNativeApisixConfiguration(ctx, si)
	if err != nil {
		c.logger.Error(err, "get native apisix configuration failed", "stageInfo", si)
		// retry
		c.retryStage(si)
		span.RecordError(err)
		return
	}

	span.AddEvent("commiter.Sync")
	err = c.synchronizer.Sync(
		ctx,
		si.GetGatewayName(),
		si.GetStageName(),
		apisixConf,
	)
	if err != nil {
		c.logger.Error(err, "sync apisix configuration failed", "stageInfo", si)
		// retry
		c.retryStage(si)
		span.RecordError(err)
		return
	}

	c.logger.Infow("commit stage success", "stageInfo", si)
}

func (c *Commiter) retryStage(si *entity.ReleaseStageInfo) {
	if si.RetryCount >= maxStageRetryCount {
		c.logger.Errorf("too many retries", "stageInfo", si)
		return
	}

	si.RetryCount++
	c.stageTimer.Update(si)
}

// GetNativeApisixConfiguration 直接从etcd获取原生apisix配置
func (c *Commiter) GetNativeApisixConfiguration(
ctx context.Context,
si *entity.ReleaseStageInfo,
) (*entity.ApisixConfiguration, error) {
	// 直接从etcd获取原生apisix配置，无需转换
	// 这里应该直接从etcd读取apisix配置数据
	// 暂时返回空配置，后续需要根据实际的etcd数据结构来实现
	return entity.NewEmptyApisixConfiguration(), nil
}
