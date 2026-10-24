// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type launcherReportStore interface {
	ReportTasks(*model.TaskReportIdentity, []*model.Task) ([]*model.Task, error)
	ReportArtifacts(*model.TaskReportIdentity, []*model.Artifact, []*model.ArtifactTask, bool) ([]*model.Artifact, []*model.ArtifactTask, error)
}

// ReportCreateTask preserves Argo creation but rejects managed materialization.
func (r *ResourceManager) ReportCreateTask(task *model.Task) (*model.Task, error) {
	return r.createTask(task, true)
}

func launcherReportError(err error) error {
	if errors.Is(err, storage.ErrRuntimeTaskConflict) || errors.Is(err, storage.ErrRuntimeRunCanceled) ||
		errors.Is(err, storage.ErrTaskReportIdentityRequired) || errors.Is(err, storage.ErrTaskReportScope) {
		return util.NewFailedPreconditionError(err, "Launcher report rejected: the current task attempt and its run must still permit this mutation")
	}
	return err
}

// ReportTasks routes launcher writes through the transactional attempt fence,
// keeping trusted coordinator writes separate from external reports.
func (r *ResourceManager) ReportTasks(identity *model.TaskReportIdentity, tasks []*model.Task) ([]*model.Task, error) {
	for _, task := range tasks {
		if task == nil {
			return nil, util.NewInvalidInputError("Reported task cannot be nil")
		}
		existing, err := r.taskStore.GetTask(task.UUID)
		if err != nil {
			return nil, err
		}
		if err := r.checkRunAllowsRuntimeWrites(existing.RunUUID); err != nil {
			return nil, err
		}
	}
	store, ok := r.taskStore.(launcherReportStore)
	if !ok {
		return nil, util.NewInternalServerError(errors.New("task store does not implement transactional launcher reporting"), "Launcher reporting storage is not configured")
	}
	updated, err := store.ReportTasks(identity, tasks)
	return updated, launcherReportError(err)
}

// ReportArtifacts atomically fences both artifact metadata and task links.
func (r *ResourceManager) ReportArtifacts(identity *model.TaskReportIdentity, artifacts []*model.Artifact, links []*model.ArtifactTask, reuse bool) ([]*model.Artifact, []*model.ArtifactTask, error) {
	if err := r.checkArtifactTasksAllowRuntimeWrites(links); err != nil {
		return nil, nil, err
	}
	store, ok := r.taskStore.(launcherReportStore)
	if !ok {
		return nil, nil, util.NewInternalServerError(errors.New("task store does not implement transactional launcher reporting"), "Launcher reporting storage is not configured")
	}
	created, createdLinks, err := store.ReportArtifacts(identity, artifacts, links, reuse)
	return created, createdLinks, launcherReportError(err)
}
