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

package common

import (
	"os"
	"reflect"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestCreateArtifactPath(t *testing.T) {
	tests := []struct {
		name         string
		runID        string
		nodeID       string
		artifactName string
		expected     string
	}{
		{
			name:         "typical values",
			runID:        "run-123",
			nodeID:       "node-456",
			artifactName: "model-output",
			expected:     "runs/run-123/nodes/node-456/artifacts/model-output",
		},
		{
			name:         "empty strings",
			runID:        "",
			nodeID:       "",
			artifactName: "",
			expected:     "runs//nodes//artifacts/",
		},
		{
			name:         "uuid style values",
			runID:        "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			nodeID:       "train-model-abc",
			artifactName: "metrics",
			expected:     "runs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/nodes/train-model-abc/artifacts/metrics",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := CreateArtifactPath(testCase.runID, testCase.nodeID, testCase.artifactName)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      string
		expectedLevel zapcore.Level
		expectError   bool
	}{
		{
			name:          "debug level",
			logLevel:      "debug",
			expectedLevel: zapcore.DebugLevel,
			expectError:   false,
		},
		{
			name:          "info level",
			logLevel:      "info",
			expectedLevel: zapcore.InfoLevel,
			expectError:   false,
		},
		{
			name:          "warn level",
			logLevel:      "warn",
			expectedLevel: zapcore.WarnLevel,
			expectError:   false,
		},
		{
			name:          "error level",
			logLevel:      "error",
			expectedLevel: zapcore.ErrorLevel,
			expectError:   false,
		},
		{
			name:          "panic level",
			logLevel:      "panic",
			expectedLevel: zapcore.PanicLevel,
			expectError:   false,
		},
		{
			name:          "fatal level",
			logLevel:      "fatal",
			expectedLevel: zapcore.FatalLevel,
			expectError:   false,
		},
		{
			name:        "invalid level returns error",
			logLevel:    "invalid",
			expectError: true,
		},
		{
			name:        "empty string returns error",
			logLevel:    "",
			expectError: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			level, err := ParseLogLevel(testCase.logLevel)
			if testCase.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "could not translate log level to ZAP levels")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedLevel, level)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	t.Run("existing file returns true", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "test-file-exists-*")
		assert.NoError(t, err)
		defer os.Remove(tempFile.Name())
		tempFile.Close()

		assert.True(t, FileExists(tempFile.Name()))
	})

	t.Run("non-existent path returns false", func(t *testing.T) {
		assert.False(t, FileExists("/tmp/non-existent-file-that-does-not-exist-12345"))
	})

	t.Run("existing directory returns true", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-dir-exists-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		assert.True(t, FileExists(tempDir))
	})
}

func TestCustomMarshaler(t *testing.T) {
	marshaler := CustomMarshaler()

	assert.NotNil(t, marshaler)
	assert.True(t, marshaler.UseProtoNames)
	assert.False(t, marshaler.EmitUnpopulated)
	assert.False(t, marshaler.DiscardUnknown)
}

