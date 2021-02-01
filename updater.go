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
	Update(sampler Sampler, strategies interface{}) (Sampler, error)
}

type adaptiveSamplerUpdater struct{}

func NewAdaptiveSamplerUpdater() SamplerUpdater {
	return &adaptiveSamplerUpdater{}
}

func (s *adaptiveSamplerUpdater) Update(sampler Sampler, strategies interface{}) (Sampler, error) {
	toUpdateSampler, ok := sampler.(*RemoteSampler)
	if !ok {
		return sampler, fmt.Errorf("sample must be type of RemoteSampler")
	}

	if resp, ok := strategies.(*api_v1.StrategyResponse); ok {
		if resp.GetStrategyType() != api_v1.StrategyType_ADAPTIVE {
			return sampler, fmt.Errorf("unmatched startegy type")
		}
		if adaptive := resp.GetAdaptive(); adaptive != nil {
			for _, strategy := range adaptive.GetStrategies() {
				op, s := strategy.GetOperation(), strategy.GetStrategy()
				toUpdateSampler.strategies[op] =
					NewPerOperationSampler(op, NewProbabilitySampler(s.GetSamplingRate())).(*PerOperationSampler)
			}
			return toUpdateSampler, nil
		} else {
			return sampler, fmt.Errorf("received nil adpative sampler")
		}
	} else {
		return sampler, fmt.Errorf("invalid type of strateigies")
	}
}

type dynamicSamplerUpdater struct{}

func NewDynamicSamplerUpdater() SamplerUpdater {
	return &dynamicSamplerUpdater{}
}

func (s *dynamicSamplerUpdater) Update(sampler Sampler, strategies interface{}) (Sampler, error) {
	toUpdateSampler, ok := sampler.(*RemoteSampler)
	if !ok {
		return sampler, fmt.Errorf("sample must be type of RemoteSampler")
	}

	if resp, ok := strategies.(*api_v1.StrategyResponse); ok {
		if resp.GetStrategyType() != api_v1.StrategyType_DYNAMIC {
			return sampler, fmt.Errorf("unmatched startegy type")
		}
		if dynamic := resp.GetDynamic(); dynamic != nil {
			for _, strategy := range dynamic.GetStrategies() {
				op := strategy.GetOperation()
				switch strategy.GetStrategyType() {
				case api_v1.StrategyType_CONST:
					toUpdateSampler.strategies[op] = &PerOperationSampler{
						operation: op,
						sampler:   NewConstSampler(strategy.GetConst().Sample),
					}
				case api_v1.StrategyType_PROBABILITY:
					toUpdateSampler.strategies[op] = &PerOperationSampler{
						operation: op,
						sampler:   NewProbabilitySampler(strategy.GetProbability().GetSamplingRate()),
					}
				case api_v1.StrategyType_RATE_LIMITING:
					toUpdateSampler.strategies[op] = &PerOperationSampler{
						operation: op,
						sampler:   NewRateLimitingSampler(float64(strategy.GetRateLimiting().GetMaxTracesPerSecond())),
					}
				}
			}
			return toUpdateSampler, nil
		} else {
			return sampler, fmt.Errorf("received nil dynamic sampler")
		}
	} else {
		return sampler, fmt.Errorf("invalid type of strateigies")
	}
}
