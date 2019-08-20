// Copyright 2018 Google LLC
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

package util

import constants "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"

const (
	// LabelKeyScheduledWorkflowEnabled is a label on a ScheduledWorkflow.
	// It captures whether the ScheduledWorkflow is enabled.
	LabelKeyScheduledWorkflowEnabled = constants.FullName + "/enabled"
	// LabelKeyScheduledWorkflowStatus is a label on a ScheduledWorkflow.
	// It captures the status of the scheduled workflow.
	LabelKeyScheduledWorkflowStatus = constants.FullName + "/status"

	// The maximum byte sizes of the parameter column in package/pipeline DB.
	MaxParameterBytes = 10000

	// LabelKeyWorkflowEpoch is a label on a Workflow.
	// It captures the epoch at which the workflow was scheduled.
	LabelKeyWorkflowEpoch = constants.FullName + "/workflowEpoch"
	// LabelKeyWorkflowIndex is a label on a Workflow.
	// It captures the index of creation the workflow by the ScheduledWorkflow.
	LabelKeyWorkflowIndex = constants.FullName + "/workflowIndex"
	// LabelKeyWorkflowIsOwnedByScheduledWorkflow is a label on a Workflow.
	// It captures whether the workflow is owned by a ScheduledWorkflow.
	LabelKeyWorkflowIsOwnedByScheduledWorkflow = constants.FullName + "/isOwnedByScheduledWorkflow"
	// LabelKeyWorkflowScheduledWorkflowName is a label on a Workflow.
	// It captures whether the name of the owning ScheduledWorkflow.
	LabelKeyWorkflowScheduledWorkflowName = constants.FullName + "/scheduledWorkflowName"


	LabelKeyWorkflowRunId = "pipeline/runid"
	LabelKeyWorkflowPersistedFinalState = "pipeline/persistedFinalState"
)
