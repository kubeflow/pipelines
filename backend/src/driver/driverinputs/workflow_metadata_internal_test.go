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
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func runtimeValueConstant(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue(value),
				},
			},
		},
	}
}

func TestGetPipelineJobTimePlaceholderUsage(t *testing.T) {
	tests := []struct {
		name       string
		driverType driverapi.DriverType
		taskSpec   *pipelinespec.PipelineTaskSpec
		want       PlaceholderUsage
	}{
		{
			name:       "root dag skips placeholder lookup",
			driverType: driverapi.RootDAGDriver,
			taskSpec: &pipelinespec.PipelineTaskSpec{
				Inputs: &pipelinespec.TaskInputsSpec{
					Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
						"create_time":   runtimeValueConstant(pipelineJobCreateTimeUTCPlaceholder),
						"schedule_time": runtimeValueConstant(pipelineJobScheduleTimeUTCPlaceholder),
					},
				},
			},
		},
		{
			name:       "nil task spec has no placeholder usage",
			driverType: driverapi.DAGDriver,
		},
		{
			name:       "tracks create and schedule placeholder usage from task inputs",
			driverType: driverapi.ContainerDriver,
			taskSpec: &pipelinespec.PipelineTaskSpec{
				Inputs: &pipelinespec.TaskInputsSpec{
					Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
						"create_time":   runtimeValueConstant(pipelineJobCreateTimeUTCPlaceholder),
						"schedule_time": runtimeValueConstant(pipelineJobScheduleTimeUTCPlaceholder),
						"literal":       runtimeValueConstant("literal-value"),
						"component_input": {
							Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
								ComponentInputParameter: "pipeline-input",
							},
						},
					},
				},
			},
			want: PlaceholderUsage{
				needsCreateTime:   true,
				needsScheduleTime: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(
				t,
				tc.want,
				PipelineJobTimePlaceholderUsage(tc.driverType, tc.taskSpec),
			)
		})
	}
}

func TestResolvePipelineJobTimes(t *testing.T) {
	workflowCreationTime := metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	tt := []struct {
		name                     string
		createTimeUTC            string
		scheduleTimeEpochSeconds string
		workflowMeta             *metav1.ObjectMeta
		expectedCreateTimeUTC    string
		expectedScheduleTimeUTC  string
		wantErr                  bool
	}{
		{
			name:                    "falls back to create time when schedule label is absent",
			createTimeUTC:           "2026-01-02T03:04:05Z",
			expectedCreateTimeUTC:   "2026-01-02T03:04:05Z",
			expectedScheduleTimeUTC: "2026-01-02T03:04:05Z",
		},
		{
			name:                     "uses Run schedule time epoch when provided",
			createTimeUTC:            "2026-01-02T03:04:05Z",
			scheduleTimeEpochSeconds: "1767225600",
			expectedCreateTimeUTC:    "2026-01-02T03:04:05Z",
			expectedScheduleTimeUTC:  "2026-01-01T00:00:00Z",
		},
		{
			name:                    "uses workflow creation time when Run create time is absent",
			workflowMeta:            &metav1.ObjectMeta{CreationTimestamp: workflowCreationTime},
			expectedCreateTimeUTC:   "2026-01-02T03:04:05Z",
			expectedScheduleTimeUTC: "2026-01-02T03:04:05Z",
		},
		{
			name:          "uses workflow epoch label for schedule time when present",
			createTimeUTC: "2026-01-02T03:04:05Z",
			workflowMeta: &metav1.ObjectMeta{
				CreationTimestamp: workflowCreationTime,
				Labels: map[string]string{
					util.LabelKeyWorkflowEpoch: util.FormatInt64ForLabel(1767225600),
				},
			},
			expectedCreateTimeUTC:   "2026-01-02T03:04:05Z",
			expectedScheduleTimeUTC: "2026-01-01T00:00:00Z",
		},
		{
			name:          "falls back to workflow creation time when workflow epoch label is invalid",
			createTimeUTC: "2026-01-02T03:04:05Z",
			workflowMeta: &metav1.ObjectMeta{
				CreationTimestamp: workflowCreationTime,
				Labels: map[string]string{
					util.LabelKeyWorkflowEpoch: "invalid",
				},
			},
			expectedCreateTimeUTC:   "2026-01-02T03:04:05Z",
			expectedScheduleTimeUTC: "2026-01-02T03:04:05Z",
		},
		{
			name:                     "converts schedule epoch seconds to UTC",
			createTimeUTC:            "2026-01-02T03:04:05Z",
			scheduleTimeEpochSeconds: "1767225600",
			expectedCreateTimeUTC:    "2026-01-02T03:04:05Z",
			expectedScheduleTimeUTC:  "2026-01-01T00:00:00Z",
		},
		{
			name:                     "rejects invalid schedule epoch seconds",
			createTimeUTC:            "2026-01-02T03:04:05Z",
			scheduleTimeEpochSeconds: "not-an-int",
			wantErr:                  true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actualCreateTimeUTC, actualScheduleTimeUTC, err := ResolvePipelineJobTimes(
				tc.createTimeUTC,
				tc.scheduleTimeEpochSeconds,
				tc.workflowMeta,
			)
			assert.Equal(t, tc.wantErr, err != nil)
			if tc.wantErr {
				return
			}
			assert.Equal(t, tc.expectedCreateTimeUTC, actualCreateTimeUTC)
			assert.Equal(t, tc.expectedScheduleTimeUTC, actualScheduleTimeUTC)
		})
	}
}

