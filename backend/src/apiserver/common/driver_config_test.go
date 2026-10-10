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

package common_test

import (
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestValidateDriverPodMetadataConfig(t *testing.T) {
	// Viper is process-global; these cases must not run in parallel.
	for _, name := range []string{"DRIVER_POD_LABELS", "DRIVER_POD_ANNOTATIONS"} {
		for _, test := range []struct {
			name      string
			value     interface{}
			wantError bool
		}{
			{name: "unset"},
			{name: "empty string", value: ""},
			{name: "whitespace", value: " \t\n"},
			{name: "empty JSON object", value: " { } "},
			{name: "JSON null", value: "null"},
			{name: "empty string map", value: map[string]string{}},
			{name: "empty config map", value: map[string]interface{}{}},
			{name: "string map", value: map[string]string{"app": "driver"}, wantError: true},
			{name: "config map", value: map[string]interface{}{"app": "driver"}, wantError: true},
			{name: "JSON labels", value: `{"sidecar.istio.io/inject":"true"}`, wantError: true},
			{name: "reserved metadata", value: `{"pipelines.kubeflow.org/key":"value"}`, wantError: true},
			{name: "empty value", value: `{"app":""}`, wantError: true},
			{name: "invalid JSON", value: "{", wantError: true},
			{name: "JSON array", value: "[]", wantError: true},
			{name: "native array", value: []string{}, wantError: true},
			{name: "boolean", value: true, wantError: true},
		} {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				if test.value != nil {
					viper.Set(name, test.value)
				}
				err := common.ValidateDriverPodMetadataConfig()
				if test.wantError {
					require.ErrorContains(t, err, name)
					require.ErrorContains(t, err, "remove it")
					require.ErrorContains(t, err, "agent-targeted")
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestValidateDriverPodMetadataConfigSources(t *testing.T) {
	for _, name := range []string{"DRIVER_POD_LABELS", "DRIVER_POD_ANNOTATIONS"} {
		t.Run(name+"/environment", func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.AutomaticEnv()
			t.Setenv(name, `{"app":"driver"}`)
			require.ErrorContains(t, common.ValidateDriverPodMetadataConfig(), name)
			t.Setenv(name, "{}")
			require.NoError(t, common.ValidateDriverPodMetadataConfig())
		})
		t.Run(name+"/config file", func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.SetConfigType("json")
			require.NoError(t, viper.ReadConfig(strings.NewReader(`{"`+name+`":{"app":"driver"}}`)))
			require.ErrorContains(t, common.ValidateDriverPodMetadataConfig(), name)
		})
	}
}
