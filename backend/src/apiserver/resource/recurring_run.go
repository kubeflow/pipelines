// Copyright 2018 The Kubeflow Authors
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

// Package resource coordinates API resource persistence and execution.
package resource

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflow "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
)

// PrepareRecurringRun resolves execution inputs from the API-created job rather than
// the editable ScheduledWorkflow. Service-account and pipeline authorization must still run afterwards.
func (r *ResourceManager) PrepareRecurringRun(ctx context.Context, run *model.Run) error {
	if !common.IsMultiUserMode() || run.RecurringRunId == "" {
		return nil
	}
	job, err := r.GetJob(run.RecurringRunId)
	if err != nil {
		return util.Wrap(err, "Failed to resolve the authorized recurring run; create schedules through the KFP API")
	}
	namespace := job.Namespace
	if r.IsEmptyNamespace(namespace) {
		namespace, err = r.GetNamespaceFromExperimentId(job.ExperimentId)
		if err != nil {
			return err
		}
	}
	if r.IsEmptyNamespace(namespace) {
		return util.NewPermissionDeniedError(fmt.Errorf("recurring run has no namespace"),
			"Recreate the recurring run in a namespaced experiment")
	}
	// Authorize access before exposing schedule state or copying its execution inputs.
	if err := r.IsAuthorized(ctx, &authorizationv1.ResourceAttributes{
		Namespace: namespace, Verb: common.RbacResourceVerbCreate,
		Group: common.RbacPipelinesGroup, Version: common.RbacPipelinesVersion, Resource: common.RbacResourceTypeRuns,
	}); err != nil {
		return util.Wrap(err, "Failed to authorize scheduled run creation")
	}
	if (!r.IsEmptyNamespace(run.Namespace) && run.Namespace != namespace) ||
		(run.ExperimentId != "" && run.ExperimentId != job.ExperimentId) {
		return util.NewPermissionDeniedError(fmt.Errorf("recurring run destination differs from its authorized job"),
			"A recurring run can only create runs in its own namespace and experiment")
	}
	if job.ExperimentId != "" {
		experimentNamespace, err := r.GetNamespaceFromExperimentId(job.ExperimentId)
		if err != nil {
			return err
		}
		if !r.IsEmptyNamespace(experimentNamespace) && experimentNamespace != namespace {
			return util.NewPermissionDeniedError(fmt.Errorf("recurring run and experiment namespaces differ"),
				"A recurring run can only create runs in its own namespace and experiment")
		}
	}
	swf, err := r.getScheduledWorkflowClient(namespace).Get(ctx, job.K8SName, v1.GetOptions{})
	if err != nil {
		return util.Wrap(err, "Failed to retrieve the authorized ScheduledWorkflow")
	}
	if swf == nil || string(swf.UID) != job.UUID || swf.Name != job.K8SName || swf.Namespace != namespace {
		return util.NewPermissionDeniedError(fmt.Errorf("ScheduledWorkflow identity does not match its job"),
			"Recreate the recurring run through the KFP API")
	}
	if !job.Enabled || !swf.Spec.Enabled {
		return util.NewPermissionDeniedError(fmt.Errorf("recurring run is disabled"),
			"Enable the recurring run through the KFP API before triggering runs")
	}
	run.Namespace = namespace
	run.ExperimentId = job.ExperimentId
	run.PipelineSpec = job.PipelineSpec
	run.ServiceAccount = job.ServiceAccount
	run.PluginsInputString = job.PluginsInputString
	return nil
}

// V1 workflows may carry the effective account inside their embedded Argo spec.
func scheduledServiceAccount(swf *scheduledworkflow.ScheduledWorkflow, fallback string) (string, error) {
	if swf.Spec.Workflow != nil && swf.Spec.Workflow.Spec != nil {
		execution, err := util.ScheduleSpecToExecutionSpec(util.ArgoWorkflow, swf.Spec.Workflow)
		if err != nil {
			return "", util.Wrap(err, "Failed to resolve the scheduled service account")
		}
		return execution.ServiceAccount(), nil
	}
	if swf.Spec.ServiceAccount != "" {
		return swf.Spec.ServiceAccount, nil
	}
	return fallback, nil
}
