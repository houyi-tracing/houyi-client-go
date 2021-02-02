// Copyright (c) 2021 The Houyi Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package houyi

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"testing"
)

func TestStartSpan(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	reporter := NewLogReporter(logger)
	sampler := NewConstSampler(true)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	span := tracer.StartSpan("op").(*Span)
	assert.NotNil(t, span.tracer)
	assert.NotNil(t, span.context)
	assert.Equal(t, span.context.traceID.Low, uint64(span.context.spanID))
	assert.True(t, span.isIngress)
	assert.True(t, span.SpanContext().IsSampled())
	assert.Equal(t, "op", span.operationName)

	span.Finish()

	childSpan := tracer.StartSpan("op1", opentracing.ChildOf(span.context)).(*Span)
	sc := childSpan.context
	assert.Equal(t, span.context.traceID.Low, sc.traceID.Low)
	assert.Equal(t, span.context.traceID.High, sc.traceID.High)
	assert.False(t, childSpan.isIngress)
	assert.True(t, childSpan.context.IsSampled())

	childSpan.Finish()
}

func TestPropagation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	reporter := NewLogReporter(logger)
	sampler := NewConstSampler(true)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	span := tracer.StartSpan("op").(*Span)
	span.SetBaggageItem("baggage-key", "baggage-value")
	sc := span.context

	httpHeaders := http.Header{}
	carrier := opentracing.HTTPHeadersCarrier(httpHeaders)
	err := tracer.Inject(sc, opentracing.HTTPHeaders, carrier)
	assert.Nil(t, err)

	newSc, err := tracer.Extract(opentracing.HTTPHeaders, carrier)
	assert.Nil(t, err)
	assert.Equal(t, newSc.(SpanContext).String(), sc.String())

	fmt.Println(newSc.(SpanContext).String())
	fmt.Println(sc.String())
}

func BenchmarkStartSpan(b *testing.B) {
	logger, _ := zap.NewDevelopment()

	reporter := NewNullReporter()
	sampler := NewProbabilitySampler(0.2)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	for i := 0; i < b.N; i++ {
		tracer.StartSpan("op")
	}
}
