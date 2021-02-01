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
	"github.com/opentracing/opentracing-go/log"
	"time"
)

type Span struct {
	context SpanContext

	operationName string

	// isIngress would be true if this span it the root of whole trace, else false.
	isIngress bool

	startTime time.Time

	duration time.Duration

	ref []opentracing.SpanReference

	tags []opentracing.Tag

	logs []opentracing.LogRecord

	tracer *tracer
}

func (s *Span) SpanContext() SpanContext {
	return s.context
}

func (s *Span) OperationName() string {
	return s.operationName
}

func (s *Span) IsIngress() bool {
	return s.isIngress
}

func (s *Span) StartTime() time.Time {
	return s.startTime
}

func (s *Span) Duration() time.Duration {
	return s.duration
}

func (s *Span) References() []opentracing.SpanReference {
	return s.ref
}

func (s *Span) Tags() []opentracing.Tag {
	return s.tags
}

func (s *Span) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s *Span) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

func (s *Span) Logs() []opentracing.LogRecord {
	return s.logs
}

func (s *Span) FinishWithOptions(options opentracing.FinishOptions) {
	if options.FinishTime.IsZero() {
		options.FinishTime = time.Now()
	}
	s.duration = options.FinishTime.Sub(s.startTime)
	if s.context.IsSampled() {
		if options.LogRecords != nil {
			s.logs = append(s.logs, options.LogRecords...)
		}
		for _, ld := range options.BulkLogData {
			s.logs = append(s.logs, ld.ToLogRecord())
		}
		s.tracer.reportSpan(s)
	}
}

func (s *Span) Context() opentracing.SpanContext {
	return s.context
}

func (s *Span) SetOperationName(operationName string) opentracing.Span {
	s.operationName = operationName
	return s
}

func (s *Span) SetTag(key string, value interface{}) opentracing.Span {
	s.tags = append(s.tags, opentracing.Tag{
		Key:   key,
		Value: value,
	})
	return s
}

func (s *Span) LogFields(fields ...log.Field) {
	s.logs = append(s.logs, opentracing.LogRecord{
		Timestamp: time.Now(),
		Fields:    fields,
	})
}

func (s *Span) LogKV(alternatingKeyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(alternatingKeyValues...)
	if err != nil {
		s.LogFields(log.Error(err), log.String("function", "LogKV"))
	}
	s.LogFields(fields...)
}

func (s *Span) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	s.context.baggage[restrictedKey] = value
	return s
}

func (s *Span) BaggageItem(restrictedKey string) string {
	if val, ok := s.context.baggage[restrictedKey]; ok {
		return val
	} else {
		return ""
	}
}

func (s *Span) LogEvent(event string) {
	s.Log(opentracing.LogData{Event: event, Timestamp: time.Now()})
}

func (s *Span) LogEventWithPayload(event string, payload interface{}) {
	s.Log(opentracing.LogData{Event: event, Timestamp: time.Now(), Payload: payload})
}

func (s *Span) Log(data opentracing.LogData) {
	if s.context.IsSampled() {
		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}
		s.logs = append(s.logs, data.ToLogRecord())
	}
}
