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

package executorplugin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	pipelineJobCreateTimeUTCPlaceholder   = "{{$.pipeline_job_create_time_utc}}"
	pipelineJobScheduleTimeUTCPlaceholder = "{{$.pipeline_job_schedule_time_utc}}"
)

func getCurrentWorkflowMetadata(ctx context.Context, restConfig *rest.Config, namespace string, workflowName string) (*metav1.ObjectMeta, error) {
	if workflowName == "" {
		return nil, fmt.Errorf("workflow name is empty; set run_name to the owning Workflow name")
	}
	argoClient, err := argoclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize argo client for workflow metadata; check the Kubernetes API endpoint and TLS settings: %w", err)
	}
	workflow, err := argoClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, workflowName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow %q; check that it exists and the plugin service account has workflows/get permission: %w", workflowName, err)
	}
	return &workflow.ObjectMeta, nil
}

type workflowMetadataGetter func(ctx context.Context, namespace string, workflowName string) (*metav1.ObjectMeta, error)

type pipelineJobTimePlaceholderUsage struct {
	needsCreateTime   bool
	needsScheduleTime bool
}

func getPipelineJobTimePlaceholderUsage(
	driverType string,
	taskSpec *pipelinespec.PipelineTaskSpec,
) pipelineJobTimePlaceholderUsage {
	usage := pipelineJobTimePlaceholderUsage{}
	if driverType == rootDAG || taskSpec == nil {
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

func getWorkflowMetadataForPipelineJobTimes(
	ctx context.Context,
	namespace string,
	workflowName string,
	placeholderUsage pipelineJobTimePlaceholderUsage,
	createTimeUTC string,
	scheduleTimeEpochSeconds string,
	getMetadata workflowMetadataGetter,
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

func resolvePipelineJobTimes(
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
		return "", "", fmt.Errorf("invalid pipeline job schedule time epoch seconds %q: %w", scheduleTimeEpochSeconds, err)
	}
	return createTimeUTC, time.Unix(scheduleTimeEpoch, 0).UTC().Format(time.RFC3339), nil
}
