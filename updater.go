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
	"github.com/houyi-tracing/houyi/idl/api_v1"
)

type SamplerUpdater interface {
	Update(sampler *RemoteSampler, strategies *api_v1.StrategiesResponse) error
}

type samplerUpdater struct{}

func NewSamplerUpdater() SamplerUpdater {
	return &samplerUpdater{}
}

func (s *samplerUpdater) Update(sampler *RemoteSampler, resp *api_v1.StrategiesResponse) error {
	for _, strategy := range resp.GetStrategies() {
		var newSampler Sampler
		switch strategy.GetType() {
		case api_v1.Type_CONST:
			newSampler = NewConstSampler(strategy.GetConst().GetAlwaysSample())
		case api_v1.Type_PROBABILITY:
			newSampler = NewProbabilitySampler(strategy.GetProbability().GetSamplingRate())
		case api_v1.Type_RATE_LIMITING:
			newSampler = NewRateLimitingSampler(float64(strategy.GetRateLimiting().GetMaxTracesPerSecond()))
		case api_v1.Type_ADAPTIVE:
			newSampler = NewAdaptiveSampler(strategy.GetAdaptive().GetSamplingRate())
		case api_v1.Type_DYNAMIC:
			newSampler = NewDynamicSampler(strategy.GetDynamic().GetSamplingRate())
		default:
			return fmt.Errorf("invalid sampler type")
		}
		sampler.strategies[strategy.GetOperation()] = &PerOperationSampler{
			strategy.GetOperation(),
			newSampler,
		}

	}
	return nil
}
