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
	"bytes"
	"encoding/binary"
	"github.com/opentracing/opentracing-go"
	"io"
	"net/url"
	"strings"
	"sync"
)

// Injector and Extractor are APIs defined in opentracing-go,
// see in [https://github.com/opentracing/opentracing-go/blob/master/mocktracer/propagation.go].

// Injector is responsible for injecting SpanContext instances in a manner suitable
// for propagation via a format-specific "carrier" object. Typically the
// injection will take place across an RPC boundary, but message queues and
// other IPC mechanisms are also reasonable places to use an Injector.
type Injector interface {
	// Inject takes `SpanContext` and injects it into `carrier`. The actual type
	// of `carrier` depends on the `format` passed to `Tracer.Inject()`.
	//
	// Implementations may return opentracing.ErrInvalidCarrier or any other
	// implementation-specific error if injection fails.
	Inject(ctx SpanContext, carrier interface{}) error
}

// Extractor is responsible for extracting SpanContext instances from a
// format-specific "carrier" object. Typically the extraction will take place
// on the server side of an RPC boundary, but message queues and other IPC
// mechanisms are also reasonable places to use an Extractor.
type Extractor interface {
	// Extract decodes a SpanContext instance from the given `carrier`,
	// or (nil, opentracing.ErrSpanContextNotFound) if no context could
	// be found in the `carrier`.
	Extract(carrier interface{}) (SpanContext, error)
}

const (
	TraceContextHeaderName   = "houyi-ctx"
	TraceBaggageHeaderPrefix = "houyi-"
)

type HttpHeadersPropagator struct{}

func (p *HttpHeadersPropagator) Inject(ctx SpanContext, carrier interface{}) error {
	writer, ok := carrier.(opentracing.TextMapWriter)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	writer.Set(TraceContextHeaderName, ctx.String())

	for baggageKey, baggageVal := range ctx.baggage {
		safeVal := url.QueryEscape(baggageVal)
		writer.Set(TraceBaggageHeaderPrefix+baggageKey, safeVal)
	}
	return nil
}

func (p *HttpHeadersPropagator) Extract(carrier interface{}) (SpanContext, error) {
	reader, ok := carrier.(opentracing.TextMapReader)
	if !ok {
		return emptyContext, opentracing.ErrInvalidCarrier
	}

	var retCtx SpanContext

	err := reader.ForeachKey(func(key, val string) error {
		lowerKer := strings.ToLower(key)
		switch {
		case lowerKer == TraceContextHeaderName:
			var err error
			if retCtx, err = ContextFromString(val); err != nil {
				return err
			}
		case strings.HasPrefix(lowerKer, TraceBaggageHeaderPrefix):
			if retCtx.baggage == nil {
				retCtx.baggage = make(map[string]string)
			}
			retCtx.baggage[lowerKer] = url.QueryEscape(val)
		}
		return nil
	})

	if err != nil {
		return SpanContext{}, err
	}

	if retCtx.traceID.Low == 0 || retCtx.traceID.High == 0 || retCtx.spanID == 0 {
		return emptyContext, opentracing.ErrSpanContextNotFound
	}
	return retCtx, nil
}

type BinaryPropagator struct {
	buffers sync.Pool
}

func (p *BinaryPropagator) Inject(ctx SpanContext, carrier interface{}) error {
	writer, ok := carrier.(io.Writer)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	// Handle the houyiTracer context
	if err := binary.Write(writer, binary.BigEndian, ctx.traceID.High); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, ctx.traceID.Low); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, ctx.spanID); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, ctx.parentID); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, ctx.flags); err != nil {
		return err
	}

	// Handle the baggage items
	if err := binary.Write(writer, binary.BigEndian, int32(len(ctx.baggage))); err != nil {
		return err
	}
	for k, v := range ctx.baggage {
		if err := binary.Write(writer, binary.BigEndian, int32(len(k))); err != nil {
			return err
		}
		io.WriteString(writer, k)
		if err := binary.Write(writer, binary.BigEndian, int32(len(v))); err != nil {
			return err
		}
		io.WriteString(writer, v)
	}
	return nil
}

func (p *BinaryPropagator) Extract(carrier interface{}) (SpanContext, error) {
	reader, ok := carrier.(io.Reader)
	if !ok {
		return emptyContext, opentracing.ErrInvalidCarrier
	}

	var ctx SpanContext
	if err := binary.Read(reader, binary.BigEndian, &ctx.traceID.High); err != nil {
		return emptyContext, opentracing.ErrSpanContextCorrupted
	}
	if err := binary.Read(reader, binary.BigEndian, &ctx.traceID.Low); err != nil {
		return emptyContext, opentracing.ErrSpanContextCorrupted
	}
	if err := binary.Read(reader, binary.BigEndian, &ctx.spanID); err != nil {
		return emptyContext, opentracing.ErrSpanContextCorrupted
	}
	if err := binary.Read(reader, binary.BigEndian, &ctx.parentID); err != nil {
		return emptyContext, opentracing.ErrSpanContextCorrupted
	}
	if err := binary.Read(reader, binary.BigEndian, &ctx.flags); err != nil {
		return emptyContext, opentracing.ErrSpanContextCorrupted
	}

	var nBaggage int32
	if err := binary.Read(reader, binary.BigEndian, &nBaggage); err != nil {
		return emptyContext, opentracing.ErrSpanContextCorrupted
	}
	if inBaggage := int(nBaggage); inBaggage > 0 {
		ctx.baggage = make(map[string]string, inBaggage)
		buf := p.buffers.Get().(*bytes.Buffer)
		defer p.buffers.Put(buf)

		var keyLen, valLen int32
		for i := 0; i < inBaggage; i++ {
			if err := binary.Read(reader, binary.BigEndian, &keyLen); err != nil {
				return emptyContext, opentracing.ErrSpanContextCorrupted
			}
			buf.Reset()
			buf.Grow(int(keyLen))
			if n, err := io.CopyN(buf, reader, int64(keyLen)); err != nil || int32(n) != keyLen {
				return emptyContext, opentracing.ErrSpanContextCorrupted
			}
			key := buf.String()

			if err := binary.Read(reader, binary.BigEndian, &valLen); err != nil {
				return emptyContext, opentracing.ErrSpanContextCorrupted
			}
			buf.Reset()
			buf.Grow(int(valLen))
			if n, err := io.CopyN(buf, reader, int64(valLen)); err != nil || int32(n) != valLen {
				return emptyContext, opentracing.ErrSpanContextCorrupted
			}
			ctx.baggage[key] = buf.String()
		}
	}

	return ctx, nil
}
