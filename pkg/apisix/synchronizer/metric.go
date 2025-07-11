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

// Package synchronizer ...
package synchronizer

import (
	"strings"
	"time"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/apisix"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/logging"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/metric"
)

// ReportApisixEtcdMetric ...
func ReportApisixEtcdMetric(resType, action string, started time.Time, err error) {
	result := metric.ResultSuccess
	if err != nil {
		result = metric.ResultFail
	}

	metric.ApisixOperationCounter.WithLabelValues(resType, action, result).Inc()
	metric.ApisixOperationHistogram.WithLabelValues(resType, action, result).
		Observe(float64(time.Since(started).Milliseconds()))
}

// ReportStageConfigSyncMetric ...
func ReportStageConfigSyncMetric(gateway, stage string) {
	metric.SynchronizerEventCounter.WithLabelValues(gateway, stage).Inc()
}

// ReportStageConfigAlterMetric ...
func ReportStageConfigAlterMetric(
	stageKey string,
	config *apisix.ApisixConfiguration,
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

	result := metric.ResultSuccess
	if err != nil {
		result = metric.ResultFail
	} else {
		metric.ReportResourceCountHelper(gateway, stage, config, func(gateway, stage, resType string, count int) {
			metric.ApisixResourceWrittenCounter.WithLabelValues(gateway, stage, resType).Add(float64(count))
		})
	}

	metric.SynchronizerFlushingCounter.WithLabelValues(gateway, stage, result).Inc()
	metric.SynchronizerFlushingHistogram.WithLabelValues(gateway, stage, result).
		Observe(float64(time.Since(started).Milliseconds()))
}

// ReportSyncCmpMetric ...
func ReportSyncCmpMetric(gateway, stage, resourceType string) {
	metric.SyncCmpCounter.WithLabelValues(gateway, stage, resourceType).Inc()
}

// ReportSyncCmpDiffMetric ...
func ReportSyncCmpDiffMetric(gateway, stage, resourceType string) {
	metric.SyncCmpDiffCounter.WithLabelValues(gateway, stage, resourceType).Inc()
}
