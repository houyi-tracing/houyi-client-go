// Copyright (c) 2021 The Houyi Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
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
	"strconv"
	"strings"
)

///////////////////////////////////////////////////////////////////////////////
// Trace ID
///////////////////////////////////////////////////////////////////////////////

type TraceID struct {
	Low  uint64
	High uint64
}

func (t TraceID) String() string {
	return fmt.Sprintf("%016x:%016x", t.High, t.Low)
}

///////////////////////////////////////////////////////////////////////////////
// Span ID
///////////////////////////////////////////////////////////////////////////////

type SpanID uint64

func (s SpanID) String() string {
	return fmt.Sprintf("%016x", uint64(s))
}

///////////////////////////////////////////////////////////////////////////////
// Span Context
///////////////////////////////////////////////////////////////////////////////

type SpanContext struct {
	traceID  TraceID
	spanID   SpanID
	parentID SpanID

	// flags is used to represent the state of span context.
	//
	// The lowest bit of flags is used to represent whether this span is sampled: 1 for sampled, 0 for not sampled.
	//
	// Other bits of flags would be used for other purposes in the future.
	flags   byte
	baggage map[string]string
}

var emptyContext SpanContext

func NewSpanContext(
	traceID TraceID,
	spanID SpanID,
	parentSpanID SpanID,
	sampled bool,
	baggage map[string]string) SpanContext {
	var flags byte
	if sampled {
		flags = 1 // is sampled
	} else {
		flags = 0
	}
	return SpanContext{
		traceID:  traceID,
		spanID:   spanID,
		parentID: parentSpanID,
		flags:    flags,
		baggage:  baggage,
	}
}

func (s SpanContext) TraceID() TraceID {
	return s.traceID
}

func (s SpanContext) SpanID() SpanID {
	return s.spanID
}

func (s SpanContext) ParentID() SpanID {
	return s.parentID
}

func (s SpanContext) IsSampled() bool {
	return s.flags&1 == 1
}

func (s SpanContext) String() string {
	return fmt.Sprintf("%016x:%016x:%016x:%016x:%o",
		s.traceID.High, s.traceID.Low, uint64(s.spanID), uint64(s.parentID), s.flags)
}

func (s SpanContext) ForeachBaggageItem(handler func(k string, v string) bool) {
	for k, v := range s.baggage {
		if !handler(k, v) {
			break
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
// Functions
///////////////////////////////////////////////////////////////////////////////

var (
	ErrEmptyContextString   = fmt.Errorf("context string can not be empty")
	ErrInvalidFormatContext = fmt.Errorf("split parts of context string should be length 5")
)

// ContextFromString reconstructs the Context encoded in a string
func ContextFromString(context *SpanContext, value string) error {
	if value == "" {
		return ErrEmptyContextString
	}

	parts := strings.Split(value, ":")
	if len(parts) != 5 {
		return ErrInvalidFormatContext
	}

	var err error
	if context.traceID.High, err = HexToUint64(parts[0]); err != nil {
		return err
	}
	if context.traceID.Low, err = HexToUint64(parts[1]); err != nil {
		return err
	}

	if sID, err := HexToUint64(parts[2]); err != nil {
		return err
	} else {
		context.spanID = SpanID(sID)
	}
	if psID, err := HexToUint64(parts[3]); err != nil {
		return err
	} else {
		context.parentID = SpanID(psID)
	}

	flags, err := strconv.ParseUint(parts[4], 8, 8)
	if err != nil {
		return err
	} else {
		context.flags = byte(flags)
	}

	return nil
}

// HexToUint64 converts a hexadecimal string to it's actual value of type uint64.
func HexToUint64(value string) (uint64, error) {
	if len(value) > 16 {
		return 0, fmt.Errorf("value string can not be longer than 16 hex characters: %s", value)
	}
	ui, err := strconv.ParseUint(value, 16, 64)
	if err != nil {
		return 0, err
	}
	return ui, nil
}
