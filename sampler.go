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
	"github.com/uber/jaeger-client-go/utils"
	"io"
	"math"
	"math/rand"
	"time"
)

const (
	SamplerTypeKey  = "sampler.type"
	SamplerParamKey = "sampler.param"
)

const (
	SamplerTypeConst        = "const"
	SamplerTypeDynamic      = "dynamic"
	SamplerTypeAdaptive     = "adaptive"
	SamplerTypeProbability  = "probability"
	SamplerTypeRateLimiting = "rate-limiting"
	SamplerTypeUnknown      = "unknown"
)

type SamplingDecision struct {
	Sampled bool
	Tag     []opentracing.Tag
}

type Sampler interface {
	OnCreateSpan(span *Span) SamplingDecision
	IsSampled(span *Span) (bool, []opentracing.Tag)

	io.Closer
}

// -----------------------

// ConstSampler
type ConstSampler struct {
	decision bool
	tags     []opentracing.Tag
}

func NewConstSampler(decision bool) Sampler {
	return &ConstSampler{
		decision: decision,
		tags: []opentracing.Tag{
			{
				Key:   SamplerTypeKey,
				Value: SamplerTypeConst,
			},
			{
				Key:   SamplerParamKey,
				Value: decision,
			},
		},
	}
}

func (s *ConstSampler) OnCreateSpan(_ *Span) SamplingDecision {
	return SamplingDecision{
		Sampled: s.decision,
		Tag:     s.tags,
	}
}

func (s *ConstSampler) IsSampled(_ *Span) (bool, []opentracing.Tag) {
	return s.decision, s.tags
}

func (s *ConstSampler) Close() error {
	return nil
}

func (s *ConstSampler) String() string {
	return fmt.Sprintf("const_sampler(decision=%v,tags=%v)", s.decision, s.tags)
}

// -----------------------

// ProbabilitySampler
type ProbabilitySampler struct {
	samplingRate float64
	tags         []opentracing.Tag
}

func NewProbabilitySampler(samplingRate float64) Sampler {
	return &ProbabilitySampler{
		samplingRate: samplingRate,
		tags: []opentracing.Tag{
			{
				Key:   SamplerTypeKey,
				Value: SamplerTypeConst,
			},
			{
				Key:   SamplerParamKey,
				Value: samplingRate,
			},
		},
	}
}

func (s *ProbabilitySampler) OnCreateSpan(_ *Span) SamplingDecision {
	rand.Seed(time.Now().UnixNano())
	return SamplingDecision{
		Sampled: rand.Float64() < s.samplingRate,
		Tag:     s.tags,
	}
}

func (s *ProbabilitySampler) IsSampled(_ *Span) (bool, []opentracing.Tag) {
	rand.Seed(time.Now().UnixNano())
	return rand.Float64() < s.samplingRate, s.tags
}

func (s *ProbabilitySampler) Close() error {
	return nil
}

// -----------------------

// RateLimitingSampler
type RateLimitingSampler struct {
	maxTracesPerSecond float64
	rateLimiter        *utils.ReconfigurableRateLimiter // used in Jaeger version of rate-limiting sampler
	tags               []opentracing.Tag
}

func NewRateLimitingSampler(maxTracesPerSecond float64) Sampler {
	return &RateLimitingSampler{
		maxTracesPerSecond: maxTracesPerSecond,
		rateLimiter:        utils.NewRateLimiter(maxTracesPerSecond, math.Max(maxTracesPerSecond, 1.0)),
		tags: []opentracing.Tag{
			{
				Key:   SamplerTypeKey,
				Value: SamplerTypeRateLimiting,
			},
			{
				Key:   SamplerParamKey,
				Value: maxTracesPerSecond,
			},
		},
	}
}

func (s *RateLimitingSampler) OnCreateSpan(_ *Span) SamplingDecision {
	return SamplingDecision{
		Sampled: s.rateLimiter.CheckCredit(1.0),
		Tag:     s.tags,
	}
}

func (s *RateLimitingSampler) IsSampled(_ *Span) (bool, []opentracing.Tag) {
	return s.rateLimiter.CheckCredit(1.0), s.tags
}

func (s *RateLimitingSampler) Close() error {
	return nil
}

// -----------------------

// PerOperationSampler
type PerOperationSampler struct {
	operation string
	sampler   Sampler
}

func NewPerOperationSampler(operation string, sampler Sampler) Sampler {
	return &PerOperationSampler{
		operation: operation,
		sampler:   sampler,
	}
}

func (s *PerOperationSampler) OnCreateSpan(span *Span) SamplingDecision {
	return s.sampler.OnCreateSpan(span)
}

func (s *PerOperationSampler) IsSampled(span *Span) (bool, []opentracing.Tag) {
	return s.sampler.IsSampled(span)
}

func (s *PerOperationSampler) Close() error {
	return s.sampler.Close()
}
