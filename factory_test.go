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

package houyi

import (
	"flag"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"testing"
)

var logger *zap.Logger

const (
	serviceName = "svc"
	op          = "NO-OP"
)

func init() {
	logger, _ = zap.NewProduction()
}

func TestDefaultFlags(t *testing.T) {
	cmd := &cobra.Command{}
	flagSet := new(flag.FlagSet)
	v := viper.New()

	_ = AddCmdFlags(v, cmd, flagSet)

	assert.Equal(t, DefaultSamplerType, v.GetString(samplerType))
	assert.Equal(t, DefaultAlwaysSample, v.GetBool(alwaysSample))
	assert.Equal(t, DefaultSamplingRate, v.GetFloat64(samplingRate))
	assert.Equal(t, DefaultRefreshInterval, v.GetDuration(refreshInterval))
	assert.Equal(t, DefaultMaxTracesPerSecond, v.GetInt(maxTracesPerSecond))
	assert.Equal(t, DefaultStrategyURI, v.GetString(strategyURI))

	assert.Equal(t, DefaultReporterType, v.GetString(reporterType))
	assert.Equal(t, DefaultBufferRefreshInterval, v.GetDuration(bufferRefreshInterval))
	assert.Equal(t, DefaultAgentHost, v.GetString(agentHost))
	assert.Equal(t, DefaultAgentGRPCPort, v.GetInt(agentGRPCPort))
	assert.Equal(t, DefaultAgentHttpPort, v.GetInt(agentHttpPort))
	assert.Equal(t, DefaultUdpMaxPacketSize, v.GetInt(udpMaxPacketSize))
}

func TestConstSampler(t *testing.T) {
	f := NewFactory(logger)
	cmd := &cobra.Command{}
	flagSet := new(flag.FlagSet)
	v := viper.New()
	_ = AddCmdFlags(v, cmd, flagSet)

	v.Set(samplerType, SamplerTypeConst)
	v.Set(alwaysSample, true)
	f.InitFromViper(v)

	tracer, closer := f.CreateTracer(serviceName)
	_, ok := tracer.(*jaeger.Tracer)
	assert.True(t, ok)
	_, ok = tracer.(*jaeger.Tracer).Sampler().(*jaeger.ConstSampler)
	assert.True(t, ok)
	assert.NotNil(t, tracer)
	assert.NotNil(t, closer)

	span := tracer.StartSpan(op).(*jaeger.Span)
	span.Finish()
	assert.True(t, span.SpanContext().IsSampled())
	assert.Equal(t, op, span.OperationName())

	if err := closer.Close(); err != nil {
		logger.Error("", zap.Error(err))
	}
}

func TestProbabilitySampler(t *testing.T) {
	f := NewFactory(logger)
	cmd := &cobra.Command{}
	flagSet := new(flag.FlagSet)
	v := viper.New()
	_ = AddCmdFlags(v, cmd, flagSet)

	v.Set(samplerType, SamplerTypeProbability)
	v.Set(samplingRate, 1)
	f.InitFromViper(v)

	tracer, closer := f.CreateTracer(serviceName)
	_, ok := tracer.(*jaeger.Tracer)
	assert.True(t, ok)
	_, ok = tracer.(*jaeger.Tracer).Sampler().(*jaeger.ProbabilisticSampler)
	assert.True(t, ok)
	assert.NotNil(t, tracer)
	assert.NotNil(t, closer)

	span := tracer.StartSpan(op).(*jaeger.Span)
	span.Finish()
	assert.True(t, span.SpanContext().IsSampled())

	assert.Nil(t, closer.Close())

	v.Set(samplingRate, 0)
	f.InitFromViper(v)
	tracer, closer = f.CreateTracer(serviceName)
	span = tracer.StartSpan(op).(*jaeger.Span)
	span.Finish()
	assert.False(t, span.SpanContext().IsSampled())

	assert.Nil(t, closer.Close())
}
