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

package main

import (
	"context"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func strPtr(s string) *string {
	return &s
}

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

func TestSpecParsing(t *testing.T) {
	tt := []struct {
		name     string
		input    *string
		expected *kubernetesplatform.KubernetesExecutorConfig
		wantErr  bool
	}{
		{
			"Valid - test kubecfg value parse.",
			strPtr("{\"imagePullSecret\":[{\"secret_name\":\"value1\"}]}"),
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "value1"},
				},
			},
			false,
		},
		{
			"Valid - test kubecfg value ignores unknown field.",
			strPtr("{\"imagePullSecret\":[{\"secret_name\":\"value1\"}], \"unknown_field\": \"something\"}"),
			&kubernetesplatform.KubernetesExecutorConfig{
				ImagePullSecret: []*kubernetesplatform.ImagePullSecret{
					{SecretName: "value1"},
				},
			},
			false,
		},
	}

	for _, tc := range tt {
		t.Logf("Running test case: %s", tc.name)
		cfg, err := parseExecConfigJSON(tc.input)
		assert.Equal(t, tc.wantErr, err != nil)
		assert.True(t, proto.Equal(tc.expected, cfg))
	}
}

func TestGetPipelineJobTimePlaceholderUsage(t *testing.T) {
	tests := []struct {
		name       string
		driverType string
		taskSpec   *pipelinespec.PipelineTaskSpec
		want       pipelineJobTimePlaceholderUsage
	}{
		{
			name:       "root dag skips placeholder lookup",
			driverType: RootDag,
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
			driverType: DAG,
		},
		{
			name:       "tracks create and schedule placeholder usage from task inputs",
			driverType: CONTAINER,
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
			want: pipelineJobTimePlaceholderUsage{
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
				getPipelineJobTimePlaceholderUsage(tc.driverType, tc.taskSpec),
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
			name:                    "uses workflow creation time when create time arg is absent",
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
			actualCreateTimeUTC, actualScheduleTimeUTC, err := resolvePipelineJobTimes(
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
		placeholderUsage         pipelineJobTimePlaceholderUsage
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
			name:             "skips lookup when create placeholder already has compiled value",
			placeholderUsage: pipelineJobTimePlaceholderUsage{needsCreateTime: true},
			createTimeUTC:    "2026-01-02T03:04:05Z",
			wantGetterCalls:  0,
		},
		{
			name:                     "skips lookup when schedule placeholder already has compiled value",
			placeholderUsage:         pipelineJobTimePlaceholderUsage{needsScheduleTime: true},
			scheduleTimeEpochSeconds: "1767225600",
			wantGetterCalls:          0,
		},
		{
			name:             "returns workflow metadata when schedule time needs lookup",
			placeholderUsage: pipelineJobTimePlaceholderUsage{needsScheduleTime: true},
			createTimeUTC:    "2026-01-02T03:04:05Z",
			getterResult:     workflowMeta,
			wantMetadata:     workflowMeta,
			wantGetterCalls:  1,
		},
		{
			name:             "falls back when only schedule-time lookup fails",
			placeholderUsage: pipelineJobTimePlaceholderUsage{needsScheduleTime: true},
			createTimeUTC:    "2026-01-02T03:04:05Z",
			getterErr:        lookupErr,
			wantGetterCalls:  1,
		},
		{
			name:             "fails when create time still needs lookup",
			placeholderUsage: pipelineJobTimePlaceholderUsage{needsCreateTime: true},
			getterErr:        lookupErr,
			wantErr:          true,
			wantGetterCalls:  1,
		},
		{
			name:             "fails when schedule time needs lookup without create time fallback",
			placeholderUsage: pipelineJobTimePlaceholderUsage{needsScheduleTime: true},
			getterErr:        lookupErr,
			wantErr:          true,
			wantGetterCalls:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getterCalls := 0
			actualMetadata, err := getWorkflowMetadataForPipelineJobTimes(
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

func Test_extractOutputParametersContainer(t *testing.T) {
	execution := &driver.Execution{}

	outputs := extractOutputParameters(execution, CONTAINER)

	require.NotEmpty(t, outputs)
	verifyOutputParameter(t, outputs, "condition", "nil")
}

func Test_extractOutputParametersRootDAG(t *testing.T) {
	execution := &driver.Execution{}

	outputs := extractOutputParameters(execution, RootDag)
	require.NotEmpty(t, outputs)
	verifyOutputParameter(t, outputs, "iteration-count", "0")
	verifyOutputParameter(t, outputs, "condition", "nil")
}

func Test_extractOutputParametersDAG(t *testing.T) {
	execution := &driver.Execution{}

	outputs := extractOutputParameters(execution, DAG)

	require.NotEmpty(t, outputs)
	verifyOutputParameter(t, outputs, "iteration-count", "0")
	verifyOutputParameter(t, outputs, "condition", "nil")
}

func verifyOutputParameter(t *testing.T, parameters []driverapi.Parameter, key, expectedValue string) {
	filtered := make([]driverapi.Parameter, 0, 1)
	for _, p := range parameters {
		if p.Name == key {
			filtered = append(filtered, p)
		}
	}
	require.Len(t, filtered, 1)
	require.Equal(t, expectedValue, filtered[0].Value)
}
