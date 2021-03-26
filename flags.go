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
	"github.com/spf13/viper"
	"time"
)

type Options struct {
	PullStrategiesInterval time.Duration
	ReporterType           string
	AgentAddr              string
	AgentPort              int
	BufferRefreshInterval  time.Duration
	MaxBufferedSize        int
	QueueSize              int
}

func AddFlags(flags *flag.FlagSet) {
	flags.Duration(
		pullStrategiesInterval,
		DefaultPullStrategiesInterval,
		"[Houyi Tracing] Interval for pulling strategies.")

	flags.String(
		reporterType,
		DefaultReporterType,
		"[Houyi Tracing] Type of reporter [null, logging, remote]")

	flags.String(
		agentHost,
		DefaultAgentAddr,
		"[Houyi Tracing] Agent host for pulling sampling strategies (adaptive/dynamic sampler) and uploading spans (remote reporter)")

	flags.Int(
		agentPort,
		DefaultAgentPort,
		"[Houyi Tracing] Agent gRPC port for uploading spans (remote reporter)")

	flags.Duration(
		bufferRefreshInterval,
		DefaultBufferRefreshInterval,
		"[Houyi Tracing] Buffer refresh interval for flush spans to agent (remote reporter)")

	flags.Int(
		maxBufferedSize,
		DefaultMaxBufferedSize,
		"[Houyi Tracing] Maximum buffered size of spans cache. (remote reporter)")

	flags.Int(
		queueSize,
		DefaultQueueSize,
		"[Houyi Tracing] Queue size of channel in reporter to cache spans.")
}

func (opts *Options) InitFromViper(v *viper.Viper) *Options {
	opts.PullStrategiesInterval = v.GetDuration(pullStrategiesInterval)
	opts.ReporterType = v.GetString(reporterType)
	opts.AgentAddr = v.GetString(agentHost)
	opts.AgentPort = v.GetInt(agentPort)
	opts.BufferRefreshInterval = v.GetDuration(bufferRefreshInterval)
	opts.MaxBufferedSize = v.GetInt(maxBufferedSize)
	opts.QueueSize = v.GetInt(queueSize)
	return opts
}
