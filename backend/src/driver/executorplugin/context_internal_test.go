// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package executorplugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	runtimemlflow "github.com/kubeflow/pipelines/backend/src/v2/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
)

func TestInvocationProxyAndVerbosityAreLocal(t *testing.T) {
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"} {
		t.Setenv(name, "")
	}
	var wg sync.WaitGroup
	for _, level := range []string{"0", "3", "4", "8"} {
		level := level
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "mlflow.invalid", r.URL.Host)
			_, _ = fmt.Fprintf(w, `{"run":{"info":{"run_id":"%s"}}}`, level)
		}))
		defer proxy.Close()
		runner, args, _, _ := invocationFixture(t, containerDriver)
		args.HTTPProxy = proxy.URL
		args.LogLevel = level
		runner.execute = func(ctx context.Context, _ drivercommon.Options, _ client_manager.ClientManagerInterface) (*driver.Execution, error) {
			util.GetLoggerFrom(ctx).Trace("verbose-driver-diagnostic")
			util.GetLoggerFrom(ctx).Info("ordinary-driver-diagnostic")
			request, err := runtimemlflow.BuildMLflowTaskRequestContext(ctx, commonmlflow.MLflowRuntimeConfig{Endpoint: "http://mlflow.invalid", Timeout: "1s", AuthType: commonmlflow.AuthTypeNone})
			if err != nil {
				return nil, err
			}
			id, err := request.Client.CreateRun(ctx, "experiment", "task", nil)
			assert.Equal(t, level, id)
			return &driver.Execution{TaskID: id}, err
		}
		runner.uploadLogs = func(_ context.Context, log *driverLogArtifactContext, _ kubernetes.Interface) error {
			data, err := os.ReadFile(log.LocalPath)
			assert.NoError(t, err)
			assert.Contains(t, string(data), "ordinary-driver-diagnostic")
			assert.Equal(t, level == "4" || level == "8", strings.Contains(string(data), "verbose-driver-diagnostic"))
			return nil
		}
		wg.Add(1)
		go func() { defer wg.Done(); _, err := runner.drive(context.Background(), args); assert.NoError(t, err) }()
	}
	wg.Wait()
}

func TestInvocationProxyDirectAndNoProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://process-proxy.invalid")
	for _, noProxy := range []string{"", "mlflow.internal"} {
		runner, args, _, _ := invocationFixture(t, containerDriver)
		args.NoProxy = noProxy
		if noProxy != "" {
			args.HTTPProxy = "http://request-proxy.invalid"
		}
		runner.execute = func(ctx context.Context, _ drivercommon.Options, _ client_manager.ClientManagerInterface) (*driver.Execution, error) {
			resolver, ok := util.HTTPProxyFrom(ctx)
			assert.True(t, ok)
			req, _ := http.NewRequest("GET", "http://mlflow.internal/api", nil)
			selected, err := resolver(req)
			assert.NoError(t, err)
			assert.Nil(t, selected)
			return &driver.Execution{}, nil
		}
		runner.uploadLogs = func(context.Context, *driverLogArtifactContext, kubernetes.Interface) error { return nil }
		_, err := runner.drive(context.Background(), args)
		require.NoError(t, err)
	}
}

func TestInvalidInvocationVerbosity(t *testing.T) {
	for _, level := range []string{"", "-1", "verbose"} {
		runner, args, _, _ := invocationFixture(t, containerDriver)
		args.LogLevel = level
		_, err := runner.drive(context.Background(), args)
		require.ErrorContains(t, err, "nonnegative integer")
	}
}
