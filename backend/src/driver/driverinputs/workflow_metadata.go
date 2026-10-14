// Copyright 2025 The Kubeflow Authors
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

package driverinputs

import (
	"context"
	"fmt"
	"strconv"
	"time"

	workflowclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pipelineJobCreateTimeUTCPlaceholder   = "{{$.pipeline_job_create_time_utc}}"
	pipelineJobScheduleTimeUTCPlaceholder = "{{$.pipeline_job_schedule_time_utc}}"
)

// CurrentWorkflowMetadata reads the owning Workflow when the run lacks times.
func CurrentWorkflowMetadata(ctx context.Context, argo workflowclient.Interface, namespace string, workflowName string) (*metav1.ObjectMeta, error) {
	if workflowName == "" {
		return nil, fmt.Errorf("workflow name is empty; set run_name to the owning Workflow name")
	}
	workflow, err := argo.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow %q; check that it exists and the plugin service account has workflows/get permission: %w", workflowName, err)
	}
	return &workflow.ObjectMeta, nil
}

// WorkflowMetadataGetter abstracts the Workflow read for tests.
type WorkflowMetadataGetter func(ctx context.Context, namespace string, workflowName string) (*metav1.ObjectMeta, error)

// PlaceholderUsage records which pipeline job time placeholders a task uses.
type PlaceholderUsage struct {
	needsCreateTime   bool
	needsScheduleTime bool
}

// PipelineJobTimePlaceholderUsage scans a task's constant inputs for time placeholders.
func PipelineJobTimePlaceholderUsage(
	driverType driverapi.DriverType,
	taskSpec *pipelinespec.PipelineTaskSpec,
) PlaceholderUsage {
	usage := PlaceholderUsage{}
	if driverType == driverapi.RootDAGDriver || taskSpec == nil {
		return usage
	}
	for _, inputParamSpec := range taskSpec.GetInputs().GetParameters() {
		runtimeValue := inputParamSpec.GetRuntimeValue()
		if runtimeValue == nil {
			continue
		}
		constant := runtimeValue.GetConstant()
		if constant == nil {
			continue
		}
		switch constant.GetStringValue() {
		case pipelineJobCreateTimeUTCPlaceholder:
			usage.needsCreateTime = true
		case pipelineJobScheduleTimeUTCPlaceholder:
			usage.needsScheduleTime = true
		}
		if usage.needsCreateTime && usage.needsScheduleTime {
			return usage
		}
	}
	return usage
}

// WorkflowMetadataForPipelineJobTimes reads the Workflow only when a used
// placeholder cannot be filled from the run.
func WorkflowMetadataForPipelineJobTimes(
	ctx context.Context,
	namespace string,
	workflowName string,
	placeholderUsage PlaceholderUsage,
	createTimeUTC string,
	scheduleTimeEpochSeconds string,
	getMetadata WorkflowMetadataGetter,
) (*metav1.ObjectMeta, error) {
	needsCreateTimeMetadata := placeholderUsage.needsCreateTime && createTimeUTC == ""
	needsScheduleTimeMetadata := placeholderUsage.needsScheduleTime && scheduleTimeEpochSeconds == ""
	if !needsCreateTimeMetadata && !needsScheduleTimeMetadata {
		return nil, nil
	}
	workflowMeta, err := getMetadata(ctx, namespace, workflowName)
	if err != nil {
		if !needsCreateTimeMetadata && needsScheduleTimeMetadata && createTimeUTC != "" {
			glog.Warningf(
				"Failed to retrieve workflow metadata for pipeline job schedule time for workflow %q, falling back to create time: %v",
				workflowName,
				err,
			)
			return nil, nil
		}
		return nil, err
	}
	return workflowMeta, nil
}

func resolvePipelineJobScheduleTimeUTCFromWorkflow(
	workflowMeta *metav1.ObjectMeta,
	fallbackCreateTimeUTC string,
) string {
	if workflowMeta == nil {
		return fallbackCreateTimeUTC
	}
	createTimeUTC := fallbackCreateTimeUTC
	if createTimeUTC == "" {
		createTimeUTC = workflowMeta.CreationTimestamp.Time.UTC().Format(time.RFC3339)
	}
	value, ok := workflowMeta.Labels[util.LabelKeyWorkflowEpoch]
	if !ok {
		return createTimeUTC
	}
	scheduledEpochSeconds, err := util.RetrieveInt64FromLabel(value)
	if err != nil {
		return createTimeUTC
	}
	return time.Unix(scheduledEpochSeconds, 0).UTC().Format(time.RFC3339)
}

// ResolvePipelineJobTimes returns RFC3339 create and schedule times, falling
// back to Workflow metadata.
func ResolvePipelineJobTimes(
	createTimeUTC string,
	scheduleTimeEpochSeconds string,
	workflowMeta *metav1.ObjectMeta,
) (string, string, error) {
	if createTimeUTC == "" && workflowMeta != nil {
		createTimeUTC = workflowMeta.CreationTimestamp.Time.UTC().Format(time.RFC3339)
	}
	if scheduleTimeEpochSeconds == "" {
		return createTimeUTC, resolvePipelineJobScheduleTimeUTCFromWorkflow(workflowMeta, createTimeUTC), nil
	}
	scheduleTimeEpoch, err := strconv.ParseInt(scheduleTimeEpochSeconds, 10, 64)
	if err != nil {
		return "", "", fmt.Errorf("invalid pipeline job schedule time epoch seconds %q; check the run's scheduled_at: %w", scheduleTimeEpochSeconds, err)
	}
	return createTimeUTC, time.Unix(scheduleTimeEpoch, 0).UTC().Format(time.RFC3339), nil
}
