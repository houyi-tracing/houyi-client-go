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

package client

import (
	"fmt"
	"github.com/houyi-tracing/houyi-client-go/client/model"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

type qpsEntry struct {
	qps        float64
	throughput int64
	endTime    time.Time
}

type AdaptiveSampler struct {
	adaptiveSamplerOptions
	sync.RWMutex

	closed      int64
	serviceName string
	qps         map[string]*qpsEntry
	doneChan    chan *sync.WaitGroup
}

func NewAdaptiveSampler(serviceName string, opts ...AdaptiveSamplerOption) *AdaptiveSampler {
	options := new(adaptiveSamplerOptions).apply(opts...)
	sampler := &AdaptiveSampler{
		adaptiveSamplerOptions: *options,
		qps:                    make(map[string]*qpsEntry),
		serviceName:            serviceName,
		doneChan:               make(chan *sync.WaitGroup),
	}

	go sampler.pollController()
	return sampler
}

func (s *AdaptiveSampler) IsSampled(_ jaeger.TraceID, _ string) (sampled bool, tags []jaeger.Tag) {
	return false, nil
}

func (s *AdaptiveSampler) Equal(_ jaeger.Sampler) bool {
	return false
}

func (s *AdaptiveSampler) OnCreateSpan(span *jaeger.Span) jaeger.SamplingDecision {
	s.Lock()
	defer s.Unlock()
	if qE, has := s.qps[span.OperationName()]; has {
		qE.throughput += 1
	} else {
		s.qps[span.OperationName()] = &qpsEntry{
			qps:        1,
			throughput: 1,
			endTime:    time.Now(),
		}
	}
	return s.sampler.OnCreateSpan(span)
}

func (s *AdaptiveSampler) OnSetOperationName(span *jaeger.Span, operationName string) jaeger.SamplingDecision {
	s.RLock()
	defer s.RUnlock()
	return s.sampler.OnSetOperationName(span, operationName)
}

func (s *AdaptiveSampler) OnSetTag(span *jaeger.Span, key string, value interface{}) jaeger.SamplingDecision {
	s.RLock()
	defer s.RUnlock()
	return s.sampler.OnSetTag(span, key, value)
}

func (s *AdaptiveSampler) OnFinishSpan(span *jaeger.Span) jaeger.SamplingDecision {
	s.RLock()
	defer s.RUnlock()
	return s.sampler.OnFinishSpan(span)
}

func (s *AdaptiveSampler) Close() {
	if swapped := atomic.CompareAndSwapInt64(&s.closed, 0, 1); !swapped {
		s.logger.Error("Repeated attempt to close the sampler is ignored")
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	s.doneChan <- &wg
	wg.Wait()
}

func (s *AdaptiveSampler) pollController() {
	ticker := time.NewTicker(s.samplingRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.UpdateSampler()
		case wg := <-s.doneChan:
			wg.Done()
			return
		}
	}
}

func (s *AdaptiveSampler) Sampler() jaeger.SamplerV2 {
	s.RLock()
	defer s.RUnlock()
	return s.sampler
}

func (s *AdaptiveSampler) UpdateSampler() {
	s.Lock()
	defer s.Unlock()

	if len(s.qps) == 0 {
		s.sampler = jaeger.NewConstSampler(true)
		s.logger.Debug("no operations for updating sampler")
		return
	}

	now := time.Now()
	operations := make([]model.Operation, len(s.qps))
	for opName, qE := range s.qps {
		if qE.qps == 0 || qE.endTime.Add(s.samplingRefreshInterval).Before(now) {
			qE.qps = float64(qE.throughput) / toSeconds(now, qE.endTime)
			qE.endTime = now
			qE.throughput = 0
		}
		operations = append(operations, model.Operation{
			Service: s.serviceName,
			Name:    opName,
			Qps:     qE.qps,
		})
	}

	res, err := s.samplingFetcher.Fetch(s.serviceName, operations, s.samplingRefreshInterval)
	if err != nil {
		s.metrics.SamplerQueryFailure.Inc(1)
		s.logger.Debug("failed to fetch sampling strategy",
			zap.Error(err),
			zap.String("response", string(res)))
		return
	}
	strategy, err := s.samplingParser.Parse(res)
	if err != nil {
		s.metrics.SamplerUpdateFailure.Inc(1)
		s.logger.Debug("failed to parse sampling strategy response",
			zap.Error(err),
			zap.String("response", string(res)))
		return
	}

	s.metrics.SamplerRetrieved.Inc(1)
	if err := s.updateSamplerViaUpdaters(strategy); err != nil {
		s.metrics.SamplerUpdateFailure.Inc(1)
		s.logger.Debug("failed to handle sampling strategy response", zap.Error(err))
		return
	}

	var newSamplerStr string
	switch s.sampler.(type) {
	case *jaeger.ProbabilisticSampler:
		newSamplerStr = s.sampler.(*jaeger.ProbabilisticSampler).String()
	case *jaeger.RateLimitingSampler:
		newSamplerStr = s.sampler.(*jaeger.RateLimitingSampler).String()
	case *jaeger.ConstSampler:
		newSamplerStr = s.sampler.(*jaeger.ConstSampler).String()
	case *jaeger.PerOperationSampler:
		newSamplerStr = s.sampler.(*jaeger.PerOperationSampler).String()
	default:
		newSamplerStr = "unknown"
	}
	s.logger.Debug("Updated sampler", zap.String("sampler", newSamplerStr))
	s.metrics.SamplerUpdated.Inc(1)
}

func (s *AdaptiveSampler) updateSamplerViaUpdaters(strategy interface{}) error {
	for _, updater := range s.updaters {
		sampler, err := updater.Update(s.sampler, strategy)
		if err != nil {
			return err
		}
		if sampler != nil {
			s.sampler = sampler
			return nil
		}
	}
	return fmt.Errorf("unsupported sampling strategy %+v", strategy)
}

func toSeconds(d1, d2 time.Time) float64 {
	return d1.Sub(d2).Seconds()
}
