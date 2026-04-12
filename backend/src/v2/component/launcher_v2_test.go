// Copyright 2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package component

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var addNumbersComponent = &pipelinespec.ComponentSpec{
	Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "add"},
	InputDefinitions: &pipelinespec.ComponentInputsSpec{
		Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{
			"a": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER, DefaultValue: structpb.NewNumberValue(5)},
			"b": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER},
		},
	},
	OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
		Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
			"Output": {ParameterType: pipelinespec.ParameterType_NUMBER_INTEGER},
		},
	},
}

// Tests that launcher correctly executes the user component and successfully writes output parameters to file.
func Test_executeV2_Parameters(t *testing.T) {
	tests := []struct {
		name          string
		executorInput *pipelinespec.ExecutorInput
		executorArgs  []string
		wantErr       bool
	}{
		{
			"happy pass",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"a": structpb.NewNumberValue(1), "b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "test {{$.inputs.parameters['a']}} -eq 1 || exit 1\ntest {{$.inputs.parameters['b']}} -eq 2 || exit 1"},
			false,
		},
		{
			"use default value",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "test {{$.inputs.parameters['a']}} -eq 5 || exit 1\ntest {{$.inputs.parameters['b']}} -eq 2 || exit 1"},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeKubernetesClientset := &fake.Clientset{}
			fakeMetadataClient := metadata.NewFakeClient()
			bucket, err := blob.OpenBucket(context.Background(), "mem://test-bucket")
			assert.Nil(t, err)
			bucketConfig, err := objectstore.ParseBucketConfig("mem://test-bucket/pipeline-root/", nil)
			assert.Nil(t, err)
			_, _, err = executeV2(
				context.Background(),
				test.executorInput,
				addNumbersComponent,
				"sh",
				test.executorArgs,
				bucket,
				bucketConfig,
				fakeMetadataClient,
				"namespace",
				fakeKubernetesClientset,
				"false",
				"",
				&OpenBucketConfig{context.Background(), fakeKubernetesClientset, "namespace", bucketConfig},
			)

			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)

			}
		})
	}
}

