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
)

type SamplingDecision struct {
	Sampled bool
	Tag     []opentracing.Tag
}

type Sampler interface {
	OnCreateSpan(span *Span) SamplingDecision
	IsSampled(span *Span) (bool, []opentracing.Tag)

	fmt.Stringer
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

func (s *ConstSampler) IsSampled(span *Span) (bool, []opentracing.Tag) {
	return span.context.IsSampled(), span.Tags()
}

func (s *ConstSampler) Close() error {
	return nil
}

func (s *ConstSampler) String() string {
	return fmt.Sprintf("const_sampler(decision=%v,tags=%v)", s.decision, s.tags)
}

// -----------------------

const samplingRateMask = ^(uint64(1) << 63)

// ProbabilitySampler
type ProbabilitySampler struct {
	samplingRate     float64
	samplingBoundary uint64
	tags             []opentracing.Tag
}

func NewProbabilitySampler(samplingRate float64) Sampler {
	return &ProbabilitySampler{
		samplingRate:     samplingRate,
		samplingBoundary: uint64(samplingRate * float64(math.MaxUint64&samplingRateMask)),
		tags: []opentracing.Tag{
			{
				Key:   SamplerTypeKey,
				Value: SamplerTypeProbability,
			},
			{
				Key:   SamplerParamKey,
				Value: samplingRate,
			},
		},
	}
}

func (s *ProbabilitySampler) OnCreateSpan(span *Span) SamplingDecision {
	// spanID was generated randomly so that we can use it to make probability sampling.
	return SamplingDecision{
		Sampled: uint64(span.context.spanID)&samplingRateMask <= s.samplingBoundary,
		Tag:     s.tags,
	}
}

func (s *ProbabilitySampler) IsSampled(span *Span) (bool, []opentracing.Tag) {
	return span.context.IsSampled(), span.Tags()
}

func (s *ProbabilitySampler) String() string {
	return fmt.Sprintf("probability_sampler(sampling_rate=%f)", s.samplingRate)
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

func (s *RateLimitingSampler) IsSampled(span *Span) (bool, []opentracing.Tag) {
	return span.context.IsSampled(), span.Tags()
}

func (s *RateLimitingSampler) String() string {
	return fmt.Sprintf("rate_limiting_sampler(max_traces_per_second=%f)", s.maxTracesPerSecond)
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

func (s *PerOperationSampler) String() string {
	return s.sampler.String()
}

func (s *PerOperationSampler) Close() error {
	return s.sampler.Close()
}
