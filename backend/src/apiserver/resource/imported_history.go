// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"errors"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func importedRunMutationError(run *model.Run) error {
	if run.ImportedFrom == "" {
		return nil
	}
	return util.NewFailedPreconditionError(errors.New("the execution belongs to another installation"),
		"Run %s is imported history and cannot accept runtime changes. Create a new run to execute it in this installation", run.UUID)
}

var terminalRunStateStrings = []string{
	string(model.RuntimeStateSucceeded),
	string(model.RuntimeStateFailed),
	string(model.RuntimeStateSkipped),
	string(model.RuntimeStateCanceled),
	string(model.RuntimeStateSucceededV1),
	string(model.RuntimeStateFailedV1),
	string(model.RuntimeStateSkippedV1),
	string(model.RuntimeStateErrorV1),
	model.LegacyStateDone,
	model.LegacyStateDisabled,
}

var terminalRunStateValues = func() map[string]bool {
	values := make(map[string]bool, len(terminalRunStateStrings))
	for _, state := range terminalRunStateStrings {
		values[state] = true
	}
	return values
}()

func terminalRunMutationError(run *model.Run) error {
	originalState := string(run.State)
	isTerminal := terminalRunStateValues[originalState]
	if isTerminal {
		return util.NewFailedPreconditionError(
			errors.New("the execution is in a terminal state"),
			"Run %s is in terminal state %s and cannot accept runtime changes", run.UUID, originalState,
		)
	}
	return nil
}

func (r *ResourceManager) checkRunAllowsRuntimeWrites(runID string) error {
	run, err := r.runStore.GetRun(runID, false)
	if err != nil {
		return util.Wrapf(err, "Failed to check whether run %s accepts runtime changes", runID)
	}
	if err := importedRunMutationError(run); err != nil {
		return err
	}

	if err := terminalRunMutationError(run); err != nil {
		return err
	}
	return nil
}

func (r *ResourceManager) checkArtifactTasksAllowRuntimeWrites(links []*model.ArtifactTask) error {
	checked := make(map[string]bool, len(links))
	for _, link := range links {
		if link == nil {
			return util.NewInvalidInputError("Artifact-task relationship cannot be nil")
		}
		if checked[link.RunUUID] {
			continue
		}
		if err := r.checkRunAllowsRuntimeWrites(link.RunUUID); err != nil {
			return err
		}
		checked[link.RunUUID] = true
	}
	return nil
}
