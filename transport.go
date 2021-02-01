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
	"context"
	"github.com/houyi-tracing/houyi/pkg/routing"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"io"
	"sync"
)

type Transport interface {
	Append(span *Span) (int, error)
	Flush() (int, error)
	io.Closer
}

type grpcSender struct {
	sync.Mutex

	logger        *zap.Logger
	maxBufferSize int
	buffer        []*model.Span
	agentEp       routing.Endpoint
	process       *Process

	conn *grpc.ClientConn
	c    api_v2.CollectorServiceClient
}

func NewTransport(logger *zap.Logger, maxBufferSize int, agentEp routing.Endpoint, process *Process) Transport {
	sender := &grpcSender{
		logger:        logger,
		maxBufferSize: maxBufferSize,
		buffer:        make([]*model.Span, maxBufferSize),
		agentEp:       agentEp,
		process:       process,
	}

	conn, err := grpc.Dial(sender.agentEp.String(), grpc.WithInsecure())
	if err != nil {
		logger.Fatal("Failed to dail to gRPC", zap.String("endpoint", agentEp.String()))
		return nil
	} else {
		defer conn.Close()
		sender.conn = conn
	}
	sender.c = api_v2.NewCollectorServiceClient(conn)
	return sender
}

func (g *grpcSender) Append(span *Span) (int, error) {
	g.Lock()
	defer g.Unlock()

	jSpan := convertSpan(span, g.process)
	if len(g.buffer) == g.maxBufferSize {
		g.buffer = append(g.buffer, jSpan)
		return g.flush()
	} else {
		g.buffer = append(g.buffer, jSpan)
		return 0, nil
	}
}

func (g *grpcSender) Flush() (int, error) {
	g.Lock()
	defer g.Unlock()

	if g.conn.GetState() == connectivity.Shutdown {
		conn, err := grpc.Dial(g.agentEp.String(), grpc.WithInsecure())
		if err != nil {
			g.logger.Fatal("Connection is closed and transport failed to dail to agent", zap.Error(err))
			return 0, err
		}
		g.conn = conn
		g.c = api_v2.NewCollectorServiceClient(conn)
	}
	return g.flush()
}

func (g *grpcSender) flush() (int, error) {
	jProcess := convertProcess(g.process)
	req := &api_v2.PostSpansRequest{
		Batch: model.Batch{
			Spans:   g.buffer,
			Process: jProcess,
		},
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	if _, err := g.c.PostSpans(ctx, req); err != nil {
		return 0, err
	} else {
		flushed := len(g.buffer)
		return flushed, nil
	}
}

func (g *grpcSender) Close() error {
	g.conn.Close()
	return nil
}

func convertSpan(span *Span, process *Process) *model.Span {
	spanCtx := span.SpanContext()
	traceID := spanCtx.TraceID()
	spanID := spanCtx.SpanID()
	return &model.Span{
		TraceID: model.TraceID{
			Low:  traceID.Low,
			High: traceID.High,
		},
		SpanID:        model.SpanID(spanID),
		OperationName: span.OperationName(),
		References:    convertReferences(span.References()),
		StartTime:     span.StartTime(),
		Duration:      span.Duration(),
		Tags:          convertTags(span.Tags()),
		Logs:          convertLogs(span.Logs()),
		Process:       convertProcess(process),
	}
}

func convertProcess(process *Process) *model.Process {
	tags := make([]model.KeyValue, 0)
	for _, t := range process.Tags {
		tags = append(tags, convertTag(&t))
	}
	return &model.Process{
		ServiceName: process.Service,
		Tags:        tags,
	}
}

func convertTags(tags []opentracing.Tag) []model.KeyValue {
	kv := make([]model.KeyValue, 0, len(tags))
	for _, t := range tags {
		kv = append(kv, convertTag(&t))
	}
	return kv
}

func convertTag(tag *opentracing.Tag) model.KeyValue {
	return toKeyValue(tag.Key, tag.Value)
}

func convertReferences(rel []opentracing.SpanReference) []model.SpanRef {
	ref := make([]model.SpanRef, 0, len(rel))
	for _, r := range rel {
		if ctx, ok := r.ReferencedContext.(SpanContext); ok {
			tID := ctx.TraceID()
			spanRel := model.SpanRef{
				TraceID: model.TraceID{
					Low:  tID.Low,
					High: tID.High,
				},
				SpanID:  model.SpanID(ctx.SpanID()),
				RefType: model.SpanRefType(r.Type),
			}
			ref = append(ref, spanRel)
		}
	}
	return ref
}

func convertLogs(logs []opentracing.LogRecord) []model.Log {
	ret := make([]model.Log, 0, len(logs))
	for _, lr := range logs {
		ret = append(ret, convertLog(lr))
	}
	return ret
}

func convertLog(lr opentracing.LogRecord) model.Log {
	logs := make([]model.KeyValue, 0, len(lr.Fields))
	for _, f := range lr.Fields {
		logs = append(logs, toKeyValue(f.Key(), f.Value()))
	}
	return model.Log{
		Timestamp: lr.Timestamp,
		Fields:    logs,
	}
}

func toKeyValue(key string, val interface{}) model.KeyValue {
	switch val.(type) {
	case string:
		return model.KeyValue{
			Key:   key,
			VType: model.ValueType_STRING,
			VStr:  val.(string),
		}
	case int64, int32, int16, int8, int, uint, uint8, uint16, uint32, uint64:
		return model.KeyValue{
			Key:    key,
			VType:  model.ValueType_INT64,
			VInt64: toInt64(val),
		}
	case bool:
		return model.KeyValue{
			Key:   key,
			VType: model.ValueType_BOOL,
			VBool: val.(bool),
		}
	case float64, float32:
		return model.KeyValue{
			Key:      key,
			VType:    model.ValueType_FLOAT64,
			VFloat64: toFloat64(val),
		}
	}
	return model.KeyValue{}
}

func toInt64(v interface{}) int64 {
	switch v.(type) {
	case int64:
		return v.(int64)
	case int32:
		return int64(v.(int32))
	case int16:
		return int64(v.(int16))
	case int8:
		return int64(v.(int8))
	case int:
		return int64(v.(int))
	case uint64:
		return int64(v.(uint64))
	case uint32:
		return int64(v.(uint32))
	case uint16:
		return int64(v.(uint16))
	case uint8:
		return int64(v.(uint8))
	case uint:
		return int64(v.(uint))
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch v.(type) {
	case float64:
		return v.(float64)
	case float32:
		return float64(v.(float32))
	}
	return 0.0
}
