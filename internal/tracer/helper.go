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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	enableFlag          bool
	tph                 = TracerProviderHelper{}
	defaultTracerHelper = &TracerHelper{}
	defaultSpanHelper   = &SpanHelper{}
)

// TracerProviderHelper ...
type TracerProviderHelper struct{}

// TracerHelper ...
type TracerHelper struct {
	trace.Tracer
}

// SpanHelper ...
type SpanHelper struct {
	trace.Span
}

// NewTracer ...
func NewTracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if !enableFlag {
		return defaultTracerHelper
	}
	return &TracerHelper{
		Tracer: otel.GetTracerProvider().Tracer(name, opts...),
	}
}

// Tracer ...
func (tp TracerProviderHelper) Tracer(instrumentationName string, opts ...trace.TracerOption) trace.Tracer {
	return NewTracer(instrumentationName, opts...)
}

// Start ...
func (t *TracerHelper) Start(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	if !enableFlag {
		return ctx, defaultSpanHelper
	}
	ctx, span := t.Tracer.Start(ctx, spanName, opts...)
	return ctx, &SpanHelper{Span: span}
}

// End ...
func (s *SpanHelper) End(options ...trace.SpanEndOption) {
	if !enableFlag {
		return
	}
	s.Span.End(options...)
}

// AddEvent ...
func (s *SpanHelper) AddEvent(name string, options ...trace.EventOption) {
	if !enableFlag {
		return
	}
	s.Span.AddEvent(name, options...)
}

// IsRecording ...
func (s *SpanHelper) IsRecording() bool {
	if !enableFlag {
		return true
	}
	return s.Span.IsRecording()
}

// RecordError ...
func (s *SpanHelper) RecordError(err error, options ...trace.EventOption) {
	if !enableFlag {
		return
	}
	s.Span.RecordError(err, options...)
}

// SpanContext ...
func (s *SpanHelper) SpanContext() trace.SpanContext {
	if !enableFlag {
		return trace.SpanContext{}
	}
	return s.Span.SpanContext()
}

// SetStatus ...
func (s *SpanHelper) SetStatus(code codes.Code, description string) {
	if !enableFlag {
		return
	}
	s.Span.SetStatus(code, description)
}

// SetName ...
func (s *SpanHelper) SetName(name string) {
	if !enableFlag {
		return
	}
	s.Span.SetName(name)
}

// SetAttributes ...
func (s *SpanHelper) SetAttributes(kv ...attribute.KeyValue) {
	if !enableFlag {
		return
	}
	s.Span.SetAttributes(kv...)
}

// TracerProvider ...
func (s *SpanHelper) TracerProvider() trace.TracerProvider {
	return tph
}
