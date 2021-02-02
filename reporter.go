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
	"go.uber.org/zap"
	"io"
	"sync"
	"time"
)

type Reporter interface {
	Report(span *Span)
	io.Closer
}

///////////////////////////////////////////////////////////////////////////////
// RemoteReporter
///////////////////////////////////////////////////////////////////////////////

type remoteReporter struct {
	logger *zap.Logger

	queueSize     int
	flushInterval time.Duration

	transport Transport

	spanCh chan *Span
	stopCh chan *sync.WaitGroup
}

func NewRemoteReporter(logger *zap.Logger, queueSize int, interval time.Duration, transport Transport) Reporter {
	r := &remoteReporter{
		logger:        logger,
		queueSize:     queueSize,
		flushInterval: interval,
		transport:     transport,
		spanCh:        make(chan *Span, queueSize+100),
		stopCh:        make(chan *sync.WaitGroup),
	}
	go r.processQueue()
	return r
}

func (r *remoteReporter) Report(span *Span) {
	select {
	case r.spanCh <- span:
		// TODO implement metrics
	default:
		// do nothing
	}
}

func (r *remoteReporter) Close() error {
	var wg sync.WaitGroup
	wg.Add(1)
	r.stopCh <- &wg
	wg.Wait()
	return nil
}

func (r *remoteReporter) processQueue() {
	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()

	flush := func() {
		if flushed, err := r.transport.Flush(); err != nil {
			r.logger.Debug("Failed to flush buffer", zap.Error(err))
		} else if flushed > 0 {
			r.logger.Debug("Flushed buffer", zap.Int("flushed", flushed))
		}
	}

	for {
		select {
		case <-ticker.C:
			flush()
		case span := <-r.spanCh:
			if flushed, err := r.transport.Append(span); err != nil {
				r.logger.Error("Failed to append span to transport", zap.Error(err))
			} else if flushed > 0 {
				r.logger.Debug("Flushed transport because of exceeding maximum buffered size",
					zap.Int("flushed", flushed))
			}
		case wg := <-r.stopCh:
			flush()
			wg.Done()
			return
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
// Log Reporter
///////////////////////////////////////////////////////////////////////////////

type logReporter struct {
	logger *zap.Logger
}

func NewLogReporter(logger *zap.Logger) Reporter {
	return &logReporter{logger: logger}
}

func (r *logReporter) Report(span *Span) {
	sc := span.context
	r.logger.Debug("report span",
		zap.Stringer("trace ID", sc.traceID),
		zap.Stringer("span ID", sc.spanID),
		zap.Stringer("parent span ID", sc.parentID),
		zap.String("operation name", span.operationName),
		zap.Time("start time", span.startTime),
		zap.Duration("duration", span.duration),
		zap.Bool("is ingress", span.isIngress),
		zap.Any("tags", span.tags),
		zap.Any("logs", span.logs),
		zap.Any("references", span.ref))
}

func (r *logReporter) Close() error {
	// do nothing
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Null Reporter
///////////////////////////////////////////////////////////////////////////////

type nullReporter struct{}

func NewNullReporter() Reporter {
	return &nullReporter{}
}

func (r *nullReporter) Report(_ *Span) {
	// do nothing
}

func (r *nullReporter) Close() error {
	return nil
}
