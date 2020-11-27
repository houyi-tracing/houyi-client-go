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
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"time"
)

const (
	defaultRefreshInterval   = time.Minute
	defaultSamplingServerURL = "http://localhost:5778/api/strategy"
)

var AdaptiveSamplerOptions AdaptiveSamplerOptionsFactory

type AdaptiveSamplerOption func(options *adaptiveSamplerOptions)

type AdaptiveSamplerOptionsFactory struct{}

type adaptiveSamplerOptions struct {
	metrics                 *jaeger.Metrics
	sampler                 jaeger.SamplerV2
	logger                  *zap.Logger
	samplingServerURL       string
	samplingRefreshInterval time.Duration
	samplingFetcher         SamplingStrategyFetcher
	samplingParser          jaeger.SamplingStrategyParser
	updaters                []jaeger.SamplerUpdater
}

// Metrics creates a AdaptiveSamplerOption that initializes Metrics on the sampler,
// which is used to emit statistics.
func (AdaptiveSamplerOptionsFactory) Metrics(m *jaeger.Metrics) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.metrics = m
	}
}

// InitialSampler creates a AdaptiveSamplerOption that sets the initial sampler
// to use before a remote sampler is created and used.
func (AdaptiveSamplerOptionsFactory) InitialSampler(sampler jaeger.SamplerV2) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.sampler = sampler
	}
}

// Logger creates a AdaptiveSamplerOption that sets the logger used by the sampler.
func (AdaptiveSamplerOptionsFactory) Logger(logger *zap.Logger) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.logger = logger
	}
}

// SamplingServerURL creates a AdaptiveSamplerOption that sets the sampling server url
// of the local agent that contains the sampling strategies.
func (AdaptiveSamplerOptionsFactory) SamplingServerURL(samplingServerURL string) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.samplingServerURL = samplingServerURL
	}
}

// SamplingRefreshInterval creates a AdaptiveSamplerOption that sets how often the
// sampler will poll local agent for the appropriate sampling strategy.
func (AdaptiveSamplerOptionsFactory) SamplingRefreshInterval(samplingRefreshInterval time.Duration) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.samplingRefreshInterval = samplingRefreshInterval
	}
}

// SamplingStrategyFetcher creates a AdaptiveSamplerOption that initializes sampling strategy fetcher.
func (AdaptiveSamplerOptionsFactory) SamplingStrategyFetcher(fetcher SamplingStrategyFetcher) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.samplingFetcher = fetcher
	}
}

// SamplingStrategyParser creates a AdaptiveSamplerOption that initializes sampling strategy parser.
func (AdaptiveSamplerOptionsFactory) SamplingStrategyParser(parser jaeger.SamplingStrategyParser) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.samplingParser = parser
	}
}

// Updaters creates a AdaptiveSamplerOption that initializes sampler updaters.
func (AdaptiveSamplerOptionsFactory) Updaters(updaters ...jaeger.SamplerUpdater) AdaptiveSamplerOption {
	return func(o *adaptiveSamplerOptions) {
		o.updaters = updaters
	}
}

func (o *adaptiveSamplerOptions) apply(opts ...AdaptiveSamplerOption) *adaptiveSamplerOptions {
	for _, option := range opts {
		option(o)
	}

	if o.metrics == nil {
		o.metrics = jaeger.NewNullMetrics()
	}
	if o.sampler == nil {
		o.sampler, _ = jaeger.NewProbabilisticSampler(1)
	}
	if o.logger == nil {
		o.logger = zap.NewNop()
	}
	if o.samplingServerURL == "" {
		o.samplingServerURL = defaultSamplingServerURL
	}
	if o.samplingRefreshInterval == 0 {
		o.samplingRefreshInterval = defaultRefreshInterval
	}
	if o.samplingFetcher == nil {
		o.samplingFetcher = NewSamplingStrategyFetcher(o.samplingServerURL, o.logger)
	}
	if o.samplingParser == nil {
		o.samplingParser = NewSamplingStrategyParser()
	}
	if o.updaters == nil {
		o.updaters = []jaeger.SamplerUpdater{
			NewSamplerUpdater(),
		}
	}
	return o
}
