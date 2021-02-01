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
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/utils"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type Tracer interface {
	opentracing.Tracer
	io.Closer

	Sampler() Sampler
	Tags() []opentracing.Tag
}

type TracerParams struct {
	Logger   *zap.Logger
	Reporter Reporter
	Sampler  Sampler
}

type tracer struct {
	logger *zap.Logger

	serviceName string
	reporter    Reporter
	sampler     Sampler
	process     Process

	randomNumber func() uint64

	injectors  map[interface{}]Injector
	extractors map[interface{}]Extractor

	tags []opentracing.Tag
}

func NewTracer(serviceName string, params *TracerParams) Tracer {
	t := &tracer{
		serviceName: serviceName,
		logger:      params.Logger,
		reporter:    params.Reporter,
		sampler:     params.Sampler,
		injectors:   make(map[interface{}]Injector),
		extractors:  make(map[interface{}]Extractor),
	}

	bp := &BinaryPropagator{
		buffers: sync.Pool{},
	}
	t.injectors[opentracing.Binary] = bp
	t.extractors[opentracing.Binary] = bp

	hdp := &HttpHeadersPropagator{}
	t.injectors[opentracing.HTTPHeaders] = hdp
	t.extractors[opentracing.HTTPHeaders] = hdp

	t.tags = make([]opentracing.Tag, 0)
	t.process = Process{
		Service: serviceName,
		UUID:    strconv.FormatUint(t.randomNumber(), 16),
		Tags:    t.tags,
	}

	seedGenerator := utils.NewRand(time.Now().UnixNano())
	pool := sync.Pool{
		New: func() interface{} {
			return rand.NewSource(seedGenerator.Int63())
		},
	}
	t.randomNumber = func() uint64 {
		generator := pool.Get().(rand.Source)
		number := uint64(generator.Int63())
		pool.Put(generator)
		return number
	}
	return t
}

func (t *tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	sso := opentracing.StartSpanOptions{}
	for _, o := range opts {
		o.Apply(&sso)
	}
	return t.startSpanWithOptions(operationName, sso)
}

// startSpanWithOptions is similar with the Jaeger version
func (t *tracer) startSpanWithOptions(operationName string, options opentracing.StartSpanOptions) opentracing.Span {
	if options.StartTime.IsZero() {
		options.StartTime = time.Now()
	}

	var references []opentracing.SpanReference
	var parentCtx SpanContext
	var ctx SpanContext
	var hasParent bool

	for _, ref := range options.References {
		ctxRef, ok := ref.ReferencedContext.(SpanContext)
		if !ok {
			t.logger.Error("ReferencedContext is not type of SpanContext")
			continue
		}

		references = append(references, opentracing.SpanReference{
			Type:              ref.Type,
			ReferencedContext: ctxRef,
		})

		if !hasParent && (ref.Type == opentracing.ChildOfRef || ref.Type == opentracing.FollowsFromRef) {
			hasParent = true
			parentCtx = ctxRef
		}
	}

	if !hasParent {
		ctx.traceID.Low = t.randomNumber()
		ctx.traceID.High = t.randomNumber()
		ctx.spanID = SpanID(ctx.traceID.Low)
		ctx.parentID = 0
	} else {
		ctx.traceID = parentCtx.traceID
		ctx.spanID = SpanID(t.randomNumber())
		ctx.parentID = parentCtx.spanID
	}

	sp := &Span{}
	sp.context = ctx
	sp.tracer = t
	sp.operationName = operationName
	sp.startTime = options.StartTime
	sp.duration = 0
	sp.ref = references
	sp.isIngress = sp.context.parentID == 0
	sp.tags = make([]opentracing.Tag, 0)

	for key, tag := range options.Tags {
		sp.SetTag(key, tag)
	}

	// make sampling decision
	decision := t.sampler.OnCreateSpan(sp)
	sp.tags = append(sp.tags, decision.Tag...)

	if decision.Sampled {
		sp.context.flags = 1
	} else {
		sp.context.flags = 0
	}

	return sp
}

func (t *tracer) Inject(ctx opentracing.SpanContext, format interface{}, carrier interface{}) error {
	if injector, ok := t.injectors[format]; ok {
		return injector.Inject(ctx.(SpanContext), carrier)
	} else {
		return opentracing.ErrUnsupportedFormat
	}
}

func (t *tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	if extractor, ok := t.extractors[format]; ok {
		return extractor.Extract(carrier)
	} else {
		return emptyContext, opentracing.ErrUnsupportedFormat
	}
}

func (t *tracer) Close() error {
	if err := t.reporter.Close(); err != nil {
		return err
	}
	if err := t.sampler.Close(); err != nil {
		return err
	}
	return nil
}

func (t *tracer) Sampler() Sampler {
	return t.sampler
}

func (t *tracer) Tags() []opentracing.Tag {
	return t.tags
}

func (t *tracer) reportSpan(span *Span) {
	t.reporter.Report(span)
}
