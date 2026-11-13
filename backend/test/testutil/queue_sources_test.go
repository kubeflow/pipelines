// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/stretchr/testify/require"
)

type queueSourceTransport func(*http.Request) (*http.Response, error)

func (f queueSourceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestQueueSDKSource(t *testing.T) {
	repo, pull, branch := *config.REPO_NAME, *config.PULL_NUMBER, *config.BRANCH_NAME
	t.Cleanup(func() { *config.REPO_NAME, *config.PULL_NUMBER, *config.BRANCH_NAME = repo, pull, branch })
	*config.REPO_NAME = "kubeflow/pipelines"
	*config.PULL_NUMBER = ""
	*config.BRANCH_NAME = strings.Repeat("a", 40)
	require.Equal(t, "git+https://github.com/kubeflow/pipelines.git@"+*config.BRANCH_NAME+"#subdirectory=sdk/python", getPackagePath("sdk/python"))
	*config.PULL_NUMBER = "7"
	require.Equal(t, "git+https://github.com/kubeflow/pipelines.git@refs/pull/7/merge#subdirectory=sdk/python", getPackagePath("sdk/python"))
}

func TestQueueRawFixtureSource(t *testing.T) {
	t.Setenv("KFP_TEST_FIXTURE_BASE_URL", "")
	t.Setenv("PULL_NUMBER", "")
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	sha := strings.Repeat("a", 40)
	want := "https://raw.githubusercontent.com/kubeflow/pipelines/" + sha + "/fixture.yaml"
	http.DefaultTransport = queueSourceTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, want, r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	got, err := GetRepoBranchURLRAW("kubeflow/pipelines", sha, "fixture.yaml")
	require.NoError(t, err)
	require.Equal(t, want, got)
}
