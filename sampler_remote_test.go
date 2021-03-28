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
	"github.com/houyi-tracing/houyi/pkg/routing"
	"github.com/houyi-tracing/houyi/ports"
	"go.uber.org/zap"
	"math/rand"
	"os"
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
		AgentEndpoint: routing.Endpoint{
			Addr: "localhost",
			Port: ports.AgentGrpcListenPort,
		},
	})
	reporter := NewLogReporter(logger)

	tracer := NewTracer(serviceName, &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	tracer.StartSpan("op")

	time.Sleep(time.Hour)
}

func TestWriteToLocal(t *testing.T) {
	fileName := "sampling_rate.record"
	for i := 0; i < 100; i++ {
		if f, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND, 0755); err == nil {
			_, _ = f.WriteString(fmt.Sprintf("%s\t%f\n", time.Now().Format(time.RFC3339), rand.Float64()))
			_ = f.Close()
		}
	}
}
