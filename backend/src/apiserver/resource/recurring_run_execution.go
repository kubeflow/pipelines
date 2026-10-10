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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	annotationKeyRecurringRunPipelineVersion = "pipelines.kubeflow.org/recurring-run-pipeline-version"
	recurringRunRuntimeConfigParameter       = "kfp-recurring-run-runtime-config"
)

// createRunExecution uses Kubernetes name uniqueness to arbitrate concurrent
// recurring-run submissions before either request has persisted its run.
// The boolean reports whether this request created the execution.
func (r *ResourceManager) createRunExecution(ctx context.Context, run *model.Run, execution util.ExecutionSpec) (util.ExecutionSpec, bool, error) {
	if run.RecurringRunId != "" {
		execution.SetExecutionName("run-" + util.NewDeterministicUUID(run.UUID))
		// The first workflow report can precede the run insert. Record the
		// selected version so recovery never resolves a newer default version.
		execution.SetAnnotations(annotationKeyRecurringRunPipelineVersion, run.PipelineVersionId)
		if !common.IsMultiUserMode() {
			config, err := json.Marshal(run.RuntimeConfig)
			if err != nil {
				return nil, false, util.Wrap(err, "Failed to preserve recurring-run runtime config")
			}
			// Arguments avoid the annotation size limit; base64 keeps template
			// substitution from modifying the submitted parameter strings.
			execution.SetSpecParameter(recurringRunRuntimeConfigParameter, base64.StdEncoding.EncodeToString(config))
		}
	}
	client := r.getWorkflowClient(execution.ExecutionNamespace())
	created, err := client.Create(ctx, execution, metav1.CreateOptions{})
	if err == nil {
		return created, true, nil
	}
	if run.RecurringRunId == "" || !apierrors.IsAlreadyExists(err) {
		return nil, false, err
	}
	existing, err := client.Get(ctx, execution.ExecutionName(), metav1.GetOptions{})
	if err != nil {
		return nil, false, util.Wrap(err, "Failed to retrieve the existing recurring-run workflow")
	}
	if !sameRecurringRunExecution(run, execution, existing) {
		return nil, false, util.NewPermissionDeniedError(
			fmt.Errorf("existing workflow identity does not match recurring run %q", run.RecurringRunId),
			"Cannot reuse the conflicting recurring-run workflow; resolve the conflicting Kubernetes object")
	}
	return existing, false, nil
}

func (r *ResourceManager) recurringRunReportPipelineSpec(job *model.Job, execution util.ExecutionSpec) (model.PipelineSpec, error) {
	pipelineSpec := job.PipelineSpec
	metadata := execution.ExecutionObjectMeta()
	versionID, apiCreated := metadata.Annotations[annotationKeyRecurringRunPipelineVersion]
	if !apiCreated {
		return pipelineSpec, nil
	}
	if common.IsMultiUserMode() {
		// Workflow annotations are editable by namespace users. The durable
		// scheduling claim is authoritative for API-created multi-user runs.
		claim, err := r.jobStore.GetRecurringRunState(job.UUID)
		if err != nil {
			return model.PipelineSpec{}, util.Wrap(err, "Failed to read the recurring-run scheduling claim")
		}
		runID := util.NewDeterministicUUID(job.UUID + "/tick/" + strconv.FormatInt(claim.LastRunIndex, 10))
		if claim.LastRunIndex <= 0 || metadata.Labels[util.LabelKeyWorkflowRunId] != runID || versionID != claim.PipelineVersionID {
			return model.PipelineSpec{}, util.NewInvalidInputError("Failed to recover recurring run: workflow does not match the selected scheduling claim")
		}
	} else {
		config, err := recurringRunRuntimeConfigSnapshot(execution)
		if err != nil {
			return model.PipelineSpec{}, err
		}
		if config != nil {
			pipelineSpec.RuntimeConfig = *config
		}
	}
	if versionID == "" {
		return pipelineSpec, nil
	}
	if job.PipelineId == "" && job.PipelineVersionId == "" {
		return model.PipelineSpec{}, util.NewInvalidInputError("Failed to recover recurring run: an inline pipeline cannot select a stored pipeline version")
	}
	if job.PipelineVersionId != "" && job.PipelineVersionId != versionID {
		return model.PipelineSpec{}, util.NewInvalidInputError("Failed to recover recurring run: workflow pipeline version differs from the pinned recurring-run version")
	}
	if job.PipelineVersionId == versionID && pipelineSpec.PipelineSpecManifest != "" {
		return pipelineSpec, nil
	}
	pipelineSpec.PipelineVersionId = versionID
	if _, _, err := r.fetchTemplateFromPipelineSpec(&pipelineSpec); err != nil {
		return model.PipelineSpec{}, util.Wrap(err, "Failed to recover the selected recurring-run pipeline version")
	}
	return pipelineSpec, nil
}

// Older workflows have no snapshot and retain the recurring-run configuration.
func recurringRunRuntimeConfigSnapshot(execution util.ExecutionSpec) (*model.RuntimeConfig, error) {
	for _, parameter := range execution.SpecParameters() {
		if parameter.Name != recurringRunRuntimeConfigParameter {
			continue
		}
		if parameter.Value == nil {
			return nil, util.NewInvalidInputError("Cannot recover recurring-run runtime config from an empty snapshot; recreate the run")
		}
		data, err := base64.StdEncoding.DecodeString(*parameter.Value)
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Cannot decode recurring-run runtime config snapshot; recreate the run")
		}
		var config *model.RuntimeConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Cannot parse recurring-run runtime config snapshot; recreate the run")
		}
		if config == nil {
			return nil, util.NewInvalidInputError("Cannot recover recurring-run runtime config from a null snapshot; recreate the run")
		}
		return config, nil
	}
	return nil, nil
}

func sameRecurringRunExecution(run *model.Run, requested, existing util.ExecutionSpec) bool {
	if existing == nil || existing.ExecutionUID() == "" ||
		existing.ExecutionName() != requested.ExecutionName() ||
		existing.ExecutionNamespace() != requested.ExecutionNamespace() ||
		existing.ServiceAccount() != requested.ServiceAccount() ||
		existing.ScheduledWorkflowUUIDAsStringOrEmpty() != run.RecurringRunId {
		return false
	}
	metadata := existing.ExecutionObjectMeta()
	if metadata.Labels[util.LabelKeyWorkflowRunId] != run.UUID ||
		metadata.Annotations[util.AnnotationKeyRunName] != run.DisplayName {
		return false
	}
	for _, expectedOwner := range requested.ExecutionObjectMeta().OwnerReferences {
		if string(expectedOwner.UID) != run.RecurringRunId {
			continue
		}
		for _, actualOwner := range metadata.OwnerReferences {
			if actualOwner.UID == expectedOwner.UID && actualOwner.Name == expectedOwner.Name &&
				actualOwner.APIVersion == expectedOwner.APIVersion && actualOwner.Kind == expectedOwner.Kind {
				return true
			}
		}
	}
	return false
}
