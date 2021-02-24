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
	"github.com/houyi-tracing/houyi/ports"
	"time"
)

// flags.go
const (
	// sampler
	samplerType = "sampler.type"

	// params of samplers
	alwaysSample       = "always.sample"
	samplingRate       = "sampling.rate"
	refreshInterval    = "refresh.interval"
	maxTracesPerSecond = "max.traces.per.second"

	DefaultSamplerType        = SamplerTypeDynamic
	DefaultAlwaysSample       = true
	DefaultSamplingRate       = 1.0
	DefaultMaxTracesPerSecond = 2000
	DefaultRefreshInterval    = time.Second * 30

	// reporter
	reporterType = "reporter.type"

	// params of reporter
	agentHost             = "agent.host"
	agentPort             = "agent.port"
	maxBufferedSize       = "max.buffered.size"
	bufferRefreshInterval = "buffer.refresh.interval"
	queueSize             = "reporter.queue.size"

	DefaultReporterType          = "remote"
	DefaultAgentAddr             = "localhost"
	DefaultAgentPort             = ports.AgentGrpcListenPort
	DefaultBufferRefreshInterval = time.Second * 1
	DefaultQueueSize             = 100
	DefaultMaxBufferedSize       = 65000
)

// reporter.go
const (
	ReporterType_Null    = "null"
	ReporterType_Logging = "logging"
	ReporterType_Remote  = "remote"
)

// sampler.go
const (
	SamplerTypeKey          = "sampler.type"
	SamplerParamKey         = "sampler.param"
	SamplerTypeConst        = "const"
	SamplerTypeDynamic      = "dynamic"
	SamplerTypeAdaptive     = "adaptive"
	SamplerTypeProbability  = "probability"
	SamplerTypeRateLimiting = "rate-limiting"
)

// tracer.go
const (
	BaggageServiceNameKey   = "svc"
	BaggageOperationNameKey = "op"

	ParentServiceNameTagKey   = "p-svc"
	ParentOperationNameTagKey = "p-op"
)

// propagation.go
const (
	TraceContextHeaderName   = "houyi-tx-ctx"
	TraceBaggageHeaderPrefix = "houyi-"
)
