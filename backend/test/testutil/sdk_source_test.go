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
	"flag"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/test/config"
)

func TestPackagePathPinsCheckedOutSDKSource(t *testing.T) {
	source := flag.Lookup("sdkSourceSHA")
	if source == nil {
		t.Fatal("SDK source cannot be pinned: missing sdkSourceSHA flag")
	}
	oldSource, oldRepo, oldPull, oldBranch := config.SDKSourceSHA, *config.REPO_NAME, *config.PULL_NUMBER, *config.BRANCH_NAME
	t.Cleanup(func() {
		_ = source.Value.Set(oldSource)
		*config.REPO_NAME = oldRepo
		*config.PULL_NUMBER = oldPull
		*config.BRANCH_NAME = oldBranch
	})
	*config.REPO_NAME = "kubeflow/pipelines"
	*config.BRANCH_NAME = "release-2.17"
	for _, pull := range []string{"15040", ""} {
		*config.PULL_NUMBER = pull
		sha := strings.Repeat("a", 40)
		_ = source.Value.Set(sha)
		for _, subdir := range []string{"sdk/python", "kubernetes_platform/python"} {
			want := "git+https://github.com/kubeflow/pipelines.git@" + sha + "#subdirectory=" + subdir
			if got := getPackagePath(subdir); got != want {
				t.Errorf("pull=%q: got %s, want %s", pull, got, want)
			}
		}
	}
	for _, invalid := range []string{"release-2.17", "refs/pull/15040/merge", strings.Repeat("a", 39), strings.Repeat("g", 40)} {
		if err := source.Value.Set(invalid); err == nil {
			t.Errorf("accepted mutable or invalid SDK source %q", invalid)
		}
	}
	_ = source.Value.Set("")
	for _, tc := range []struct{ pull, ref string }{{"15040", "refs/pull/15040/merge"}, {"", "release-2.17"}} {
		*config.PULL_NUMBER = tc.pull
		want := "git+https://github.com/kubeflow/pipelines.git@" + tc.ref + "#subdirectory=sdk/python"
		if got := getPackagePath("sdk/python"); got != want {
			t.Errorf("fallback got %s, want %s", got, want)
		}
	}
}
