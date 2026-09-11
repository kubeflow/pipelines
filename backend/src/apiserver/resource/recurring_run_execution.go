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
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createRunExecution uses Kubernetes name uniqueness to arbitrate concurrent
// recurring-run submissions before either request has persisted its run.
// The boolean reports whether this request created the execution.
func (r *ResourceManager) createRunExecution(ctx context.Context, run *model.Run, execution util.ExecutionSpec) (util.ExecutionSpec, bool, error) {
	if run.RecurringRunId != "" {
		execution.SetExecutionName("run-" + util.NewDeterministicUUID(run.UUID))
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
