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

// ApisixConfigurationSynchronizer is implementation for Synchronizer
type ApisixConfigSynchronizer interface {
	Sync(
		ctx context.Context,
		gatewayName, stageName string,
		config *apisix.ApisixConfiguration,
	) error
	RemoveNotExistStage(ctx context.Context, existStageKeys []string) error
}

// apisixConfigurationSynchronizer is implementation for Synchronizer
type apisixConfigurationSynchronizer struct {
	store    ApisixConfigStore
	flushMux sync.Mutex

	apisixHealthzURI string

	logger *zap.SugaredLogger
}

// NewSynchronizer create new Synchronizer
func NewSynchronizer(store ApisixConfigStore, apisixHealthzURI string) ApisixConfigSynchronizer {
	syncer := &apisixConfigurationSynchronizer{
		store:            store,
		apisixHealthzURI: apisixHealthzURI,
		logger:           logging.GetLogger().Named("apisix-config-synchronizer"),
	}
	return syncer
}

// Sync will sync new staged apisix configuration
func (as *apisixConfigurationSynchronizer) Sync(
	ctx context.Context,
	gatewayName, stageName string,
	config *apisix.ApisixConfiguration,
) error {
	key := cfg.GenStagePrimaryKey(gatewayName, stageName)

	as.flushMux.Lock()
	defer as.flushMux.Unlock()

	as.logger.Debug("flush changes")
	err := as.store.Alter(ctx, key, config)
	if err != nil {
		as.logger.Errorw("Failed to sync stage", "err", err, "key", key, "content", config)
		return err
	}

	as.logger.Debug("flush virtual stage")
	virtualStage := NewVirtualStage(as.apisixHealthzURI)
	err = as.store.Alter(ctx, cfg.VirtualStageKey, virtualStage.MakeConfiguration())
	if err != nil {
		as.logger.Errorw(
			"Failed to sync virtual stage",
			"err", err, "key", cfg.VirtualStageKey,
			"content", virtualStage.MakeConfiguration(),
		)
		return err
	}

	ReportStageConfigSyncMetric(gatewayName, stageName)

	return nil
}

// RemoveNotExistStage remove stages that does not exist
func (as *apisixConfigurationSynchronizer) RemoveNotExistStage(ctx context.Context, existStageKeys []string) error {
	as.flushMux.Lock()
	defer as.flushMux.Unlock()

	existStageConfig := as.store.GetAll()

	keySet := make(map[string]struct{})
	for _, key := range existStageKeys {
		keySet[key] = struct{}{}
	}
	for key := range existStageConfig {
		if _, ok := keySet[key]; !ok {
			err := as.store.Alter(ctx, key, apisix.NewEmptyApisixConfiguration())
			if err != nil {
				as.logger.Errorw("Remove not exist stage failed", "key", key, "err", err)
				return err
			}
		}
	}

	return nil
}