func TestGetWorkflowMetadataForPipelineJobTimes(t *testing.T) {
	workflowMeta := &metav1.ObjectMeta{Name: "workflow-name"}
	lookupErr := assert.AnError

	tests := []struct {
		name                     string
		placeholderUsage         PlaceholderUsage
		createTimeUTC            string
		scheduleTimeEpochSeconds string
		getterResult             *metav1.ObjectMeta
		getterErr                error
		wantMetadata             *metav1.ObjectMeta
		wantErr                  bool
		wantGetterCalls          int
	}{
		{
			name:            "skips lookup when current task does not use placeholders",
			wantGetterCalls: 0,
		},
		{
			name:             "skips lookup when Run already provides create time",
			placeholderUsage: PlaceholderUsage{needsCreateTime: true},
			createTimeUTC:    "2026-01-02T03:04:05Z",
			wantGetterCalls:  0,
		},
		{
			name:                     "skips lookup when Run already provides schedule time",
			placeholderUsage:         PlaceholderUsage{needsScheduleTime: true},
			scheduleTimeEpochSeconds: "1767225600",
			wantGetterCalls:          0,
		},
		{
			name:             "returns workflow metadata when schedule time needs lookup",
			placeholderUsage: PlaceholderUsage{needsScheduleTime: true},
			createTimeUTC:    "2026-01-02T03:04:05Z",
			getterResult:     workflowMeta,
			wantMetadata:     workflowMeta,
			wantGetterCalls:  1,
		},
		{
			name:             "falls back when only schedule-time lookup fails",
			placeholderUsage: PlaceholderUsage{needsScheduleTime: true},
			createTimeUTC:    "2026-01-02T03:04:05Z",
			getterErr:        lookupErr,
			wantGetterCalls:  1,
		},
		{
			name:             "fails when create time still needs lookup",
			placeholderUsage: PlaceholderUsage{needsCreateTime: true},
			getterErr:        lookupErr,
			wantErr:          true,
			wantGetterCalls:  1,
		},
		{
			name:             "fails when schedule time needs lookup without create time fallback",
			placeholderUsage: PlaceholderUsage{needsScheduleTime: true},
			getterErr:        lookupErr,
			wantErr:          true,
			wantGetterCalls:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getterCalls := 0
			actualMetadata, err := WorkflowMetadataForPipelineJobTimes(
				context.Background(),
				"kubeflow",
				"workflow-name",
				tc.placeholderUsage,
				tc.createTimeUTC,
				tc.scheduleTimeEpochSeconds,
				func(ctx context.Context, namespace string, workflowName string) (*metav1.ObjectMeta, error) {
					getterCalls++
					assert.Equal(t, "kubeflow", namespace)
					assert.Equal(t, "workflow-name", workflowName)
					return tc.getterResult, tc.getterErr
				},
			)
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantGetterCalls, getterCalls)
			assert.Same(t, tc.wantMetadata, actualMetadata)
		})
	}
}
