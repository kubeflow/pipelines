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

package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestImagePullFailureWaitTimeoutPreservesDuration(t *testing.T) {
	for _, grace := range []time.Duration{0, time.Minute, 5 * time.Minute} {
		t.Run(grace.String(), func(t *testing.T) {
			timeout := imagePullFailureWaitTimeout(grace)
			require.Equal(t, grace+4*time.Minute, timeout)
			require.GreaterOrEqual(t, timeout, 4*time.Minute)
		})
	}

}

func TestImagePullFailureConfigurationFollowsExecutionEngine(t *testing.T) {
	deployment, container := imagePullFailureBackendDeployment(false)
	require.Equal(t, "ml-pipeline-persistenceagent", deployment)
	require.Equal(t, "ml-pipeline-persistenceagent", container)
	deployment, container = imagePullFailureBackendDeployment(true)
	require.Equal(t, "ml-pipeline", deployment)
	require.Equal(t, "ml-pipeline-api-server", container)
}
