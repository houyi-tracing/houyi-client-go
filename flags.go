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
	"flag"
	"github.com/houyi-tracing/houyi/ports"
	"github.com/spf13/viper"
	"time"
)

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

type Options struct {
	SamplerType string

	AlwaysSample       bool
	SamplingRate       float64
	RefreshInterval    time.Duration
	MaxTracesPerSecond float64

	ReporterType string

	AgentAddr             string
	AgentPort             int
	BufferRefreshInterval time.Duration
	MaxBufferedSize       int
	QueueSize             int
}

func AddFlags(flags *flag.FlagSet) {
	addSamplerFlags(flags)
	addReporterFlags(flags)
}

func addSamplerFlags(flags *flag.FlagSet) {
	flags.String(
		samplerType,
		DefaultSamplerType,
		"Sampler type for tracing. [const, probability, rate-limit, adaptive, dynamic]")

	flags.Duration(
		refreshInterval,
		DefaultRefreshInterval,
		"Interval of pulling sampling strategy.")

	flags.Bool(
		alwaysSample,
		DefaultAlwaysSample,
		"Always sample or not (const sampler)")

	flags.Float64(
		samplingRate,
		DefaultSamplingRate,
		"Sampling rate (probability sampler)")

	flags.Float64(
		maxTracesPerSecond,
		DefaultMaxTracesPerSecond,
		"Maximum traces per second (rate-limit sampler)")
}

func addReporterFlags(flags *flag.FlagSet) {
	flags.String(
		reporterType,
		DefaultReporterType,
		"Type of reporter [null, logging, remote]")

	flags.String(
		agentHost,
		DefaultAgentAddr,
		"Agent host for pulling sampling strategies (adaptive/dynamic sampler) and uploading spans (remote reporter)")

	flags.Int(
		agentPort,
		DefaultAgentPort,
		"Agent gRPC port for uploading spans (remote reporter)")

	flags.Duration(
		bufferRefreshInterval,
		DefaultBufferRefreshInterval,
		"Buffer refresh interval for flush spans to agent (remote reporter)")

	flags.Int(
		maxBufferedSize,
		DefaultMaxBufferedSize,
		"Maximum buffered size of spans cache. (remote reporter)")

	flags.Int(
		queueSize,
		DefaultQueueSize,
		"Queue size of channel in reporter to cache spans.")
}

func (opts *Options) InitFromViper(v *viper.Viper) *Options {
	opts.SamplerType = v.GetString(samplerType)
	opts.RefreshInterval = v.GetDuration(refreshInterval)
	opts.AlwaysSample = v.GetBool(alwaysSample)
	opts.SamplingRate = v.GetFloat64(samplingRate)
	opts.MaxTracesPerSecond = v.GetFloat64(maxTracesPerSecond)

	opts.ReporterType = v.GetString(reporterType)
	opts.AgentAddr = v.GetString(agentHost)
	opts.AgentPort = v.GetInt(agentPort)
	opts.BufferRefreshInterval = v.GetDuration(bufferRefreshInterval)
	opts.MaxBufferedSize = v.GetInt(maxBufferedSize)
	opts.QueueSize = v.GetInt(queueSize)
	return opts
}
