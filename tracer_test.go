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
	"github.com/houyi-tracing/houyi/pkg/routing"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"testing"
	"time"
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

func TestHttpHeadersPropagation(t *testing.T) {
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

func TestTracerReportSpans(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	transport := NewTransport(&TransportParams{
		Logger:          logger,
		MaxBufferedSize: 1000,
		AgentEndpoint: routing.Endpoint{
			Addr: "192.168.31.204",
			Port: 14680,
		},
	})
	reporter := NewRemoteReporter(&RemoteReporterParams{
		Logger:    logger,
		QueueSize: 1000,
		Interval:  time.Second,
		Transport: transport,
	})

	sampler := NewProbabilitySampler(0.1)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	n := 100
	for i := 0; i < n; i++ {
		s := tracer.StartSpan("houyi_test_op")
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)))
		s.Finish()
	}

	time.Sleep(time.Minute)
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

func TestExtractCallerReceiverRelationship(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	reporter := NewLogReporter(logger)
	sampler := NewConstSampler(true)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	var err error

	sp0 := tracer.StartSpan("op-a")
	defer sp0.Finish()

	headers := http.Header{}
	carrier := opentracing.HTTPHeadersCarrier(headers)
	err = tracer.Inject(sp0.Context(), opentracing.HTTPHeaders, carrier)
	assert.Nil(t, err)
	ctx, err := tracer.Extract(opentracing.HTTPHeaders, carrier)
	assert.Nil(t, err)

	sp1 := tracer.StartSpan("op-b", opentracing.ChildOf(ctx))
	defer sp1.Finish()

	headers = http.Header{}
	carrier = opentracing.HTTPHeadersCarrier(headers)
	err = tracer.Inject(sp1.Context(), opentracing.HTTPHeaders, carrier)
	assert.Nil(t, err)
	ctx, err = tracer.Extract(opentracing.HTTPHeaders, carrier)
	assert.Nil(t, err)

	sp2 := tracer.StartSpan("op-c", opentracing.ChildOf(ctx))
	defer sp2.Finish()
}