func Test_executeV2_publishLogs(t *testing.T) {
	tests := []struct {
		name          string
		executorInput *pipelinespec.ExecutorInput
		executorArgs  []string
		retryIndex    string
		wantErr       bool
	}{
		{
			"happy pass",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"a": structpb.NewNumberValue(1), "b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "echo testoutput && test {{$.inputs.parameters['a']}} -eq 1 || exit 1\ntest {{$.inputs.parameters['b']}} -eq 2 || exit 1"},
			"",
			false,
		},
		{
			"use default value",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "echo testoutput && test {{$.inputs.parameters['a']}} -eq 5 || exit 1\ntest {{$.inputs.parameters['b']}} -eq 2 || exit 1"},
			"",
			false,
		},
		{
			"sad fail",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"a": structpb.NewNumberValue(1), "b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "echo testoutput && exit 1"},
			"",
			true,
		},
		{
			// KFP_RETRY_INDEX is injected by the Argo compiler via "{{retries}}".
			// The executor-logs URI must be qualified with the retry index so each
			// attempt writes to a distinct, human-readable path (executor-logs-0,
			// executor-logs-1, …).
			"retry index qualifies executor-logs URI",
			&pipelinespec.ExecutorInput{
				Inputs: &pipelinespec.ExecutorInput_Inputs{
					ParameterValues: map[string]*structpb.Value{"a": structpb.NewNumberValue(1), "b": structpb.NewNumberValue(2)},
				},
			},
			[]string{"-c", "echo testoutput && test {{$.inputs.parameters['a']}} -eq 1 || exit 1"},
			"3",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeKubernetesClientset := &fake.Clientset{}
			fakeMetadataClient := metadata.NewFakeClient()
			bucket, err := blob.OpenBucket(context.Background(), "mem://test-bucket")
			assert.Nil(t, err)
			bucketConfig, err := objectstore.ParseBucketConfig("mem://test-bucket/pipeline-root/", nil)
			assert.Nil(t, err)
			// Add executor-logs artifact to outputs
			if test.executorInput.Outputs == nil {
				test.executorInput.Outputs = &pipelinespec.ExecutorInput_Outputs{}
			}
			if test.executorInput.Outputs.Artifacts == nil {
				test.executorInput.Outputs.Artifacts = make(map[string]*pipelinespec.ArtifactList)
			}
			// Use a temp directory for CustomPath to avoid writing to filesystem
			tempDir := t.TempDir()
			customPath := filepath.Join(tempDir, "executor-logs")
			test.executorInput.Outputs.Artifacts["executor-logs"] = &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{
					{
						Uri:        "mem://test-bucket/pipeline-root/executor-logs",
						Type:       &pipelinespec.ArtifactTypeSchema{Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Artifact"}},
						CustomPath: &customPath,
					},
				},
			}

			// Simulate Argo injecting KFP_RETRY_INDEX into the pod env.
			if test.retryIndex != "" {
				t.Setenv(EnvRetryIndex, test.retryIndex)
			}

			_, outputArtifacts, err := executeV2(
				context.Background(),
				test.executorInput,
				addNumbersComponent,
				"sh",
				test.executorArgs,
				bucket,
				bucketConfig,
				fakeMetadataClient,
				"namespace",
				fakeKubernetesClientset,
				"true",
				"",
				&OpenBucketConfig{context.Background(), fakeKubernetesClientset, "namespace", bucketConfig},
			)

			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			// When a retry index is set, the executor-logs URI (and therefore the
			// object-store key) must be suffixed with the index so retries don't
			// overwrite each other (e.g. executor-logs-3).
			effectiveIndex := test.retryIndex
			if effectiveIndex == "" {
				effectiveIndex = "0"
			}
			logKey := "executor-logs-" + effectiveIndex
			logArt := test.executorInput.Outputs.Artifacts["executor-logs"].Artifacts[0]
			assert.Contains(t, logArt.Uri, effectiveIndex,
				"executor-logs URI should contain the retry index for attempt isolation")

			outputLog, err := bucket.ReadAll(context.TODO(), logKey)
			assert.Nil(t, err, "Expected executor-logs to be readable at key %q", logKey)
			assert.Equal(t, "testoutput\n", string(outputLog))

			assert.Len(t, outputArtifacts, 1, "Expected 1 output artifact (executor-logs)")
		})
	}
}

func Test_getPlaceholders_WorkspaceArtifactPath(t *testing.T) {
	execIn := &pipelinespec.ExecutorInput{
		Inputs: &pipelinespec.ExecutorInput_Inputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"data": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{Uri: "minio://mlpipeline/sample/sample.txt", Metadata: &structpb.Struct{Fields: map[string]*structpb.Value{"_kfp_workspace": structpb.NewBoolValue(true)}}},
					},
				},
			},
		},
	}
	ph, err := getPlaceholders(execIn)
	if err != nil {
		t.Fatalf("getPlaceholders error: %v", err)
	}
	actual := ph["{{$.inputs.artifacts['data'].path}}"]
	expected := filepath.Join(WorkspaceMountPath, ".artifacts", "minio", "mlpipeline", "sample", "sample.txt")
	if actual != expected {
		t.Fatalf("placeholder path mismatch: actual=%q expected=%q", actual, expected)
	}
}

