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

package plugins

import (
	"context"
	"encoding/json"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
)

// PluginsInputMap is the JSON envelope for the plugins_input model
// field. Each key is a plugin name and the value is that plugin's raw config.
type PluginsInputMap map[string]json.RawMessage

// PendingRun holds the minimal run information available before a KFP run is
// persisted.
type PendingRun struct {
	RunID             string
	DisplayName       string
	Namespace         string
	PipelineID        string
	PipelineVersionID string
	PluginsInput      *string // raw JSON from model, read by dispatcher
	PluginsOutput     *string // raw JSON produced by dispatcher, written back to model
}

// PersistedRun holds the minimal run information for post-run plugin hooks
type PersistedRun struct {
	RunID      string
	Namespace  string
	State      string     // RuntimeState string, e.g. "SUCCEEDED", "FAILED"
	FinishedAt *time.Time // nil if not yet finished
	// PluginsOutput is the deserialized plugins_output map.
	PluginsOutput map[string]*apiv2beta1.PluginOutput
}

// RunPluginHandler defines the generic run-level plugin lifecycle hooks
type RunPluginHandler interface {
	OnBeforeRunCreation(ctx context.Context, run *PendingRun, config interface{}) (*apiv2beta1.PluginOutput, error)
	OnRunEnd(ctx context.Context, run *PersistedRun, config interface{}) error
}
