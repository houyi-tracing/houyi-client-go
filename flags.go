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
	"github.com/jaegertracing/jaeger/ports"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

const (
	// sampler
	samplerType        = "sampler.type"
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
	reporterType          = "reporter.type"
	bufferRefreshInterval = "buffer.refresh.interval"
	agentHost             = "agent.host"
	agentGRPCPort         = "agent.grpc.port"
	agentHttpPort         = "agent.http.port"
	udpMaxPacketSize      = "udp.max.packet.size"

	DefaultReporterType          = "remote"
	DefaultAgentHost             = "localhost"
	DefaultAgentGRPCPort         = ports.AgentJaegerThriftCompactUDP
	DefaultAgentHttpPort         = ports.AgentConfigServerHTTP
	DefaultBufferRefreshInterval = time.Second * 10
	DefaultUdpMaxPacketSize      = 65000
)

type Options struct {
	AlwaysSample       bool
	MaxTracesPerSecond float64
	RefreshInterval    time.Duration
	SamplerType        string
	SamplingRate       float64
	StrategyURI        string

	AgentHost             string
	AgentGRPCPort         int
	AgentHttpPort         int
	BufferRefreshInterval time.Duration
	ReporterType          string
	UdpMaxPacketSize      int
}

func AddCmdFlags(v *viper.Viper, cmd *cobra.Command, flagSet *flag.FlagSet) error {
	AddFlags(flagSet)
	cmd.Flags().AddGoFlagSet(flagSet)
	return v.BindPFlags(cmd.Flags())
}

func AddFlags(flags *flag.FlagSet) {
	addSamplerFlags(flags)
	addReporterFlags(flags)
}

func addSamplerFlags(flags *flag.FlagSet) {
	flags.String(
		samplerType,
		DefaultSamplerType,
		"Sampler type for tracing [const, probability, rate-limit, adaptive]")

	flags.Duration(
		refreshInterval,
		DefaultRefreshInterval,
		"Refresh interval between requests to pull sampling strategy from jaeger-agent.")

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
		DefaultAgentHost,
		"Agent host for pulling sampling strategies (dynamic sampler) and uploading spans (remote reporter)")

	flags.Int(
		agentGRPCPort,
		DefaultAgentGRPCPort,
		"Agent gRPC port for uploading spans (remote reporter)")

	flags.Duration(
		bufferRefreshInterval,
		DefaultBufferRefreshInterval,
		"Buffer refresh interval for upload spans to agent (remote reporter)")

	flags.Int(
		agentHttpPort,
		DefaultAgentHttpPort,
		"Agent HTTP port for receiving pulling sampling strategies requests (dynamic sampler & remote reporter)")

	flags.Int(
		udpMaxPacketSize,
		DefaultUdpMaxPacketSize,
		"Max size of UDP packet to push spans to agent")
}

func (opts *Options) InitFromViper(v *viper.Viper) *Options {
	opts.SamplerType = v.GetString(samplerType)
	opts.RefreshInterval = v.GetDuration(refreshInterval)
	opts.AlwaysSample = v.GetBool(alwaysSample)
	opts.SamplingRate = v.GetFloat64(samplingRate)
	opts.MaxTracesPerSecond = v.GetFloat64(maxTracesPerSecond)

	opts.AgentHost = v.GetString(agentHost)
	opts.AgentGRPCPort = v.GetInt(agentGRPCPort)
	opts.AgentHttpPort = v.GetInt(agentHttpPort)
	opts.BufferRefreshInterval = v.GetDuration(bufferRefreshInterval)
	opts.ReporterType = v.GetString(reporterType)
	return opts
}
