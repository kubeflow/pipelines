package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
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
		cfg, err := parseExecConfigJson(tc.input)
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
			driverType: ROOT_DAG,
			taskSpec: &pipelinespec.PipelineTaskSpec{
				Inputs: &pipelinespec.TaskInputsSpec{
					Parameters: map[string]*pipelinespec.TaskInputsSpec_InputParameterSpec{
						"create_time": runtimeValueConstant(pipelineJobCreateTimeUTCPlaceholder),
						"schedule_time": runtimeValueConstant(
							pipelineJobScheduleTimeUTCPlaceholder,
						),
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

func allProvided(flags []string) map[string]bool {
	provided := make(map[string]bool, len(flags))
	for _, name := range flags {
		provided[name] = true
	}
	return provided
}

func TestRequiredDriverFlags(t *testing.T) {
	common := []string{
		"type", "pipeline_name", "run_id", "run_name", "run_display_name",
		"component", "ml_pipeline_server_address", "ml_pipeline_server_port",
		"mlmd_server_address", "mlmd_server_port", "log_level", "publish_logs",
		"condition_path",
	}
	withCommon := func(extra ...string) []string {
		return append(append([]string{}, common...), extra...)
	}
	tests := []struct {
		driverType string
		want       []string
	}{
		{driverType: ROOT_DAG, want: withCommon("execution_id_path", "iteration_count_path", "runtime_config")},
		{driverType: DAG, want: withCommon("execution_id_path", "iteration_count_path", "task", "dag_execution_id", "iteration_index", "task_name")},
		{driverType: CONTAINER, want: withCommon("task", "dag_execution_id", "iteration_index", "task_name", "container", "kubernetes_config", "cached_decision_path", "pod_spec_patch_path")},
	}
	for _, tc := range tests {
		t.Run(tc.driverType, func(t *testing.T) {
			got, err := requiredDriverFlags(tc.driverType)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tc.want, got)
		})
	}

	_, err := requiredDriverFlags("UNKNOWN")
	assert.Error(t, err)
}

func TestValidateRequiredFlags(t *testing.T) {
	tests := []struct {
		name       string
		driverType string
		omit       []string
		wantErr    bool
	}{
		{
			name:       "ROOT_DAG with all required flags",
			driverType: ROOT_DAG,
		},
		{
			name:       "DAG with all required flags",
			driverType: DAG,
		},
		{
			name:       "CONTAINER with all required flags",
			driverType: CONTAINER,
		},
		{
			name:       "ROOT_DAG missing runtime_config",
			driverType: ROOT_DAG,
			omit:       []string{"runtime_config"},
			wantErr:    true,
		},
		{
			name:       "DAG missing dag_execution_id",
			driverType: DAG,
			omit:       []string{"dag_execution_id"},
			wantErr:    true,
		},
		{
			name:       "CONTAINER missing container",
			driverType: CONTAINER,
			omit:       []string{"container"},
			wantErr:    true,
		},
		{
			name:       "CONTAINER missing common flag run_id",
			driverType: CONTAINER,
			omit:       []string{"run_id"},
			wantErr:    true,
		},
		{
			name:       "DAG missing log_level",
			driverType: DAG,
			omit:       []string{"log_level"},
			wantErr:    true,
		},
		{
			name:       "CONTAINER missing publish_logs",
			driverType: CONTAINER,
			omit:       []string{"publish_logs"},
			wantErr:    true,
		},
		{
			name:       "ROOT_DAG missing execution_id_path",
			driverType: ROOT_DAG,
			omit:       []string{"execution_id_path"},
			wantErr:    true,
		},
		{
			name:       "DAG missing iteration_count_path",
			driverType: DAG,
			omit:       []string{"iteration_count_path"},
			wantErr:    true,
		},
		{
			name:       "CONTAINER missing pod_spec_patch_path",
			driverType: CONTAINER,
			omit:       []string{"pod_spec_patch_path"},
			wantErr:    true,
		},
		{
			name:       "CONTAINER missing cached_decision_path",
			driverType: CONTAINER,
			omit:       []string{"cached_decision_path"},
			wantErr:    true,
		},
		{
			name:       "DAG missing condition_path",
			driverType: DAG,
			omit:       []string{"condition_path"},
			wantErr:    true,
		},
		{
			name:       "unknown driver type",
			driverType: "UNKNOWN",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			required, err := requiredDriverFlags(tc.driverType)
			if err != nil {
				assert.True(t, tc.wantErr)
				assert.Error(t, validateRequiredFlags(map[string]bool{}, tc.driverType))
				return
			}
			provided := allProvided(required)
			for _, name := range tc.omit {
				delete(provided, name)
			}
			err = validateRequiredFlags(provided, tc.driverType)
			assert.Equal(t, tc.wantErr, err != nil, "unexpected error state: %v", err)
		})
	}
}

func Test_handleExecutionContainer(t *testing.T) {
	execution := &driver.Execution{}

	executionPaths := &ExecutionPaths{
		Condition: "condition.txt",
	}

	err := handleExecution(execution, CONTAINER, executionPaths)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	verifyFileContent(t, executionPaths.Condition, "nil")

	cleanup(t, executionPaths)
}

func Test_handleExecutionRootDAG(t *testing.T) {
	execution := &driver.Execution{}

	executionPaths := &ExecutionPaths{
		IterationCount: "iteration_count.txt",
		Condition:      "condition.txt",
	}

	err := handleExecution(execution, ROOT_DAG, executionPaths)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	verifyFileContent(t, executionPaths.IterationCount, "0")
	verifyFileContent(t, executionPaths.Condition, "nil")

	cleanup(t, executionPaths)
}

func Test_handleExecutionDAG(t *testing.T) {
	execution := &driver.Execution{}

	executionPaths := &ExecutionPaths{
		IterationCount: "iteration_count.txt",
		Condition:      "condition.txt",
	}

	err := handleExecution(execution, DAG, executionPaths)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	verifyFileContent(t, executionPaths.IterationCount, "0")
	verifyFileContent(t, executionPaths.Condition, "nil")

	cleanup(t, executionPaths)
}

func cleanup(t *testing.T, executionPaths *ExecutionPaths) {
	removeIfExists(t, executionPaths.IterationCount)
	removeIfExists(t, executionPaths.ExecutionID)
	removeIfExists(t, executionPaths.Condition)
	removeIfExists(t, executionPaths.PodSpecPatch)
	removeIfExists(t, executionPaths.CachedDecision)
}

func removeIfExists(t *testing.T, filePath string) {
	_, err := os.Stat(filePath)
	if err == nil {
		err = os.Remove(filePath)
		if err != nil {
			t.Errorf("Unexpected error while removing the created file: %v", err)
		}
	}
}

func verifyFileContent(t *testing.T, filePath string, expectedContent string) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created, but it doesn't exist", filePath)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read file contents: %v", err)
	}

	if string(fileContent) != expectedContent {
		t.Errorf("Expected file fileContent to be %q, got %q", expectedContent, string(fileContent))
	}
}