func Test_executorInput_compileCmdAndArgs(t *testing.T) {
	executorInputJSON := `{
		"inputs": {
			"parameterValues": {
				"config": {
					"category_ids": "{{$.inputs.parameters['pipelinechannel--category_ids']}}",
					"dump_filename": "{{$.inputs.parameters['pipelinechannel--dump_filename']}}",
					"sphinx_host": "{{$.inputs.parameters['pipelinechannel--sphinx_host']}}",
					"sphinx_port": "{{$.inputs.parameters['pipelinechannel--sphinx_port']}}"
				},
				"pipelinechannel--category_ids": "116",
				"pipelinechannel--dump_filename": "dump_filename_test.txt",
				"pipelinechannel--sphinx_host": "sphinx-default-host.ru",
				"pipelinechannel--sphinx_port": 9312
			}
		},
		"outputs": {
			"artifacts": {
				"dataset": {
					"artifacts": [{
						"type": {
							"schemaTitle": "system.Dataset",
							"schemaVersion": "0.0.1"
						},
						"uri": "s3://aviflow-stage-kfp-artifacts/debug-component-pipeline/ae02034e-bd96-4b8a-a06b-55c99fe9eccb/sayhello/c98ac032-2448-4637-bf37-3ad1e13a112c/dataset"
					}]
				}
			},
			"outputFile": "/tmp/kfp_outputs/output_metadata.json"
		}
	}`

	executorInput := &pipelinespec.ExecutorInput{}
	err := protojson.Unmarshal([]byte(executorInputJSON), executorInput)

	assert.NoError(t, err)

	cmd := "sh"
	args := []string{
		"--executor_input", "{{$}}",
		"--function_to_execute", "sayHello",
	}
	cmd, args, err = compileCmdAndArgs(executorInput, cmd, args)

	assert.NoError(t, err)

	var actualExecutorInput string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--executor_input" {
			actualExecutorInput = args[i+1]
			break
		}
	}
	assert.NotEmpty(t, actualExecutorInput, "--executor_input not found")

	var parsed map[string]any
	err = json.Unmarshal([]byte(actualExecutorInput), &parsed)
	assert.NoError(t, err)

	inputs := parsed["inputs"].(map[string]any)
	paramValues := inputs["parameterValues"].(map[string]any)
	config := paramValues["config"].(map[string]any)

	assert.Equal(t, "116", config["category_ids"])
	assert.Equal(t, "dump_filename_test.txt", config["dump_filename"])
	assert.Equal(t, "sphinx-default-host.ru", config["sphinx_host"])
	assert.Equal(t, "9312", config["sphinx_port"])
}

func Test_get_log_Writer(t *testing.T) {
	old := osCreateFunc
	defer func() { osCreateFunc = old }()

	osCreateFunc = func(name string) (*os.File, error) {
		tmpdir := t.TempDir()
		file, _ := os.CreateTemp(tmpdir, "*")
		return file, nil
	}

	tests := []struct {
		name        string
		artifacts   map[string]*pipelinespec.ArtifactList
		multiWriter bool
	}{
		{
			"single writer - no key logs",
			map[string]*pipelinespec.ArtifactList{
				"notLog": {},
			},
			false,
		},
		{
			"single writer - key log has empty list",
			map[string]*pipelinespec.ArtifactList{
				"logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{},
				},
			},
			false,
		},
		{
			"single writer - malformed uri",
			map[string]*pipelinespec.ArtifactList{
				"logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri: "",
						},
					},
				},
			},
			false,
		},
		{
			"multiwriter",
			map[string]*pipelinespec.ArtifactList{
				"executor-logs": {
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Uri: "minio://testinguri",
						},
					},
				},
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := getLogWriter(test.artifacts)
			if test.multiWriter == false {
				assert.Equal(t, os.Stdout, writer)
			} else {
				assert.IsType(t, io.MultiWriter(), writer)
			}
		})
	}
}

