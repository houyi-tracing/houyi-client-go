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
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io"
)

type Factory interface {
	InitFromViper(*viper.Viper)
	CreateTracer(serviceName string, logger *zap.Logger) (opentracing.Tracer, io.Closer)
}

type tracerFactory struct {
	Options
}

func NewTracerFactory() Factory {
	return &tracerFactory{}
}

func (f *tracerFactory) InitFromViper(v *viper.Viper) {
	f.Options.InitFromViper(v)
}

func (f *tracerFactory) CreateTracer(serviceName string, logger *zap.Logger) (opentracing.Tracer, io.Closer) {
	sampler := f.createSampler(serviceName, logger)
	reporter, err := f.createReporter(logger)
	if err != nil {
		logger.Fatal("Failed to create reporter", zap.Error(err))
		return nil, nil
	}
	tracer := NewTracer(serviceName, &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})
	return tracer, tracer
}

func (f *tracerFactory) createSampler(serviceName string, logger *zap.Logger) Sampler {
	return NewRemoteSampler(&RemoteSamplerParams{
		Logger:       logger,
		ServiceName:  serviceName,
		PullInterval: f.PullStrategiesInterval,
		AgentEndpoint: routing.Endpoint{
			Addr: f.AgentAddr,
			Port: f.AgentPort,
		},
	})
}

func (f *tracerFactory) createReporter(logger *zap.Logger) (Reporter, error) {
	switch f.ReporterType {
	case ReporterType_Null:
		return NewNullReporter(), nil
	case ReporterType_Logging:
		return NewLogReporter(logger), nil
	case ReporterType_Remote:
		transport := NewTransport(&TransportParams{
			Logger:          logger,
			MaxBufferedSize: f.MaxBufferedSize,
			AgentEndpoint: routing.Endpoint{
				Addr: f.AgentAddr,
				Port: f.AgentPort,
			},
		})
		return NewRemoteReporter(&RemoteReporterParams{
			Logger:    logger,
			QueueSize: f.QueueSize,
			Interval:  f.BufferRefreshInterval,
			Transport: transport,
		}), nil
	default:
		return nil, fmt.Errorf("unsupportted type of reporter: %s", f.ReporterType)
	}
}
