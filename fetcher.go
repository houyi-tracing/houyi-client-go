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
	"context"
	"github.com/houyi-tracing/houyi/idl/api_v1"
	"github.com/houyi-tracing/houyi/pkg/routing"
	"google.golang.org/grpc"
)

type SamplingStrategyFetcher interface {
	Fetch(service string, operations []Operation) (*api_v1.StrategiesResponse, error)
}

type samplingStrategyFetcher struct {
	agentEp routing.Endpoint
}

func NewSamplingStrategyFetcher(agentEp routing.Endpoint) SamplingStrategyFetcher {
	return &samplingStrategyFetcher{agentEp: agentEp}
}

func (f *samplingStrategyFetcher) Fetch(s string, operations []Operation) (*api_v1.StrategiesResponse, error) {
	conn, err := grpc.Dial(f.agentEp.String(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	} else {
		defer conn.Close()
	}

	c := api_v1.NewStrategyManagerClient(conn)
	req := &api_v1.StrategyRequest{
		Service:    s,
		Operations: convertOperations(operations),
	}
	return c.GetStrategies(context.TODO(), req)
}

func convertOperations(ops []Operation) []*api_v1.StrategyRequest_Operation {
	ret := make([]*api_v1.StrategyRequest_Operation, 0, len(ops))
	for _, op := range ops {
		ret = append(ret, &api_v1.StrategyRequest_Operation{
			Name: op.Name,
			Qps:  op.Qps,
		})
	}
	return ret
}
