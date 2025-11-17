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

// Package committer provides the functionality to commit changes
package committer

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/constant"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/agent/timer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/registry"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/core/synchronizer"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/eventreporter"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/trace"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

const maxStageRetryCount = 3

// Committer ...
type Committer struct {
	resourceRegistry   *registry.APIGWEtcdRegistry
	commitResourceChan chan []*entity.ReleaseInfo

	synchronizer *synchronizer.ApisixConfigSynchronizer

	resourceTimer *timer.ReleaseTimer

	logger *zap.SugaredLogger

	// Gateway stage dimension
	gatewayStageChanMap map[string]chan struct{}
	gatewayStageMapLock *sync.RWMutex
}

// NewCommitter 创建Committer
func NewCommitter(
	resourceRegistry *registry.APIGWEtcdRegistry,
	synchronizer *synchronizer.ApisixConfigSynchronizer,
	stageTimer *timer.ReleaseTimer,
) *Committer {
	return &Committer{
		resourceRegistry:    resourceRegistry,                      // Registry for resource management
		commitResourceChan:  make(chan []*entity.ReleaseInfo),      // Channel for committing resource information
		synchronizer:        synchronizer,                          // Configuration synchronizer
		resourceTimer:       stageTimer,                            // Timer for stage management
		logger:              logging.GetLogger().Named("commiter"), // Logger instance named "commiter"
		gatewayStageChanMap: make(map[string]chan struct{}),        // Map for storing gateway stage channels
		gatewayStageMapLock: &sync.RWMutex{},
	}
}

// Run ...
func (c *Committer) Run(ctx context.Context) {
	// 分批次处理需要同步的resource
	for {
		c.logger.Debugw("commiter waiting for commit command")
		select {
		case resourceList := <-c.commitResourceChan:
			c.logger.Infow("received commit command", "resourceList", resourceList)

			// 分批处理resource，避免一次性处理过多resource
			segmentLength := 10
			for offset := 0; offset < len(resourceList); offset += segmentLength {
				if offset+segmentLength > len(resourceList) {
					c.commitGroup(ctx, resourceList[offset:])
					break
				}
				c.commitGroup(ctx, resourceList[offset:(offset+segmentLength)])

				c.logger.Infow("Commit resource group done", "resourceList",
					resourceList[offset:(offset+segmentLength)])
			}

		case <-ctx.Done():
			c.logger.Info("gateway agent stopped, stop commit")
			return
		}
	}
}

// GetCommitChan 获取提交channel
func (c *Committer) GetCommitChan() chan []*entity.ReleaseInfo {
	return c.commitResourceChan
}

// ForceCommit ...
func (c *Committer) ForceCommit(ctx context.Context, stageList []*entity.ReleaseInfo) {
	c.logger.Infow("force commit stage changes", "stageList", stageList)
	c.commitResourceChan <- stageList
}

func (c *Committer) commitGroup(ctx context.Context, releaseInfoList []*entity.ReleaseInfo) {
	c.logger.Debugw("Commit resource group", "resourceList", releaseInfoList)
	// batch write apisix conf to buffer
	wg := &sync.WaitGroup{}
	for _, resourceInfo := range releaseInfoList {
		wg.Add(1)
		tempResourceInfo := resourceInfo
		// 判断是否是 global 资源：PluginMetadata 且 Stage 为空
		if tempResourceInfo.Kind == constant.PluginMetadata && tempResourceInfo.GetStageName() == "" {
			// Global 资源需要单独处理
			utils.GoroutineWithRecovery(ctx, func() {
				c.logger.Infof("begin commit global resource: %s", tempResourceInfo.GetID())
				c.commitGlobalResource(ctx, tempResourceInfo)
				c.logger.Infof("end commit global resource: %s", tempResourceInfo.GetID())
				wg.Done()
			})
		} else {
			// Stage 资源按 gateway 维度串行处理
			utils.GoroutineWithRecovery(ctx, func() {
				c.logger.Infof("begin commit gateway channel: %s", tempResourceInfo.GetID())
				c.commitGatewayStage(ctx, tempResourceInfo, wg)
				c.logger.Infof("end commit gateway channel: %s", tempResourceInfo.GetID())
			})
		}
	}
	wg.Wait()
}

