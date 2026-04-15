// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package driver

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
)

func TestNeedsWorkspaceMount(t *testing.T) {
	tests := []struct {
		name          string
		executorInput *pipelinespec.ExecutorInput
		expected      bool
	}{
		{
			name: "workspace path placeholder in parameters",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"workspace_path": {
							Kind: &structpb.Value_StringValue{
								StringValue: "{{$.workspace_path}}",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "workspace path prefix in parameters",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file_path": {
							Kind: &structpb.Value_StringValue{
								StringValue: "/kfp-workspace/data/file.txt",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "artifact with workspace metadata",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					Artifacts: map[string]*pipelinespec.ArtifactList{
						"model": {
							Artifacts: []*pipelinespec.RuntimeArtifact{
								{
									Metadata: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"_kfp_workspace": {
												Kind: &structpb.Value_BoolValue{
													BoolValue: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "workspace path with subdirectory",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"file_path": {
							Kind: &structpb.Value_StringValue{
								StringValue: "{{$.workspace_path}}/data",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no workspace usage",
			executorInput: &pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{
						"text": {
							Kind: &structpb.Value_StringValue{
								StringValue: "hello world",
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsWorkspaceMount(tt.executorInput)
			if result != tt.expected {
				t.Errorf("needsWorkspaceMount() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetWorkspaceMount(t *testing.T) {
	pvcName := "test-workflow-kfp-workspace"

	workspaceVolume, workspaceVolumeMount := getWorkspaceMount(pvcName)

	if workspaceVolume.Name != "kfp-workspace" {
		t.Errorf("Expected volume name kfp-workspace, got %s", workspaceVolume.Name)
	}

	if workspaceVolume.PersistentVolumeClaim == nil {
		t.Error("Expected PersistentVolumeClaim to be set")
	}

	if workspaceVolume.PersistentVolumeClaim.ClaimName != pvcName {
		t.Errorf("Expected claim name %s, got %s", pvcName, workspaceVolume.PersistentVolumeClaim.ClaimName)
	}

	if workspaceVolumeMount.Name != "kfp-workspace" {
		t.Errorf("Expected volume mount name kfp-workspace, got %s", workspaceVolumeMount.Name)
	}

	if workspaceVolumeMount.MountPath != "/kfp-workspace" {
		t.Errorf("Expected mount path /kfp-workspace, got %s", workspaceVolumeMount.MountPath)
	}
}

func TestValidateVolumeMounts(t *testing.T) {
	tests := []struct {
		name        string
		podSpec     *k8score.PodSpec
		expectError bool
	}{
		{
			name: "no conflicting volume mounts",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "data",
								MountPath: "/data",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "conflicting volume mount path",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "workspace",
								MountPath: "/kfp-workspace",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "conflicting volume mount subpath",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "data",
								MountPath: "/kfp-workspace/data",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "conflicting volume name kfp-workspace",
			podSpec: &k8score.PodSpec{
				Containers: []k8score.Container{
					{
						Name: "main",
						VolumeMounts: []k8score.VolumeMount{
							{
								Name:      "kfp-workspace",
								MountPath: "/data",
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumeMounts(tt.podSpec)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func Test_validateNonRoot(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing pipeline name returns error",
			opts: Options{
				PipelineName: "",
			},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing run ID returns error",
			opts: Options{
				PipelineName: "pipeline-1",
				RunID:        "",
			},
			wantErr: true,
			errMsg:  "KFP run ID is required",
		},
		{
			name: "nil component spec returns error",
			opts: Options{
				PipelineName: "pipeline-1",
				RunID:        "run-1",
				Component:    nil,
			},
			wantErr: true,
			errMsg:  "component spec is required",
		},
		{
			name: "missing task name returns error",
			opts: Options{
				PipelineName: "pipeline-1",
				RunID:        "run-1",
				Component:    &pipelinespec.ComponentSpec{},
				Task:         nil,
			},
			wantErr: true,
			errMsg:  "task spec is required",
		},
		{
			name: "runtime config present returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				Task:           &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-1"}},
				RuntimeConfig:  &pipelinespec.PipelineJob_RuntimeConfig{},
				DAGExecutionID: 1,
			},
			wantErr: true,
			errMsg:  "runtime config is unnecessary",
		},
		{
			name: "zero DAG execution ID returns error",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				Task:           &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-1"}},
				DAGExecutionID: 0,
			},
			wantErr: true,
			errMsg:  "DAG execution ID is required",
		},
		{
			name: "valid non-root options pass validation",
			opts: Options{
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				Component:      &pipelinespec.ComponentSpec{},
				Task:           &pipelinespec.PipelineTaskSpec{TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "task-1"}},
				DAGExecutionID: 1,
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateNonRoot(test.opts)
			if test.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_provisionOutputs(t *testing.T) {
	tests := []struct {
		name             string
		pipelineRoot     string
		taskName         string
		outputsSpec      *pipelinespec.ComponentOutputsSpec
		outputURISalt    string
		publishOutput    string
		wantArtifacts    []string
		wantParameters   []string
		wantLogsArtifact bool
		wantOutputFile   string
	}{
		{
			name:         "provisions output artifacts with URIs",
			pipelineRoot: "gs://my-bucket/pipeline-root",
			taskName:     "my-task",
			outputsSpec: &pipelinespec.ComponentOutputsSpec{
				Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
					"model": {
						ArtifactType: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
								SchemaTitle: "system.Model",
							},
						},
					},
				},
			},
			outputURISalt:    "salt-123",
			publishOutput:    "false",
			wantArtifacts:    []string{"model"},
			wantLogsArtifact: false,
			wantOutputFile:   "/gcs/my-bucket/pipeline-root/my-task/salt-123/output_metadata.json",
		},
		{
			name:         "provisions output parameters",
			pipelineRoot: "gs://my-bucket/pipeline-root",
			taskName:     "my-task",
			outputsSpec: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
					"accuracy": {ParameterType: pipelinespec.ParameterType_NUMBER_DOUBLE},
					"model_id": {ParameterType: pipelinespec.ParameterType_STRING},
				},
			},
			outputURISalt:  "salt-456",
			publishOutput:  "false",
			wantParameters: []string{"accuracy", "model_id"},
		},
		{
			name:         "publish logs adds executor-logs artifact",
			pipelineRoot: "gs://my-bucket/pipeline-root",
			taskName:     "my-task",
			outputsSpec: &pipelinespec.ComponentOutputsSpec{
				Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
					"output": {
						ArtifactType: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
								SchemaTitle: "system.Artifact",
							},
						},
					},
				},
			},
			outputURISalt:    "salt-789",
			publishOutput:    "true",
			wantArtifacts:    []string{"output", "executor-logs"},
			wantLogsArtifact: true,
		},
		{
			name:             "nil artifacts with publish logs only adds executor-logs",
			pipelineRoot:     "gs://my-bucket/pipeline-root",
			taskName:         "my-task",
			outputsSpec:      &pipelinespec.ComponentOutputsSpec{},
			outputURISalt:    "salt-000",
			publishOutput:    "true",
			wantArtifacts:    []string{"executor-logs"},
			wantLogsArtifact: true,
		},
		{
			name:          "empty outputs spec produces no artifacts or parameters",
			pipelineRoot:  "gs://my-bucket/pipeline-root",
			taskName:      "my-task",
			outputsSpec:   &pipelinespec.ComponentOutputsSpec{},
			outputURISalt: "salt-empty",
			publishOutput: "false",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputs := provisionOutputs(
				test.pipelineRoot,
				test.taskName,
				test.outputsSpec,
				test.outputURISalt,
				test.publishOutput,
			)
			assert.NotNil(t, outputs)
			assert.NotNil(t, outputs.Artifacts)
			assert.NotNil(t, outputs.Parameters)
			assert.NotEmpty(t, outputs.OutputFile)
			if test.wantOutputFile != "" {
				assert.Equal(t, test.wantOutputFile, outputs.OutputFile)
			}

			for _, artifactName := range test.wantArtifacts {
				artifactList, ok := outputs.Artifacts[artifactName]
				assert.True(t, ok, "expected artifact %q", artifactName)
				if ok {
					assert.Len(t, artifactList.Artifacts, 1)
					assert.NotEmpty(t, artifactList.Artifacts[0].Uri)
					assert.Contains(t, artifactList.Artifacts[0].Uri, test.pipelineRoot)
				}
			}

			for _, paramName := range test.wantParameters {
				param, ok := outputs.Parameters[paramName]
				assert.True(t, ok, "expected parameter %q", paramName)
				if ok {
					assert.Equal(t, fmt.Sprintf("/tmp/kfp/outputs/%s", paramName), param.OutputFile)
				}
			}

			if test.wantLogsArtifact {
				_, ok := outputs.Artifacts["executor-logs"]
				assert.True(t, ok, "expected executor-logs artifact when publishOutput is true")
			}

			if len(test.wantArtifacts) == 0 && !test.wantLogsArtifact {
				assert.Empty(t, outputs.Artifacts, "expected no artifacts for empty outputs spec")
			}
			if len(test.wantParameters) == 0 {
				assert.Empty(t, outputs.Parameters, "expected no parameters for empty outputs spec")
			}
		})
	}
}