func Test_qualifyExecutorLogsURI(t *testing.T) {
	baseURI := "minio://mlpipeline/v2/artifacts/my-pipeline/run-id/always-fail/salt123/executor-logs"

	tests := []struct {
		name       string
		artifacts  map[string]*pipelinespec.ArtifactList
		retryIndex string
		wantURI    string
	}{
		{
			name: "appends retry index to executor-logs URI",
			artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: baseURI}}},
			},
			retryIndex: "2",
			wantURI:    baseURI + "-2",
		},
		{
			name: "no-op when retry index is empty",
			artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{{Uri: baseURI}}},
			},
			retryIndex: "",
			wantURI:    baseURI,
		},
		{
			name:       "no-op when executor-logs key is absent",
			artifacts:  map[string]*pipelinespec.ArtifactList{},
			retryIndex: "1",
			wantURI:    "", // no artifact to check
		},
		{
			name: "no-op when executor-logs list is empty",
			artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{}},
			},
			retryIndex: "1",
			wantURI:    "", // no artifact to check
		},
		{
			name: "no-op when executor-logs list has multiple artifacts",
			artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{
					{Uri: baseURI},
					{Uri: baseURI + "-2"},
				}},
			},
			retryIndex: "1",
			// list len != 1: guard should skip, original URIs unchanged
			wantURI: baseURI,
		},
		{
			name:       "no-op when ArtifactList value is nil",
			artifacts:  map[string]*pipelinespec.ArtifactList{"executor-logs": nil},
			retryIndex: "1",
			wantURI:    "", // nil list: no artifact to check
		},
		{
			name: "no-op when first artifact is nil",
			artifacts: map[string]*pipelinespec.ArtifactList{
				"executor-logs": {Artifacts: []*pipelinespec.RuntimeArtifact{nil}},
			},
			retryIndex: "1",
			wantURI:    "", // nil artifact: no URI to check
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				qualifyExecutorLogsURI(tc.artifacts, tc.retryIndex)
			})
			list, ok := tc.artifacts["executor-logs"]
			if !ok || list == nil || len(list.Artifacts) == 0 || list.Artifacts[0] == nil {
				// Cases where there is nothing to assert on
				return
			}
			assert.Equal(t, tc.wantURI, list.Artifacts[0].Uri)
		})
	}
}

func Test_retryIndexFromPodAnnotation(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		wantIndex  string
		wantErr    bool
	}{
		{
			name:       "parses first attempt (0)",
			annotation: "my-pipeline-abc.root.always-fail.executor(0)",
			wantIndex:  "0",
		},
		{
			name:       "parses fourth retry (4)",
			annotation: "retry-e2e-pzhkb.root.always-fail.executor(4)",
			wantIndex:  "4",
		},
		{
			name:       "no annotation",
			annotation: "",
			wantErr:    true,
		},
		{
			name:       "annotation without parenthesised suffix",
			annotation: "my-pipeline-abc.root.always-fail.executor",
			wantErr:    true,
		},
		{
			name:       "annotation with non-integer index",
			annotation: "my-pipeline-abc.root.always-fail.executor(abc)",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clientset := fake.NewClientset()
			if tc.annotation != "" {
				pod := &k8score.Pod{}
				pod.Name = "test-pod"
				pod.Namespace = "test-ns"
				pod.Annotations = map[string]string{
					"workflows.argoproj.io/node-name": tc.annotation,
				}
				_, err := clientset.CoreV1().Pods("test-ns").Create(context.Background(), pod, metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			idx, err := retryIndexFromPodAnnotation(context.Background(), clientset, "test-ns", "test-pod")
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantIndex, idx)
			}
		})
	}
}

