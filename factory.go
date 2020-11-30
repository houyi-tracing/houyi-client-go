// Copyright (c) 2020 The Houyi Authors.
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
	"github.com/houyi-tracing/houyi-client-go/client"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"io"
	"time"
)

const (
	SamplerTypeConst       = "const"
	SamplerTypeAdaptive    = "adaptive"
	SamplerTypeProbability = "probability"
	SamplerTypeRateLimit   = "rate-limit"
	SamplerTypeUnknown     = "unknown"
)

const (
	ReporterTypeNull    = "null"
	ReporterTypeRemote  = "remote"
	ReporterTypeLogging = "logging"
	ReporterTypeUnknown = "unknown"
)

type tracerFactory struct {
	logger *zap.Logger

	// type of sampler
	samplerType string

	// Refresh interval of buffer to report spans to agent
	bufferRefreshInterval time.Duration

	// const sampler
	alwaysSample bool

	// probability sampler
	samplingRate float64

	// rate-limit sampler
	maxTracesPerSecond float64

	// dynamic strategy sampler
	refreshInterval time.Duration
	agentHost       string
	agentHttpPort   int
	strategyURI     string

	// gRPC port for agent to receive spans
	agentGRPCPort int

	// type of reporter
	reporterType string

	// udp max packet size
	udpMaxPacketSize int
}

func NewFactory() *tracerFactory {
	return &tracerFactory{}
}

func (t *tracerFactory) InitFromViper(v *viper.Viper, logger *zap.Logger) {
	t.logger = logger

	options := new(Options).InitFromViper(v)
	t.samplerType = options.SamplerType
	t.bufferRefreshInterval = options.BufferRefreshInterval

	switch options.SamplerType {
	case SamplerTypeProbability:
		t.samplingRate = options.SamplingRate
	case SamplerTypeConst:
		t.alwaysSample = options.AlwaysSample
	case SamplerTypeAdaptive:
		t.refreshInterval = options.RefreshInterval
		t.strategyURI = options.StrategyURI
	case SamplerTypeRateLimit:
		t.maxTracesPerSecond = options.MaxTracesPerSecond
	default:
		t.logger.Fatal("unknown sampler type", zap.String("sampler.type", options.SamplerType))
		t.samplerType = SamplerTypeUnknown
	}

	t.reporterType = options.ReporterType

	if t.samplerType == SamplerTypeAdaptive && t.reporterType != ReporterTypeRemote {
		t.logger.Fatal("adaptive sampler must work with remote reporter")
		return
	}

	switch options.ReporterType {
	case ReporterTypeNull, ReporterTypeLogging:
	// do nothing
	case ReporterTypeRemote:
		t.agentHost = options.AgentHost
		t.agentGRPCPort = options.AgentGRPCPort
		t.agentHttpPort = options.AgentHttpPort
	default:
		t.logger.Error("unknown reporter type", zap.String("reporter.type", options.ReporterType))
		t.reporterType = ReporterTypeUnknown
	}
}

func (t *tracerFactory) CreateTracer(serviceName string) (opentracing.Tracer, io.Closer) {
	sampler := t.createSampler(serviceName)
	reporter := t.createReporter()
	return jaeger.NewTracer(serviceName, sampler, reporter)
}

func (t *tracerFactory) createSampler(serviceName string) jaeger.Sampler {
	var sampler jaeger.Sampler
	var err error

	switch t.samplerType {
	case SamplerTypeProbability:
		sampler, err = jaeger.NewProbabilisticSampler(t.samplingRate)
		if err != nil {
			t.logger.Error("failed to new probability", zap.Error(err))
		}
	case SamplerTypeConst:
		sampler = jaeger.NewConstSampler(t.alwaysSample)
	case SamplerTypeAdaptive:
		initSampler := jaeger.NewConstSampler(true)
		sampler = client.NewAdaptiveSampler(
			serviceName,
			client.AdaptiveSamplerOptions.Logger(t.logger),
			client.AdaptiveSamplerOptions.InitialSampler(initSampler),
			client.AdaptiveSamplerOptions.SamplingRefreshInterval(t.refreshInterval),
			client.AdaptiveSamplerOptions.SamplingServerURL(
				toURL(t.agentHost, t.agentHttpPort, t.strategyURI)))
	case SamplerTypeRateLimit:
		sampler = jaeger.NewRateLimitingSampler(t.maxTracesPerSecond)
	case SamplerTypeUnknown:
		t.logger.Error("unknown sampler type, use const sampler (sample=false) by default")
		sampler = jaeger.NewConstSampler(false)
		t.samplerType = SamplerTypeConst
	}

	t.logger.Info("created sampler", zap.String("sample.type", t.samplerType))
	return sampler
}

func (t *tracerFactory) createReporter() jaeger.Reporter {
	var reporter jaeger.Reporter

	switch t.reporterType {
	case ReporterTypeNull, ReporterTypeUnknown:
		reporter = jaeger.NewNullReporter()
	case ReporterTypeLogging:
		reporter = jaeger.NewLoggingReporter(newInnerLogger(t.logger))
	case ReporterTypeRemote:
		udpHostPort := toHostPort(t.agentHost, t.agentGRPCPort)
		udpTransport, err := jaeger.NewUDPTransport(udpHostPort, 1000)
		if err != nil {
			t.logger.Fatal("failed to new UDP transport", zap.Error(err))
		}
		reporter = jaeger.NewRemoteReporter(
			udpTransport,
			jaeger.ReporterOptions.BufferFlushInterval(t.bufferRefreshInterval))
		t.logger.Info("created udp transport", zap.String("host", t.agentHost), zap.Int("port", t.agentGRPCPort))
	}

	t.logger.Info("created reporter", zap.String("reporter.type", t.reporterType))
	return reporter
}

func toHostPort(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

func toURL(host string, port int, uri string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, uri)
}

// convert zap.Logger to jaeger.Logger
type innerLogger struct {
	*zap.Logger
}

func newInnerLogger(logger *zap.Logger) jaeger.Logger {
	return &innerLogger{logger}
}

func (i *innerLogger) Error(msg string) {
	i.Logger.Error(msg)
}

func (i *innerLogger) Infof(msg string, _ ...interface{}) {
	i.Logger.Info(msg)
}
