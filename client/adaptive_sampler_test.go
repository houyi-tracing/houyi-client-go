// Copyright (c) 2020 Fuhai Yan.
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

package client

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/log"
	"io"
	"testing"
	"time"
)

const (
	udpAgentHostPort = "localhost:6831"
	serviceName      = "my-demo"
)

var tracer opentracing.Tracer
var closer io.Closer
var sampler jaeger.SamplerV2
var logger log.Logger
var reporter jaeger.Reporter

func TestAdaptiveSampler(t *testing.T) {
	sampler = jaeger.NewConstSampler(true)
	logger = log.StdLogger

	adaptiveSampler := NewAdaptiveSampler(
		serviceName,
		AdaptiveSamplerOptions.InitialSampler(sampler),
		AdaptiveSamplerOptions.SamplingRefreshInterval(time.Minute*10))

	//reporter = jaeger.NewLoggingReporter(logger)
	udpTransport, err := jaeger.NewUDPTransport(udpAgentHostPort, 65000)
	if err != nil {
		logger.Error(err.Error())
	}
	reporter = jaeger.NewRemoteReporter(
		udpTransport,
		jaeger.ReporterOptions.Logger(logger),
		jaeger.ReporterOptions.BufferFlushInterval(time.Second*1))

	tracer, closer = jaeger.NewTracer(
		serviceName,
		adaptiveSampler,
		reporter,
		jaeger.TracerOptions.Logger(logger))

	for i := 0; i < 100; i++ {
		span := tracer.StartSpan("no-op")
		span.Finish()
	}

	adaptiveSampler.UpdateSampler()

	if err := closer.Close(); err != nil {
		logger.Error("failed to close tracer")
	}
}
