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
	"github.com/houyi-tracing/houyi/idl/api_v1"
	"github.com/houyi-tracing/houyi/pkg/routing"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

const (
	InnerSamplerTypeKey  = "inner.sampler.type"
	InnerSamplerParamKey = "inner.sampler.param"
)

const (
	RemoteSampler_Adaptive = iota
	RemoteSampler_Dynamic
)

type RemoteSamplerParams struct {
	Logger        *zap.Logger
	ServiceName   string
	PullInterval  time.Duration
	Type          int
	AgentEndpoint routing.Endpoint
}

// RemoteSampler implements a type of sampler that pulls sampling strategies from remote strategy manger.
//
// Strategies that RemoteSampler supports to are:
//
// - Adaptive: An operation with higher QPS(reQuest Per Second) would gain a lower sampling rate.
//
// - Dynamic: An operation evaluated with high priority would gain a higher sampling rate. The way to evaluate
//
// an operation is to check whether the span of this operation has high-valued tags. For details,
// see in https://github.com/houyi-tracing/houyi/tree/master/pkg/evaluator.
type RemoteSampler struct {
	sync.RWMutex
	logger       *zap.Logger
	serviceName  string
	samplerType  int // adaptive or dynamic
	pullInterval time.Duration
	strategies   map[string]*PerOperationSampler
	qpsStat      map[string]*Throughput
	fetcher      SamplingStrategyFetcher
	updater      SamplerUpdater
	stopCh       chan *sync.WaitGroup
}

func NewRemoteSampler(logger *zap.Logger, params *RemoteSamplerParams) Sampler {
	s := &RemoteSampler{
		logger:       logger,
		serviceName:  params.ServiceName,
		pullInterval: params.PullInterval,
		strategies:   make(map[string]*PerOperationSampler),
		qpsStat:      make(map[string]*Throughput),
		samplerType:  params.Type,
		stopCh:       make(chan *sync.WaitGroup),
	}

	switch params.Type {
	case RemoteSampler_Adaptive:
		s.fetcher = NewSamplingStrategyFetcher(api_v1.StrategyType_ADAPTIVE, params.AgentEndpoint)
		s.updater = NewAdaptiveSamplerUpdater()
	case RemoteSampler_Dynamic:
		s.fetcher = NewSamplingStrategyFetcher(api_v1.StrategyType_DYNAMIC, params.AgentEndpoint)
		s.updater = NewDynamicSamplerUpdater()
	}

	go s.timer()
	return s
}

func (s *RemoteSampler) OnCreateSpan(span *Span) SamplingDecision {
	s.Lock()
	defer s.Unlock()

	op := span.OperationName()
	if qpsI, ok := s.qpsStat[op]; ok {
		qpsI.Throughput += 1
	} else {
		s.qpsStat[op] = &Throughput{
			Since:      time.Now(),
			Throughput: 1,
		}
		s.strategies[op] = &PerOperationSampler{
			operation: op,
			sampler:   NewProbabilitySampler(1.0),
		}
	}

	sampled, tags := s.trySampling(span)
	return SamplingDecision{
		Sampled: sampled,
		Tag:     tags,
	}
}

func (s *RemoteSampler) IsSampled(span *Span) (bool, []opentracing.Tag) {
	return span.context.IsSampled(), span.Tags()
}

func (s *RemoteSampler) UpdateSampler(operation string, sampler Sampler) {
	s.Lock()
	defer s.Unlock()

	s.strategies[operation] = &PerOperationSampler{
		operation: operation,
		sampler:   sampler,
	}
}

func (s *RemoteSampler) String() string {
	var builder strings.Builder
	builder.WriteString("remote_sampler[")
	for op, strategy := range s.strategies {
		builder.WriteString(fmt.Sprintf("(operation=%s,sampler=%s)", op, strategy.String()))
	}
	builder.WriteString("]")
	return builder.String()
}

func (s *RemoteSampler) trySampling(span *Span) (bool, []opentracing.Tag) {
	op := span.OperationName()
	if sampler, ok := s.strategies[op]; ok {
		decision := sampler.OnCreateSpan(span)
		return decision.Sampled, s.convertTags(decision.Tag)
	} else {
		return false, make([]opentracing.Tag, 0)
	}
}

func (s *RemoteSampler) timer() {
	ticker := time.NewTicker(s.pullInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.Lock()
			s.update()
			s.Unlock()
		case wg := <-s.stopCh:
			wg.Done()
			return
		}
	}
}

func (s *RemoteSampler) update() {
	operations := make([]Operation, 0, len(s.qpsStat))
	now := time.Now()

	for op, item := range s.qpsStat {
		operations = append(operations, Operation{
			Service: s.serviceName,
			Name:    op,
			Qps:     float64(item.Throughput) / now.Sub(item.Since).Seconds(),
		})
		item.Throughput = 0
		item.Since = now
	}

	strategyResp, err := s.fetcher.Fetch(s.serviceName, operations)
	if err != nil {
		s.logger.Error("failed to fetch strategy", zap.Error(err))
		return
	}

	if _, err := s.updater.Update(s, strategyResp); err != nil {
		s.logger.Error("failed to update sampler", zap.Error(err))
	} else {
		s.logger.Debug("updated sampler", zap.Stringer("new sampler", s))
	}
}

func (s *RemoteSampler) convertTags(tags []opentracing.Tag) []opentracing.Tag {
	var sType string
	if s.samplerType == RemoteSampler_Adaptive {
		sType = SamplerTypeAdaptive
	} else if s.samplerType == RemoteSampler_Dynamic {
		sType = SamplerTypeDynamic
	}
	newTags := []opentracing.Tag{
		{
			Key:   SamplerTypeKey,
			Value: sType,
		},
	}

	for _, t := range tags {
		if t.Key == SamplerTypeKey {
			newTags = append(newTags, opentracing.Tag{
				Key:   InnerSamplerTypeKey,
				Value: t.Value,
			})
		} else if t.Key == SamplerParamKey {
			newTags = append(newTags, opentracing.Tag{
				Key:   InnerSamplerParamKey,
				Value: t.Value,
			})
		} else {
			newTags = append(newTags, opentracing.Tag{
				Key:   t.Key,
				Value: t.Value,
			})
		}
	}
	return newTags
}

func (s *RemoteSampler) Close() error {
	for _, s := range s.strategies {
		_ = s.Close()
	}
	return nil
}
