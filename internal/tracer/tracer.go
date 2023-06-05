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
	"os"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/config"
	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/utils"
)

// InitTracing ...
func InitTracing(ctx context.Context, opts *Options) error {
	enableFlag = true
	var exporter sdktrace.SpanExporter
	var err error
	switch opts.ExporterMode {
	case ExporterHTTP:
		exporter, err = newHTTPExporter(ctx, opts)
	case ExporterSTDOUT:
		exporter, err = newStdoutExporter()
	default:
		exporter, err = newGrpcExporter(ctx, opts)
	}
	if err != nil {
		return err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(mustNewResource(opts.BkMonitorAPMToken)),
	)
	otel.SetTracerProvider(tp)
	otel.SetLogger(ctrl.Log.WithName("otel"))
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)
	return nil
}

func newGrpcExporter(ctx context.Context, opts *Options) (sdktrace.SpanExporter, error) {
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(opts.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
}

func newHTTPExporter(ctx context.Context, opts *Options) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(opts.Endpoint),
		otlptracehttp.WithURLPath(opts.URLPath),
		otlptracehttp.WithInsecure(),
	)
}

func newStdoutExporter() (sdktrace.SpanExporter, error) {
	return stdout.New(stdout.WithPrettyPrint())
}

func mustNewResource(token string) *resource.Resource {
	var err error
	var r *resource.Resource
	hostname, _ := os.Hostname()
	r, err = resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.HostNameKey.String(hostname),
			semconv.ServiceNameKey.String("bk-micro-gateway-operator"),
			semconv.ServiceInstanceIDKey.String(utils.GetGeneratedUUID()),
			semconv.ServiceVersionKey.String(version.Version),
			attribute.String("bk.data.token", token),
		),
	)
	if err != nil {
		panic(err)
	}

	withKube := config.InstanceName != "" && config.InstanceNamespace != ""
	if withKube {
		r, err = resource.Merge(r, resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.K8SPodNameKey.String(config.InstanceName),
			attribute.String("service.instance.name", config.InstanceName),
			semconv.K8SNamespaceNameKey.String(config.InstanceNamespace),
			semconv.ServiceNamespaceKey.String(config.InstanceNamespace),
		))
	} else {
		r, err = resource.Merge(r, resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String("service.instance.name", config.InstanceName),
		))
	}

	if err != nil {
		panic(err)
	}
	return r
}
