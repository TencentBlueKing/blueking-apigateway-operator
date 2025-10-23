/*
 *  TencentBlueKing is pleased to support the open source community by making
 *  蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 *  Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 *  Licensed under the MIT License (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at
 *
 *      http://opensource.org/licenses/MIT
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under
 *  the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 *  either express or implied. See the License for the specific language governing permissions and
 *   limitations under the License.
 *
 *   We undertake not to change the open source license (MIT license) applicable
 *   to the current version of the project delivered to anyone in the future.
 */

// Package metric ...
package metric

import (
	"strings"
	"time"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/entity"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
)

// ReportApisixEtcdMetric ...
func ReportApisixEtcdMetric(resType, action string, started time.Time, err error) {
	result := ResultSuccess
	if err != nil {
		result = ResultFail
	}

	ApisixOperationCounter.WithLabelValues(resType, action, result).Inc()
	ApisixOperationHistogram.WithLabelValues(resType, action, result).
		Observe(float64(time.Since(started).Milliseconds()))
}

// ReportStageConfigSyncMetric ...
func ReportStageConfigSyncMetric(gateway, stage string) {
	SynchronizerEventCounter.WithLabelValues(gateway, stage).Inc()
}

// ReportStageConfigAlterMetric ...
func ReportStageConfigAlterMetric(
	stageKey string,
	config *entity.ApisixConfiguration,
	started time.Time,
	err error,
) {
	parts := strings.Split(strings.Trim(stageKey, "/"), "/")
	var (
		gateway string
		stage   string
	)
	if len(parts) == 2 {
		gateway = parts[0]
		stage = parts[1]
	} else {
		logging.GetLogger().Infow("Invalid stage key", "stageKey", stageKey)
		return
	}

	result := ResultSuccess
	if err != nil {
		result = ResultFail
	} else {
		ReportResourceCountHelper(gateway, stage, config, func(gateway, stage, resType string, count int) {
			ApisixResourceWrittenCounter.WithLabelValues(gateway, stage, resType).Add(float64(count))
		})
	}

	SynchronizerFlushingCounter.WithLabelValues(gateway, stage, result).Inc()
	SynchronizerFlushingHistogram.WithLabelValues(gateway, stage, result).
		Observe(float64(time.Since(started).Milliseconds()))
}

// ReportSyncCmpMetric ...
func ReportSyncCmpMetric(gateway, stage, resourceType string) {
	SyncCmpCounter.WithLabelValues(gateway, stage, resourceType).Inc()
}

// ReportSyncCmpDiffMetric ...
func ReportSyncCmpDiffMetric(gateway, stage, resourceType string) {
	SyncCmpDiffCounter.WithLabelValues(gateway, stage, resourceType).Inc()
}
