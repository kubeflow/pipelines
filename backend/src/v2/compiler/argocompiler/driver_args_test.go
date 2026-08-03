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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// registeredDriverFlags mirrors flags defined in backend/src/v2/cmd/driver/main.go.
// Keep in sync when adding compiler-emitted driver arguments.
var registeredDriverFlags = map[string]struct{}{
	"--type":                       {},
	"--pipeline_name":              {},
	"--run_id":                     {},
	"--run_name":                   {},
	"--run_display_name":           {},
	"--runtime_config":             {},
	"--iteration_index":            {},
	"--task_name":                  {},
	"--namespace":                  {},
	"--parent_task_id":             {},
	"--kubernetes_config":          {},
	"--ml_pipeline_server_address": {},
	"--ml_pipeline_server_port":    {},
	"--parent_task_id_path":        {},
	"--iteration_count_path":       {},
	"--pod_spec_patch_path":        {},
	"--cached_decision_path":       {},
	"--condition_path":             {},
	"--log_level":                  {},
	"--http_proxy":                 {},
	"--https_proxy":                {},
	"--no_proxy":                   {},
	"--publish_logs":               {},
	"--cache_disabled":             {},
	"--ml_pipeline_tls_enabled":    {},
	"--ca_cert_path":               {},
	"--default_run_as_user":        {},
	"--default_run_as_group":       {},
	"--default_run_as_non_root":    {},
	"--default_host_users":         {},
}

func assertRegisteredDriverArgs(t *testing.T, args []string) {
	t.Helper()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		_, ok := registeredDriverFlags[arg]
		require.Truef(t, ok, "compiler emits unregistered driver flag %q at index %d", arg, i)
		// Boolean flags may omit an explicit value.
		switch arg {
		case "--cache_disabled", "--ml_pipeline_tls_enabled":
			continue
		default:
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
			}
		}
	}
}
