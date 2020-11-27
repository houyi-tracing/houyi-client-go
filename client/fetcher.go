// Copyright (c) 2020 Fuhai Yan.
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

package client

import (
	"encoding/json"
	"fmt"
	"github.com/houyi-tracing/houyi-client-go/client/model"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type SamplingStrategyFetcher interface {
	Fetch(string, []model.Operation, time.Duration) ([]byte, error)
}

type httpSamplingStrategyFetcher struct {
	serverURL string
	logger    *zap.Logger
}

func NewSamplingStrategyFetcher(serverURL string, logger *zap.Logger) SamplingStrategyFetcher {
	return &httpSamplingStrategyFetcher{
		serverURL: serverURL,
		logger:    logger,
	}
}

func (f *httpSamplingStrategyFetcher) Fetch(serviceName string, operations []model.Operation, interval time.Duration) ([]byte, error) {
	params := url.Values{}
	for _, op := range operations {
		if jsonStr, err := json.Marshal(op); err != nil {
			f.logger.Error(err.Error())
		} else {
			params.Add("operation", string(jsonStr))
		}
	}
	params.Set("service", serviceName)
	params.Set("interval", interval.String())

	f.logger.Debug("pulling sampling strategy",
		zap.String("service", serviceName),
		zap.String("interval", interval.String()))

	uri := fmt.Sprintf("%s?%s", f.serverURL, params.Encode())
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			f.logger.Error("failed to close HTTP response body", zap.Error(err))
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("StatusCode: %d, Body: %s", resp.StatusCode, body)
	}

	return body, nil
}