func TestPatchPipelineDefaultParameter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		bucket   string
		project  string
		expected string
	}{
		{
			name:     "replaces both placeholders",
			input:    "gs://{{kfp-default-bucket}}/{{kfp-project-id}}/output",
			bucket:   "my-bucket",
			project:  "my-project",
			expected: "gs://my-bucket/my-project/output",
		},
		{
			name:     "text with no placeholders is unchanged",
			input:    "gs://some-bucket/some-project/output",
			bucket:   "my-bucket",
			project:  "my-project",
			expected: "gs://some-bucket/some-project/output",
		},
		{
			name:     "replaces only bucket placeholder",
			input:    "gs://{{kfp-default-bucket}}/data",
			bucket:   "test-bucket",
			project:  "test-project",
			expected: "gs://test-bucket/data",
		},
		{
			name:     "replaces multiple occurrences",
			input:    "{{kfp-default-bucket}} and {{kfp-default-bucket}} with {{kfp-project-id}}",
			bucket:   "b1",
			project:  "p1",
			expected: "b1 and b1 with p1",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			viper.Reset()
			t.Setenv(DefaultBucketNameEnvVar, testCase.bucket)
			t.Setenv(ProjectIDEnvVar, testCase.project)
			viper.AutomaticEnv()

			result, err := PatchPipelineDefaultParameter(testCase.input)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestParseResourceIdsFromFullName(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			"namespace-experiment-run",
			args{"namespaces/default/experiments/Default/runs/run-one"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "run-one",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"namespace-experiment-job-run",
			args{"namespaces/default/experiments/Default/jobs/j1/runs/run-one"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "run-one",
				"RecurringRunId":    "j1",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"empty-namespace-experiment-run",
			args{"namespaces//experiments/Default/runs/run-one"},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "run-one",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"experiment-run",
			args{"experiments/Default/runs/run-one"},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "run-one",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"namespace-pipeline-version",
			args{"namespaces/default/pipelines/p1/versions/pv1"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "",
				"PipelineId":        "p1",
				"PipelineVersionId": "pv1",
				"RunID":             "",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"namespace-experiment-job-exec-artifact",
			args{"namespaces/default/experiments/Default/jobs/j2/executions/e1/artifacts/a1"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "",
				"RecurringRunId":    "j2",
				"ArtifactId":        "a1",
				"ExecutionId":       "e1",
			},
		},
		{
			"everything",
			args{"namespaces/default/experiments/Default/pipelines/p1/versions/pv1/jobs/j2/runs/r1/executions/e1/artifacts/a1"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "p1",
				"PipelineVersionId": "pv1",
				"RunID":             "r1",
				"RecurringRunId":    "j2",
				"ArtifactId":        "a1",
				"ExecutionId":       "e1",
			},
		},
		{
			"empty",
			args{""},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
		{
			"everything starts and ends with /",
			args{"///namespaces/default/experiments/Default/pipelines/p1/versions/pv1/jobs/j2/runs/r1/executions/e1/artifacts/a1//"},
			map[string]string{
				"Namespace":         "default",
				"ExperimentId":      "Default",
				"PipelineId":        "p1",
				"PipelineVersionId": "pv1",
				"RunID":             "r1",
				"RecurringRunId":    "j2",
				"ArtifactId":        "a1",
				"ExecutionId":       "e1",
			},
		},
		{
			"wrong keys",
			args{"///foo/bar/fiat/lux"},
			map[string]string{
				"Namespace":         "",
				"ExperimentId":      "",
				"PipelineId":        "",
				"PipelineVersionId": "",
				"RunID":             "",
				"RecurringRunId":    "",
				"ArtifactId":        "",
				"ExecutionId":       "",
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := ParseResourceIdsFromFullName(testCase.args.p); !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("ParseResourceIdsFromFullName() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func Test_validatePipelineName(t *testing.T) {
	tests := []struct {
		name         string
		pipelineName string
		wantErr      bool
		errMsg       string
	}{
		{
			"Valid - starts with letter",
			"pipeline-name123",
			false,
			"",
		},
		{
			"Valid - starts with number",
			"2nd-valid-nam3",
			false,
			"",
		},
		{
			"Valid - ends with dash",
			"pipeline-name-",
			false,
			"",
		},
		{
			"Invalid - too long",
			"more than  128 characters more than  128 characters more than  128 characters more than  128 characters more than  128 characters",
			true,
			"pipeline's name must contain no more than 128 characters",
		},
		{
			"Invalid - empty",
			"",
			true,
			"pipeline's name cannot be empty",
		},
		{
			"Invalid - starts with a dash",
			"-pipeline-name",
			true,
			"pipeline's name must contain only lowercase alphanumeric characters or '-' and must start with alphanumeric characters",
		},
		{
			"Invalid - has a capital letter",
			"pipeline-nAme",
			true,
			"pipeline's name must contain only lowercase alphanumeric characters or '-' and must start with alphanumeric characters",
		},
		{
			"Invalid - has a special character",
			"pipeline-$name",
			true,
			"pipeline's name must contain only lowercase alphanumeric characters or '-' and must start with alphanumeric characters",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if err := ValidatePipelineName(testCase.pipelineName); testCase.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), testCase.errMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
