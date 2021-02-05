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
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"math"
	"testing"
	"time"
)

func TestProbabilitySampler(t *testing.T) {
	sr := 0.3
	err := 0.01

	logger, _ := zap.NewDevelopment()
	reporter := NewNullReporter()
	sampler := NewProbabilitySampler(sr)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	times := 50000000
	cnt := 0
	for i := 0; i < times; i++ {
		span := tracer.StartSpan("op").(*Span)
		if span.context.IsSampled() {
			cnt += 1
		}
	}
	fmt.Println("Actual sampling rate:", float64(cnt)/float64(times))
	assert.Less(t, math.Abs(float64(cnt)/float64(times)-sr), err)
}

func TestConstSampler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	reporter := NewNullReporter()
	sampler := NewConstSampler(true)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	times := 50000000
	cnt := 0
	for i := 0; i < times; i++ {
		span := tracer.StartSpan("op").(*Span)
		if span.context.IsSampled() {
			cnt += 1
		}
	}
	assert.Equal(t, cnt, times)
}

func TestRateLimitingSampler(t *testing.T) {
	maxTracesPerSecond := 1000.0

	logger, _ := zap.NewDevelopment()
	reporter := NewNullReporter()
	sampler := NewRateLimitingSampler(maxTracesPerSecond)
	tracer := NewTracer("svc", &TracerParams{
		Logger:   logger,
		Reporter: reporter,
		Sampler:  sampler,
	})

	times := 50000000
	cnt := 0

	startTime := time.Now()
	for i := 0; i < times; i++ {
		span := tracer.StartSpan("op").(*Span)
		if span.context.IsSampled() {
			cnt += 1
		}
	}
	endTime := time.Now()

	seconds := endTime.Sub(startTime).Seconds()
	fmt.Println("Total: ", times)
	fmt.Println("Sampled: ", cnt)
	fmt.Println("Seconds: ", seconds)
	fmt.Println("Expected Trace Per Second: ", maxTracesPerSecond)
	fmt.Println("Actual Trace Per Second: ", float64(cnt)/seconds)
}
