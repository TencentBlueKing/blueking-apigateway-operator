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

package integration

import (
	"net/http"
	"os"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v2"
)

const (
	BootstrapSyncingCountMetric       = "bootstrap_syncing_count"
	ResourceEventTriggeredCountMetric = "resource_event_triggered_count"
	ResourceConvertedCountMetric      = "resource_converted_count"
	ResourceSyncCmpCount              = "sync_cmp_count"
	ResourceSyncCmpDiffCount          = "sync_cmp_diff_count"
	ApisixOperationCountMetric        = "apisix_operation_count"
)

// EtcdConfig is the config for the etcd
type EtcdConfig struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// MetricsAdapter is the adapter for the metrics
type MetricsAdapter struct {
	Metrics map[string]*dto.MetricFamily
}

// GetHttpBinGatewayResource returns the httpbin gateway resource
func GetHttpBinGatewayResource() []EtcdConfig {
	var resources []EtcdConfig
	data, err := os.ReadFile("bk_apigw_httpbin_resources.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(data, &resources)
	if err != nil {
		panic(err)
	}
	return resources
}

// NewMetricsAdapter creates a new MetricsAdapter
func NewMetricsAdapter(host string) (*MetricsAdapter, error) {
	resp, err := http.Get(host + "/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 创建一个解析器
	parser := expfmt.TextParser{}
	// 使用解析器解析metrics 数据
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}
	return &MetricsAdapter{
		Metrics: metrics,
	}, nil
}

// GetResourceMetrics returns the resource metrics by metricsType and labels
func (m *MetricsAdapter) GetResourceMetrics(metricsType string, labels []string) int {
	resourceEventTriggeredCountMetric := m.Metrics[metricsType]
	if resourceEventTriggeredCountMetric == nil {
		return 0
	}
	for _, metric := range resourceEventTriggeredCountMetric.Metric {
		if len(labels) != len(metric.Label) {
			continue
		}
		marched := true
		for i, lab := range metric.Label {
			if labels[i] != lab.GetValue() {
				marched = false
				break
			}
		}
		if marched {
			return int(metric.Counter.GetValue())
		}
	}
	return 0
}

// GetResourceEventTriggeredCountMetric ResourceEventTriggeredCountMetric returns the resource event triggered
// count metric
func (m *MetricsAdapter) GetResourceEventTriggeredCountMetric(gateway string, stage string, resourceType string) int {
	return m.GetResourceMetrics(ResourceEventTriggeredCountMetric, []string{gateway, stage, resourceType})
}

// GetResourceConvertedCountMetric ResourceConvertedCountMetric returns the resource event triggered count metric
func (m *MetricsAdapter) GetResourceConvertedCountMetric(gateway string, stage string, resourceType string) int {
	return m.GetResourceMetrics(ResourceConvertedCountMetric, []string{gateway, stage, resourceType})
}

// GetResourceSyncCmpCountMetric ResourceSyncCmpCount returns the resource event triggered count metric
func (m *MetricsAdapter) GetResourceSyncCmpCountMetric(gateway string, stage string, resourceType string) int {
	return m.GetResourceMetrics(ResourceSyncCmpCount, []string{gateway, stage, resourceType})
}

// GetResourceSyncCmpDiffCountMetric ResourceSyncCmpDiffCount returns the resource event triggered count metric
func (m *MetricsAdapter) GetResourceSyncCmpDiffCountMetric(gateway string, stage string, resourceType string) int {
	return m.GetResourceMetrics(ResourceSyncCmpDiffCount, []string{gateway, stage, resourceType})
}

// GetApisixOperationCountMetric ApisixOperationCountMetric returns the resource event triggered count metric
func (m *MetricsAdapter) GetApisixOperationCountMetric(action string, result string, resourceType string) int {
	return m.GetResourceMetrics(ApisixOperationCountMetric, []string{action, result, resourceType})
}

// GetBootstrapSyncingSuccessCountMetric BootstrapSyncingCountMetric returns the resource event triggered count metric
func (m *MetricsAdapter) GetBootstrapSyncingSuccessCountMetric(result string) int {
	return m.GetResourceMetrics(BootstrapSyncingCountMetric, []string{result})
}
