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

package driverapi

import "fmt"

// DriverType selects which driver runs for a task.
type DriverType string

const (
	// RootDAGDriver starts a run and resolves pipeline-level inputs.
	RootDAGDriver DriverType = "ROOT_DAG"
	// DAGDriver resolves a nested DAG, loop, or condition.
	DAGDriver DriverType = "DAG"
	// ContainerDriver resolves a container task and derives its pod patch.
	ContainerDriver DriverType = "CONTAINER"
)

// driverTypes is the single description of what each driver needs, so callers
// do not switch on the type themselves.
var driverTypes = map[DriverType]struct {
	requiredArgs []string
	dag          bool
}{
	RootDAGDriver:   {requiredArgs: []string{"runtime_config"}, dag: true},
	DAGDriver:       {dag: true},
	ContainerDriver: {requiredArgs: []string{"kubernetes_config"}},
}

// commonRequiredArgFields must be present, possibly empty, in every driver request.
var commonRequiredArgFields = []string{
	"type", "pipeline_name", "run_id", "run_name", "run_display_name",
	"parent_task_id", "task_name", "namespace", "iteration_index",
	"ml_pipeline_server_address", "ml_pipeline_server_port", "log_level", "publish_logs",
	"cache_disabled", "ml_pipeline_tls_enabled", "http_proxy", "https_proxy", "no_proxy",
}

func (t DriverType) String() string { return string(t) }

// Validate reports an unknown driver type.
func (t DriverType) Validate() error {
	if t == "" {
		return fmt.Errorf("driver type is empty; set type to %s, %s, or %s", RootDAGDriver, DAGDriver, ContainerDriver)
	}
	if _, ok := driverTypes[t]; !ok {
		return fmt.Errorf("unknown driver type %q; use %s, %s, or %s", t, RootDAGDriver, DAGDriver, ContainerDriver)
	}
	return nil
}

// IsDAG reports whether the driver produces an iteration count.
func (t DriverType) IsDAG() bool { return driverTypes[t].dag }

// RequiredArgFields lists the argument fields a request of this type must carry.
func (t DriverType) RequiredArgFields() []string {
	return append(append([]string{}, commonRequiredArgFields...), driverTypes[t].requiredArgs...)
}
