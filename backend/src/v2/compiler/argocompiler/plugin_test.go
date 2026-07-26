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

package argocompiler

import (
	"encoding/json"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/require"
)

func requireDriverPluginArgs(t *testing.T, tmpl *wfapi.Template) map[string]interface{} {
	t.Helper()
	require.NotNil(t, tmpl)
	require.NotNil(t, tmpl.Plugin, "driver template should use an executor plugin")

	var pluginConfig map[string]map[string]map[string]interface{}
	require.NoError(t, json.Unmarshal(tmpl.Plugin.Value, &pluginConfig))
	driverPlugin, ok := pluginConfig["driver-plugin"]
	require.True(t, ok, "driver-plugin config should exist")
	args, ok := driverPlugin["args"]
	require.True(t, ok, "driver-plugin args should exist")
	return args
}
