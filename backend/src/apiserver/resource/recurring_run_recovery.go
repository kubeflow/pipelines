// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"fmt"
	"strconv"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// A report can insert the run before CreateRun does. Preserve the source already
// selected for this execution rather than resolving the schedule's latest again.
func (r *ResourceManager) recurringRunReportPipelineSpec(job *model.Job, execution util.ExecutionSpec, runID string) (model.PipelineSpec, error) {
	source := job.PipelineSpec
	// Inline schedules already retain their source in the authorized job.
	if source.PipelineId == "" && source.PipelineVersionId == "" {
		return source, nil
	}
	selectedVersion := ""
	if common.IsMultiUserMode() && source.PipelineVersionId == "" {
		claim, err := r.jobStore.GetRecurringRunState(job.UUID)
		if err != nil {
			return source, err
		}
		if claim.LastRunIndex > 0 && runID == util.NewDeterministicUUID(job.UUID+"/tick/"+strconv.FormatInt(claim.LastRunIndex, 10)) {
			selectedVersion = claim.PipelineVersionID
		}
	} else if !common.IsMultiUserMode() {
		selectedVersion = execution.ExecutionObjectMeta().Annotations[recurringRunPipelineVersionAnnotation]
	}
	if selectedVersion != "" {
		if source.PipelineVersionId != "" && source.PipelineVersionId != selectedVersion {
			return source, util.NewInvalidInputError("Cannot recover recurring run %s: selected pipeline version differs from its pinned schedule", runID)
		}
		source.PipelineVersionId = selectedVersion
	}
	if source.PipelineVersionId == "" {
		return source, util.NewFailedPreconditionError(fmt.Errorf("selected pipeline version is missing"),
			"Cannot recover recurring run %s without its exact pipeline version; allow the creating request to persist or restore the selected version metadata", runID)
	}
	if source.PipelineSpecManifest != "" {
		return source, nil
	}
	version, err := r.GetPipelineVersion(source.PipelineVersionId)
	if err != nil {
		return source, util.Wrapf(err, "Failed to restore the selected pipeline source for recurring run %s", runID)
	}
	if source.PipelineId != "" && version.PipelineId != source.PipelineId {
		return source, util.NewInvalidInputError("Cannot recover recurring run %s: selected pipeline version belongs to a different pipeline", runID)
	}
	if version.PipelineSpec == "" {
		return source, util.NewFailedPreconditionError(fmt.Errorf("selected pipeline manifest is missing"),
			"Restore the manifest of pipeline version %s before recovering recurring run %s", version.UUID, runID)
	}
	source.PipelineId = version.PipelineId
	source.PipelineName = version.Name
	source.PipelineSpecManifest = version.PipelineSpec
	source.WorkflowSpecManifest = ""
	return source, nil
}