// 按照gateway的维度串行更新etcd
func (c *Committer) commitGatewayStage(ctx context.Context, si *entity.ReleaseInfo, wg *sync.WaitGroup) {
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
		c.logger.Infof("begin commit stage channel: %s", si.GetReleaseID())
		c.commitStage(ctx, si, stageChan)
		c.logger.Infof("end commit stage channel: %s", si.GetReleaseID())
	})
}

func (c *Committer) commitStage(ctx context.Context, si *entity.ReleaseInfo, stageChan chan struct{}) {
	defer func() {
		<-stageChan
	}()
	// trace
	_, span := trace.StartTrace(si.Ctx, "commiter.commitStage")
	defer span.End()

	span.AddEvent("commiter.GetNativeApisixConfiguration")
	eventreporter.ReportParseConfigurationDoingEvent(ctx, si)
	// 直接从etcd获取原生apisix配置，无需转换
	apisixConf, err := c.GetStageReleaseNativeApisixConfiguration(ctx, si)
	if err != nil {
		c.logger.Error(err, "get native apisix configuration failed", "stageInfo", si)
		// retry
		c.retryStage(si)
		span.RecordError(err)
		eventreporter.ReportParseConfigurationFailureEvent(ctx, si, err)
		return
	} else {
		eventreporter.ReportParseConfigurationSuccessEvent(ctx, si)
	}
	eventreporter.ReportApplyConfigurationDoingEvent(ctx, si)

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
	// eventrepoter.ReportApplyConfigurationSuccessEvent(ctx, stage) // 可以由事件之前的关系推断出来
	eventreporter.ReportLoadConfigurationResultEvent(ctx, si)
	c.logger.Infow("commit stage success", "stageInfo", si)
}

func (c *Committer) retryStage(si *entity.ReleaseInfo) {
	if si.RetryCount >= maxStageRetryCount {
		c.logger.Error("too many retries", "stageInfo", si)
		return
	}
	si.RetryCount++
	c.resourceTimer.Update(si)
}

// GetStageReleaseNativeApisixConfiguration 直接从etcd获取原生apisix配置
func (c *Committer) GetStageReleaseNativeApisixConfiguration(
	ctx context.Context,
	si *entity.ReleaseInfo,
) (*entity.ApisixStageResource, error) {
	// 直接从etcd获取原生apisix配置
	resources, err := c.resourceRegistry.ListStageResources(si)
	if err != nil {
		c.logger.Error(err, "list resources failed", "stageInfo", si)
		return nil, err
	}
	metric.ReportResourceCountHelper(si.GetGatewayName(), si.GetStageName(), resources, ReportResourceConvertedMetric)
	return resources, nil
}

// GetGlobalApisixConfiguration 直接从etcd获取原生全局apisix配置
func (c *Committer) GetGlobalApisixConfiguration(
	ctx context.Context,
	si *entity.ReleaseInfo,
) (*entity.ApisixGlobalResource, error) {
	// 直接从etcd获取原生apisix配置
	resources, err := c.resourceRegistry.ListGlobalResources(si)
	if err != nil {
		c.logger.Error(err, "list resources failed", "stageInfo", si)
		return nil, err
	}
	return resources, nil
}

func (c *Committer) commitGlobalResource(ctx context.Context, si *entity.ReleaseInfo) {
	// trace
	_, span := trace.StartTrace(si.Ctx, "commiter.commitGlobalResource")
	defer span.End()

	span.AddEvent("commiter.GetGlobalApisixConfiguration")
	// 直接从etcd获取原生全局apisix配置，无需转换
	apisixGlobalConf, err := c.GetGlobalApisixConfiguration(ctx, si)
	if err != nil {
		c.logger.Error(err, "  get native global apisix configuration failed", "globalInfo", si)
		// retry
		c.retryStage(si)
		span.RecordError(err)
		return
	}
	span.AddEvent("commiter.SyncGlobal")
	err = c.synchronizer.SyncGlobal(
		ctx,
		apisixGlobalConf,
	)
	if err != nil {
		c.logger.Error(err, "sync global apisix configuration failed", "globalInfo", si)
		// retry
		c.retryStage(si)
		span.RecordError(err)
		return
	}
	c.logger.Infow("commit global resource success", "globalInfo", si)
}