// Tests happy and unhappy paths for constructing a new LauncherV2
func Test_NewLauncherV2(t *testing.T) {
	var testCmdArgs = []string{"sh", "-c", "echo \"hello world\""}

	disabledCacheClient, _ := cacheutils.NewClient("ml-pipeline.kubeflow", "8887", true, &tls.Config{})
	var testLauncherV2Deps = client_manager.NewFakeClientManager(
		fake.NewClientset(),
		metadata.NewFakeClient(),
		disabledCacheClient,
	)

	var testValidLauncherV2Opts = LauncherV2Options{
		Namespace:         "my-namespace",
		PodName:           "my-pod",
		PodUID:            "abcd",
		MLMDServerAddress: "example.com",
		MLMDServerPort:    "1234",
	}

	type args struct {
		executionID       int64
		executorInputJSON string
		componentSpecJSON string
		cmdArgs           []string
		opts              LauncherV2Options
		cm                client_manager.ClientManagerInterface
	}
	tests := []struct {
		name        string
		args        *args
		expectedErr error
	}{
		{
			name: "happy path",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{}",
				cmdArgs:           testCmdArgs,
				opts:              testValidLauncherV2Opts,
				cm:                testLauncherV2Deps,
			},
			expectedErr: nil,
		},
		{
			name: "missing executionID",
			args: &args{
				executionID: 0,
			},
			expectedErr: errors.New("must specify execution ID"),
		},
		{
			name: "invalid executorInput",
			args: &args{
				executionID:       1,
				executorInputJSON: "{",
			},
			expectedErr: errors.New("unexpected EOF"),
		},
		{
			name: "invalid componentSpec",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{",
			},
			expectedErr: errors.New("unexpected EOF\ncomponentSpec: {"),
		},
		{
			name: "missing cmdArgs",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{}",
				cmdArgs:           []string{},
			},
			expectedErr: errors.New("command and arguments are empty"),
		},
		{
			name: "invalid opts",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{}",
				cmdArgs:           testCmdArgs,
				opts:              LauncherV2Options{},
			},
			expectedErr: errors.New("invalid launcher options: must specify Namespace"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.args
			_, err := NewLauncherV2(context.Background(), args.executionID, args.executorInputJSON, args.componentSpecJSON, args.cmdArgs, &args.opts, args.cm)
			if test.expectedErr != nil {
				assert.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_retrieve_artifact_path(t *testing.T) {
	customPath := "/var/lib/kubelet/pods/pod-uid/volumes/kubernetes.io~csi/pvc-uuid/mount"
	tests := []struct {
		name         string
		artifact     *pipelinespec.RuntimeArtifact
		expectedPath string
	}{
		{
			"Artifact with no custom path",
			&pipelinespec.RuntimeArtifact{
				Uri: "gs://bucket/path/to/artifact",
			},
			"/gcs/bucket/path/to/artifact",
		},
		{
			"Artifact with custom path",
			&pipelinespec.RuntimeArtifact{
				Uri:        "gs://bucket/path/to/artifact",
				CustomPath: &customPath,
			},
			customPath,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, err := retrieveArtifactPath(test.artifact)
			assert.Nil(t, err)
			assert.Equal(t, path, test.expectedPath)
		})
	}
}

func Test_injectOCIArtifactMountPaths(t *testing.T) {
tests := []struct {
name            string
executorInput   *pipelinespec.ExecutorInput
wantCustomPaths map[string]string // artifactName → expected CustomPath
}{
{
name: "single OCI input artifact gets index-0 mount dir",
executorInput: &pipelinespec.ExecutorInput{
Inputs: &pipelinespec.ExecutorInput_Inputs{
Artifacts: map[string]*pipelinespec.ArtifactList{
"model": {
Artifacts: []*pipelinespec.RuntimeArtifact{
{Uri: "oci://registry.example.com/model:v1"},
},
},
},
},
},
wantCustomPaths: map[string]string{
"model": "/oci-artifact-0",
},
},
{
name: "two OCI input artifacts are sorted and indexed",
executorInput: &pipelinespec.ExecutorInput{
Inputs: &pipelinespec.ExecutorInput_Inputs{
Artifacts: map[string]*pipelinespec.ArtifactList{
"z_model": {
Artifacts: []*pipelinespec.RuntimeArtifact{
{Uri: "oci://registry.example.com/z:v1"},
},
},
"a_model": {
Artifacts: []*pipelinespec.RuntimeArtifact{
{Uri: "oci://registry.example.com/a:v1"},
},
},
},
},
},
// alphabetically: a_model < z_model
wantCustomPaths: map[string]string{
"a_model": "/oci-artifact-0",
"z_model": "/oci-artifact-1",
},
},
{
name: "non-OCI artifact is not modified",
executorInput: &pipelinespec.ExecutorInput{
Inputs: &pipelinespec.ExecutorInput_Inputs{
Artifacts: map[string]*pipelinespec.ArtifactList{
"dataset": {
Artifacts: []*pipelinespec.RuntimeArtifact{
{Uri: "gs://bucket/dataset"},
},
},
},
},
},
wantCustomPaths: map[string]string{
"dataset": "",
},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
injectOCIArtifactMountPaths(tt.executorInput)
for name, wantPath := range tt.wantCustomPaths {
art := tt.executorInput.GetInputs().GetArtifacts()[name].Artifacts[0]
if wantPath == "" {
assert.Nil(t, art.CustomPath, "artifact %q should have no CustomPath", name)
} else {
require.NotNil(t, art.CustomPath, "artifact %q should have CustomPath set", name)
assert.Equal(t, wantPath, *art.CustomPath, "artifact %q CustomPath", name)
}
}
})
}
}
