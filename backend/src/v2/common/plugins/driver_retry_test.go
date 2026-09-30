// Copyright 2026 The Kubeflow Authors
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

package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type unknownRetryDispatcher struct {
	NoOpDispatcher
}

func TestValidateDriverRetry_NoPlugins(t *testing.T) {
	for _, dispatcher := range []TaskPluginDispatcher{
		nil, NoOpDispatcher{}, &NoOpDispatcher{}, &TaskPluginDispatcherImpl{},
	} {
		require.NoError(t, ValidateDriverRetry(dispatcher))
	}
}

func TestValidateDriverRetry_RejectsPluginsAndUnknownDispatchers(t *testing.T) {
	active, err := NewTaskPluginDispatcherImpl([]TaskPluginHandler{&fakeHandler{name: "mlflow"}})
	require.NoError(t, err)
	for _, dispatcher := range []TaskPluginDispatcher{
		active, &unknownRetryDispatcher{}, (*TaskPluginDispatcherImpl)(nil), (*NoOpDispatcher)(nil),
	} {
		err := ValidateDriverRetry(dispatcher)
		require.ErrorContains(t, err, "do not support replay")
		require.ErrorContains(t, err, "disable driver retries or submit the run through the KFP API server")
	}
}
