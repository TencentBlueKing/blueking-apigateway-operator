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

package tracer

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

var _ = Describe("Tracer", func() {
})

func BenchmarkStartEmptySpan(b *testing.B) {
	InitTracing(context.Background(), &Options{
		ExporterMode: ExporterGRPC,
	})
	tp := otel.GetTracerProvider().Tracer("test")
	for n := 0; n < b.N; n++ {
		_, span := tp.Start(context.Background(), "span", trace.WithAttributes(semconv.ServiceVersionKey.Int(n)))
		span.End()
	}
}

func BenchmarkStartSpanWithEvent(b *testing.B) {
	InitTracing(context.Background(), &Options{
		ExporterMode: ExporterGRPC,
	})
	tp := otel.GetTracerProvider().Tracer("test")
	for n := 0; n < b.N; n++ {
		_, span := tp.Start(context.Background(), "span", trace.WithAttributes(semconv.ServiceVersionKey.Int(n)))
		span.AddEvent("something", trace.WithAttributes(semconv.ServiceVersionKey.Int(n)))
		span.End()
	}
}

func BenchmarkBenchmark(b *testing.B) {
	InitTracing(context.Background(), &Options{
		ExporterMode: ExporterGRPC,
	})
	for n := 0; n < b.N; n++ {
		_ = n << 1
	}
}
