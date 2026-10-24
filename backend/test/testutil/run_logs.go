// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/test/config"
)

// ReadRunLog uses the suite's resolved token, or the live projected token in Kubeflow mode.
func ReadRunLog(ctx context.Context, runID, nodeID, authToken string) (string, error) {
	if runID == "" || nodeID == "" {
		return "", fmt.Errorf("run and node/task IDs are required to read logs")
	}
	logURL := fmt.Sprintf("%s/apis/v2/runs/%s/nodes/%s/log?follow=false",
		strings.TrimRight(*config.ApiUrl, "/"), url.PathEscape(runID), url.PathEscape(nodeID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, logURL, nil)
	if err != nil {
		return "", err
	}
	if *config.KubeflowMode {
		authToken, err = api_server.ReadProjectedSAToken()
		if err != nil {
			return "", err
		}
	}
	if authToken != "" {
		request.Header.Set("Authorization", "Bearer "+authToken)
	}
	tlsConfig, err := GetTLSConfig(*config.CaCertPath)
	if err != nil {
		return "", err
	}
	if tlsConfig == nil {
		return "", fmt.Errorf("the configured API CA certificate could not be loaded")
	}
	tlsConfig.InsecureSkipVerify = *config.DisableTLSCheck
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment, TLSClientConfig: tlsConfig,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return string(contents), fmt.Errorf("run log request returned %s", response.Status)
	}
	return string(contents), nil
}
