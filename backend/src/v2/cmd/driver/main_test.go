package main

import (
	"os"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func strPtr(s string) *string {
	return &s
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
