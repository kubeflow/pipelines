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

package main

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/stretchr/testify/require"
	"testing"
)

type failingRetryPluginFactory struct{}

func (failingRetryPluginFactory) Name() string    { return "failing-retry-test" }
func (failingRetryPluginFactory) IsEnabled() bool { return true }
func (failingRetryPluginFactory) Create() (plugins.TaskPluginHandler, error) {
	return nil, fmt.Errorf("plugin unavailable")
}

func TestDriverRetryDoesNotSilentlyDisableFailedPluginInitialization(t *testing.T) {
	factories := plugins.RegisteredFactories()
	plugins.ResetRegistry()
	t.Cleanup(func() {
		plugins.ResetRegistry()
		for _, f := range factories {
			plugins.RegisterHandlerFactory(f)
		}
	})
	dispatcher, err := pluginDispatcherForDriver(true)
	require.NoError(t, err)
	require.NotNil(t, dispatcher)
	plugins.RegisterHandlerFactory(failingRetryPluginFactory{})
	_, err = pluginDispatcherForDriver(true)
	require.ErrorContains(t, err, "plugin unavailable")
	dispatcher, err = pluginDispatcherForDriver(false)
	require.NoError(t, err, "legacy driver fallback must remain unchanged")
	require.IsType(t, plugins.NoOpDispatcher{}, dispatcher)
}
