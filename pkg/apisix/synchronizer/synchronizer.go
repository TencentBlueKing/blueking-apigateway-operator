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

package synchronizer

//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE -package=mock

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	cfg "github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

const (
	bufferCnt = 2
)

// ApisixConfigurationSynchronizer is implementation for Synchronizer
type ApisixConfigSynchronizer interface {
	Sync(
		ctx context.Context,
		gatewayName, stageName string,
		config *apisix.ApisixConfiguration,
	)
	Flush(ctx context.Context)
	RemoveNotExistStage(ctx context.Context, existStageKeys []string)
}

// apisixConfigurationSynchronizer is implementation for Synchronizer
type apisixConfigurationSynchronizer struct {
	sync.Mutex

	buffer          *apisix.SynchronizerBuffer
	bufferInd       int
	bufferSelection []*apisix.SynchronizerBuffer

	store    ApisixConfigStore
	flushMux sync.Mutex

	apisixHealthzURI string

	logger *zap.SugaredLogger
}

// NewSynchronizer create new Synchronizer
func NewSynchronizer(store ApisixConfigStore, apisixHealthzURI string) ApisixConfigSynchronizer {
	bufferSelection := make([]*apisix.SynchronizerBuffer, bufferCnt)
	for i := range bufferSelection {
		bufferSelection[i] = apisix.NewSynchronizerBuffer()
	}
	syncer := &apisixConfigurationSynchronizer{
		buffer:           bufferSelection[0],
		bufferSelection:  bufferSelection,
		store:            store,
		apisixHealthzURI: apisixHealthzURI,
		logger:           logging.GetLogger().Named("apisix-config-synchronizer"),
	}
	return syncer
}

func (as *apisixConfigurationSynchronizer) put(
	ctx context.Context,
	key string,
	config *apisix.ApisixConfiguration,
	retry bool,
) {
	as.Lock()
	defer as.Unlock()

	// 一般同步事件
	if !retry {
		// 处理非重试的同步事件
		as.buffer.Put(key, config)
		return
	}

	// 处理重试的同步事件
	_, ok := as.buffer.Get(key) // 如果已存在, 不更新
	if ok {
		return
	}
	as.logger.Debugw("Resync Message", "key", key, "content", config)
	as.buffer.Put(key, config)
}

// Sync will sync new staged apisix configuration
func (as *apisixConfigurationSynchronizer) Sync(
	ctx context.Context,
	gatewayName, stageName string,
	config *apisix.ApisixConfiguration,
) {
	key := cfg.GenStagePrimaryKey(gatewayName, stageName)
	as.put(ctx, key, config, false)

	ReportStageConfigSyncMetric(gatewayName, stageName)
}

func (as *apisixConfigurationSynchronizer) resync(ctx context.Context, key string, conf *apisix.ApisixConfiguration) {
	as.put(ctx, key, conf, true)
}

// Flush will flush the cached apisix configuration changes
func (as *apisixConfigurationSynchronizer) Flush(ctx context.Context) {
	go as.flush(ctx)
}

// RemoveNotExistStage remove stages that does not exist
func (as *apisixConfigurationSynchronizer) RemoveNotExistStage(ctx context.Context, existStageKeys []string) {
	as.flushMux.Lock()
	defer as.flushMux.Unlock()

	changedConfig := make(map[string]*apisix.ApisixConfiguration)
	existStageConfig := as.store.GetAll()

	keySet := make(map[string]struct{})
	for _, key := range existStageKeys {
		keySet[key] = struct{}{}
	}
	for key := range existStageConfig {
		if _, ok := keySet[key]; !ok {
			changedConfig[key] = apisix.NewEmptyApisixConfiguration()
		}
	}
	as.store.Alter(ctx, changedConfig, as.resync)
}

func (as *apisixConfigurationSynchronizer) flush(ctx context.Context) {
	as.Lock()

	// 取出buffer中的数据, 并重置buffer
	buffer := as.buffer
	changedConfig := buffer.LockAll()
	defer buffer.Done()

	as.bufferInd = (as.bufferInd + 1) % bufferCnt
	as.buffer = as.bufferSelection[as.bufferInd]
	as.Unlock()

	if len(changedConfig) == 0 {
		// 无变更，无需flush
		as.logger.Debug("No changes to resources has been made, do not need flush")
		return
	}

	as.flushMux.Lock()
	defer as.flushMux.Unlock()

	as.logger.Debug("flush changes")
	as.store.Alter(ctx, changedConfig, as.resync)

	as.logger.Debug("flush virtual stage")
	controlPlaneConfiguration := make(map[string]*apisix.ApisixConfiguration)
	virtualStage := NewVirtualStage(as.apisixHealthzURI)
	controlPlaneConfiguration[cfg.VirtualStageKey] = virtualStage.MakeConfiguration()
	as.store.Alter(ctx, controlPlaneConfiguration, as.resync)
}
