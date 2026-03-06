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

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// RunPluginOutputStore abstracts the DB write needed to persist plugin output.
type RunPluginOutputStore interface {
	UpdateRunPluginsOutput(runID string, pluginsOutput *model.LargeText) error
}

// PluginDispatcher orchestrates plugin lifecycle hooks.
type PluginDispatcher interface {
	// OnBeforeRunCreation is called before the workflow is created.
	// The dispatcher reads run.PluginsInput and may write run.PluginsOutput.
	// Returns an error only for validation failures that should block
	// run creation.
	OnBeforeRunCreation(ctx context.Context, run *PendingRun, executionSpec util.ExecutionSpec) error

	// OnRunEnd is called when a run reaches a terminal state. Returns
	// true if all plugin syncs succeeded.
	OnRunEnd(ctx context.Context, run *PersistedRun) bool

	// OnRunRetry is called when a run is retried.
	OnRunRetry(ctx context.Context, run *PersistedRun)
}

// NoOpDispatcher is a PluginDispatcher that does nothing.
type NoOpDispatcher struct{}

func (NoOpDispatcher) OnBeforeRunCreation(context.Context, *PendingRun, util.ExecutionSpec) error {
	return nil
}
func (NoOpDispatcher) OnRunEnd(context.Context, *PersistedRun) bool { return true }
func (NoOpDispatcher) OnRunRetry(context.Context, *PersistedRun)    {}

var _ PluginDispatcher = NoOpDispatcher{}
