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
	"github.com/houyi-tracing/houyi/pkg/routing"
	"go.uber.org/zap"
	"testing"
	"time"
)

const serviceName = "houyi-debug"

func TestPullStrategies(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sampler := NewRemoteSampler(&RemoteSamplerParams{
		Logger:       logger,
		ServiceName:  serviceName,
		PullInterval: time.Second * 3,
		Type:         RemoteSampler_Adaptive,
		AgentEndpoint: routing.Endpoint{
			Addr: "192.168.31.77",
			Port: 18760,
		},
	})
	reporter := NewNullReporter()
	tracer := NewTracer(serviceName, &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	go func() {
		for {
			for i := 0; i < 200; i++ {
				s := tracer.StartSpan("op1")
				s.Finish()
			}
			time.Sleep(time.Second)
		}
	}()

	go func() {
		for {
			for i := 0; i < 200; i++ {
				s := tracer.StartSpan("op2")
				s.Finish()
			}
			time.Sleep(time.Second)
		}
	}()

	go func() {
		for {
			for i := 0; i < 200; i++ {
				s := tracer.StartSpan("op3")
				s.Finish()
			}
			time.Sleep(time.Second)
		}
	}()

	time.Sleep(time.Minute)
	_ = sampler.Close()
}

func TestReportSpans(t *testing.T) {

}
