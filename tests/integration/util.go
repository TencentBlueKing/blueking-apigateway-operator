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

	json "github.com/json-iterator/go"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const (
	BootstrapSyncingCountMetric       = "bootstrap_syncing_count"
	ResourceEventTriggeredCountMetric = "resource_event_triggered_count"
	ResourceConvertedCountMetric      = "resource_converted_count"
	ResourceSyncCmpCount              = "sync_cmp_count"
	ResourceSyncCmpDiffCount          = "sync_cmp_diff_count"
	ApisixOperationCountMetric        = "apisix_operation_count"
)

type EtcdConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func GetHttpBinGatewayResource() []EtcdConfig {
	// load json
	var resources []EtcdConfig
	data, err := os.ReadFile("bk_apigw_httpbin_resources.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &resources)
	if err != nil {
		panic(err)
	}
	return resources
}

func GetAllMetrics() (map[string]*dto.MetricFamily, error) {
	resp, err := http.Get("http://127.0.0.1:6004/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 创建一个解析器
	parser := expfmt.TextParser{}
	// 使用解析器解析metrics 数据
	return parser.TextToMetricFamilies(resp.Body)
}

func GetResourceEventTriggeredCountMetric(
	metrics map[string]*dto.MetricFamily, gateway string, stage string, resourceType string) float64 {
	resourceEventTriggeredCountMetric := metrics[ResourceEventTriggeredCountMetric]
	if resourceEventTriggeredCountMetric == nil {
		return 0
	}
	for _, metric := range resourceEventTriggeredCountMetric.Metric {
		if len(metric.Label) == 3 && metric.Label[0].GetValue() == gateway &&
			metric.Label[1].GetValue() == stage && metric.Label[2].GetValue() == resourceType {
			return metric.Counter.GetValue()
		}
	}
	return 0
}

func GetResourceConvertedCountMetric(
	metrics map[string]*dto.MetricFamily, gateway string, stage string, resourceType string) float64 {
	resourceConvertedCountMetric := metrics[ResourceEventTriggeredCountMetric]
	if resourceConvertedCountMetric == nil {
		return 0
	}
	for _, metric := range resourceConvertedCountMetric.Metric {
		if len(metric.Label) == 3 && metric.Label[0].GetValue() == gateway &&
			metric.Label[1].GetValue() == stage && metric.Label[2].GetValue() == resourceType {
			return metric.Counter.GetValue()
		}
	}
	return 0
}

func GetResourceSyncCmpCountMetrics(
	metrics map[string]*dto.MetricFamily, gateway string, stage string, resourceType string) float64 {
	resourceSyncCmpCountMetric := metrics[ResourceSyncCmpCount]
	if resourceSyncCmpCountMetric == nil {
		return 0
	}
	for _, metric := range resourceSyncCmpCountMetric.Metric {
		if len(metric.Label) == 3 && metric.Label[0].GetValue() == gateway &&
			metric.Label[1].GetValue() == stage && metric.Label[2].GetValue() == resourceType {
			return metric.Counter.GetValue()
		}
	}
	return 0
}

func GetResourceSyncCmpDiffCountMetrics(
	metrics map[string]*dto.MetricFamily, gateway string, stage string, resourceType string) float64 {
	resourceSyncCmpDiffCountMetric := metrics[ResourceSyncCmpDiffCount]
	if resourceSyncCmpDiffCountMetric == nil {
		return 0
	}
	for _, metric := range resourceSyncCmpDiffCountMetric.Metric {
		if len(metric.Label) == 3 && metric.Label[0].GetValue() == gateway &&
			metric.Label[1].GetValue() == stage && metric.Label[2].GetValue() == resourceType {
			return metric.Counter.GetValue()
		}
	}
	return 0
}

func GetApisixOperationCountMetric(
	metrics map[string]*dto.MetricFamily, action string, result string, resourceType string) float64 {
	apisixOperationCountMetric := metrics[ApisixOperationCountMetric]
	if apisixOperationCountMetric == nil {
		return 0
	}
	for _, metric := range apisixOperationCountMetric.Metric {
		if len(metric.Label) == 3 && metric.Label[0].GetValue() == action &&
			metric.Label[1].GetValue() == result && metric.Label[2].GetValue() == resourceType {
			return metric.Counter.GetValue()
		}
	}
	return 0
}

func GetBootstrapSyncingSuccessCountMetric(metrics map[string]*dto.MetricFamily) float64 {
	bootstrapSyncingCountMetric := metrics[BootstrapSyncingCountMetric]
	if bootstrapSyncingCountMetric == nil {
		return 0
	}
	for _, metric := range bootstrapSyncingCountMetric.Metric {
		if len(metric.Label) == 1 && metric.Label[0].GetValue() == "succ" {
			return metric.Counter.GetValue()
		}
	}
	return 0
}
