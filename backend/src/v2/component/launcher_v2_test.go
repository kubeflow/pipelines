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
	"errors"
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"go.uber.org/mock/gomock"
	"io"
	"k8s.io/client-go/kubernetes"
	"os"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
	"google.golang.org/protobuf/types/known/structpb"
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

var testExecutorInputJSON = `{
  "inputs": {
    "artifacts": {
      "list1": {
        "artifacts": [
          {
            "name": "runtime artifact 1",
            "uri": "oci://runtime-artifact-1"
          }
        ]
      }
    }
  }
}
`

var helloCmdArgs = []string{"sh", "-c", "echo \"hello world\""}

var happyLauncherV2Deps = LauncherV2Dependencies{
	k8sClientProvider: func() (kubernetes.Interface, error) {
		return &fake.Clientset{}, nil
	},
	metadataClientProvider: func(address string, port string) (metadata.ClientInterface, error) {
		return metadata.NewFakeClient(), nil
	},
	cacheClientProvider: func() (*cacheutils.Client, error) {
		return &cacheutils.Client{}, nil
	},
}

var validOpts = LauncherV2Options{
	Namespace:         "my-namespace",
	PodName:           "my-pod",
	PodUID:            "abcd",
	MLMDServerAddress: "example.com",
	MLMDServerPort:    "1234",
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
			)

			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)

			}
		})
	}
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

// Tests happy and unhappy paths for constructing a new LauncherV2
func Test_NewLauncherV2(t *testing.T) {
	type args struct {
		executionID       int64
		executorInputJSON string
		componentSpecJSON string
		cmdArgs           []string
		opts              LauncherV2Options
		deps              LauncherV2Dependencies
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
				cmdArgs:           helloCmdArgs,
				opts:              validOpts,
				deps:              happyLauncherV2Deps,
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
				cmdArgs:           helloCmdArgs,
				opts:              LauncherV2Options{},
			},
			expectedErr: errors.New("invalid launcher options: must specify Namespace"),
		},
		{
			name: "k8s client error",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{}",
				cmdArgs:           helloCmdArgs,
				opts:              validOpts,
				deps: (func() LauncherV2Dependencies {
					myDeps := happyLauncherV2Deps
					myDeps.k8sClientProvider = func() (kubernetes.Interface, error) {
						return nil, errors.New("k8s client error")
					}
					return myDeps
				})(),
			},
			expectedErr: errors.New("k8s client error"),
		},
		{
			name: "metadata client error",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{}",
				cmdArgs:           helloCmdArgs,
				opts:              validOpts,
				deps: (func() LauncherV2Dependencies {
					myDeps := happyLauncherV2Deps
					myDeps.metadataClientProvider = func(address string, port string) (metadata.ClientInterface, error) {
						return nil, errors.New("metadata client error")
					}
					return myDeps
				})(),
			},
			expectedErr: errors.New("metadata client error"),
		},
		{
			name: "cache client error",
			args: &args{
				executionID:       1,
				executorInputJSON: "{}",
				componentSpecJSON: "{}",
				cmdArgs:           helloCmdArgs,
				opts:              validOpts,
				deps: (func() LauncherV2Dependencies {
					myDeps := happyLauncherV2Deps
					myDeps.cacheClientProvider = func() (*cacheutils.Client, error) {
						return nil, errors.New("cache client error")
					}
					return myDeps
				})(),
			},
			expectedErr: errors.New("cache client error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.args
			_, err := NewLauncherV2(context.TODO(), args.executionID, args.executorInputJSON, args.componentSpecJSON, args.cmdArgs, &args.opts, &args.deps)
			if test.expectedErr != nil {
				assert.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_DefaultLauncherV2Dependencies(t *testing.T) {
	defaultDeps := DefaultLauncherV2Dependencies()
	assert.IsType(t, defaultK8sClientProvider, defaultDeps.k8sClientProvider)
	assert.IsType(t, defaultMetadataClientProvider, defaultDeps.metadataClientProvider)
	assert.IsType(t, defaultCacheClientProvider, defaultDeps.cacheClientProvider)
}

func Test_LauncherV2_Execute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validOpts := &LauncherV2Options{
		Namespace:         "my-namespace",
		PodName:           "my-pod",
		PodUID:            "abcd",
		MLMDServerAddress: "example.com",
		MLMDServerPort:    "1234",
	}
	testDeps := LauncherV2Dependencies{
		k8sClientProvider: func() (kubernetes.Interface, error) {
			return &fake.Clientset{}, nil
		},
		metadataClientProvider: func(address string, port string) (metadata.ClientInterface, error) {
			fakeClient := metadata.NewFakeClient()
			pipeline, _ := fakeClient.GetPipelineFromExecution(context.TODO(), int64(1))
			execution, _ := fakeClient.CreateExecution(context.TODO(), pipeline, &metadata.ExecutionConfig{})
			mockMetadataClient := metadata.NewMockClientInterface(ctrl)
			mockMetadataClient.
				EXPECT().
				GetExecution(gomock.Any(), int64(1)).
				Return(execution, nil)
			mockMetadataClient.
				EXPECT().
				PrePublishExecution(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(execution, nil)
			mockMetadataClient.
				EXPECT().
				PublishExecution(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)
			mockMetadataClient.
				EXPECT().
				GetDAG(gomock.Any(), gomock.Any()).
				Return(&metadata.DAG{}, nil)
			mockMetadataClient.
				EXPECT().
				GetPipelineFromExecution(gomock.Any(), gomock.Any()).
				Return(pipeline, nil)
			mockMetadataClient.
				EXPECT().
				UpdateDAGExecutionsState(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)
			return mockMetadataClient, nil
		},
		cacheClientProvider: func() (*cacheutils.Client, error) {
			mockSvc := cacheutils.NewMockTaskServiceClient(ctrl)
			mockSvc.
				EXPECT().
				CreateTaskV1(gomock.Any(), gomock.Any()).
				Return(nil, nil)
			return cacheutils.NewCustomClient(mockSvc), nil
		},
		executionFunc: func(
			ctx context.Context,
			executorInput *pipelinespec.ExecutorInput,
			component *pipelinespec.ComponentSpec,
			cmd string,
			args []string,
			bucket *blob.Bucket,
			bucketConfig *objectstore.Config,
			metadataClient metadata.ClientInterface,
			namespace string,
			k8sClient kubernetes.Interface,
			publishLogs string) (*pipelinespec.ExecutorOutput, []*metadata.OutputArtifact, error) {
			return &pipelinespec.ExecutorOutput{}, nil, nil
		},
		createFileFunc: func(s string) (*os.File, error) {
			return nil, nil
		},
	}
	type args struct {
		executionID       int64
		executorInputJSON string
		componentSpecJSON string
		cmdArgs           []string
		opts              *LauncherV2Options
		deps              LauncherV2Dependencies
	}
	tests := []struct {
		name        string
		args        *args
		expectedErr error
	}{
		{
			name: "happy path",
			args: &args{
				executionID: 1,
				// executorInputJSON: "{}",
				executorInputJSON: testExecutorInputJSON,
				componentSpecJSON: "{}",
				cmdArgs:           helloCmdArgs,
				opts:              validOpts,
				deps:              testDeps,
			},
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.args
			launcherV2, err := NewLauncherV2(context.TODO(), args.executionID, args.executorInputJSON, args.componentSpecJSON, args.cmdArgs, args.opts, &args.deps)
			assert.NoError(t, err)
			err = launcherV2.Execute(context.TODO())
			assert.NoError(t, err)
		})
	}
}

func Test_stopWaitingArtifacts(t *testing.T) {
	artifacts := map[string]*pipelinespec.ArtifactList{
		"list1": &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					Name: "runtime artifact 1",
					Uri:  "oci://runtime-artifact-1",
				},
			},
		},
		"list2": &pipelinespec.ArtifactList{
			Artifacts: make([]*pipelinespec.RuntimeArtifact, 0),
		},
		"list3": &pipelinespec.ArtifactList{
			Artifacts: []*pipelinespec.RuntimeArtifact{
				{
					Name: "runtime artifact 2",
					Uri:  "runtime-artifact-2",
				},
			},
		},
	}
	tests := []struct {
		name               string
		createFileProvider CreateFileFunc
		artifacts          map[string]*pipelinespec.ArtifactList
		expectedErrs       []error
	}{
		{
			name: "happy path",
			createFileProvider: func(s string) (*os.File, error) {
				return nil, nil
			},
			expectedErrs: make([]error, 0),
		},
		{
			name: "create file error",
			createFileProvider: func(s string) (*os.File, error) {
				return nil, errors.New("create file error")
			},
			expectedErrs: []error{errors.New("create file error")},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := stopWaitingArtifacts(artifacts, test.createFileProvider)
			assert.Equal(t, test.expectedErrs, errs)
		})
	}
}

func Test_LauncherV2_Info(t *testing.T) {
	validOpts := &LauncherV2Options{
		Namespace:         "my-namespace",
		PodName:           "my-pod",
		PodUID:            "abcd",
		MLMDServerAddress: "example.com",
		MLMDServerPort:    "1234",
	}
	deps := LauncherV2Dependencies{
		k8sClientProvider: func() (kubernetes.Interface, error) {
			return &fake.Clientset{}, nil
		},
		metadataClientProvider: func(address string, port string) (metadata.ClientInterface, error) {
			return metadata.NewFakeClient(), nil
		},
		cacheClientProvider: func() (*cacheutils.Client, error) {
			return &cacheutils.Client{}, nil
		},
	}

	launcher, err := NewLauncherV2(context.TODO(), int64(1), testExecutorInputJSON, "{}", helloCmdArgs, validOpts, &deps)
	assert.NoError(t, err)
	res := launcher.Info()
	assert.Equal(t, "launcher info:\nexecutorInput="+testExecutorInputJSON, res)
}

func Test_LauncherV2Options_validate(t *testing.T) {
	tests := []struct {
		name        string
		opts        LauncherV2Options
		expectedErr error
	}{
		{
			name:        "happy path",
			opts:        validOpts,
			expectedErr: nil,
		},
		{
			name: "missing namespace",
			opts: (func() LauncherV2Options {
				myOpts := validOpts
				myOpts.Namespace = ""
				return myOpts
			})(),
			expectedErr: errors.New("invalid launcher options: must specify Namespace"),
		},
		{
			name: "missing pod name",
			opts: (func() LauncherV2Options {
				myOpts := validOpts
				myOpts.PodName = ""
				return myOpts
			})(),
			expectedErr: errors.New("invalid launcher options: must specify PodName"),
		},
		{
			name: "missing pod uid",
			opts: (func() LauncherV2Options {
				myOpts := validOpts
				myOpts.PodUID = ""
				return myOpts
			})(),
			expectedErr: errors.New("invalid launcher options: must specify PodUID"),
		},
		{
			name: "missing mlmdserver address",
			opts: (func() LauncherV2Options {
				myOpts := validOpts
				myOpts.MLMDServerAddress = ""
				return myOpts
			})(),
			expectedErr: errors.New("invalid launcher options: must specify MLMDServerAddress"),
		},
		{
			name: "missing mlmdserver port",
			opts: (func() LauncherV2Options {
				myOpts := validOpts
				myOpts.MLMDServerPort = ""
				return myOpts
			})(),
			expectedErr: errors.New("invalid launcher options: must specify MLMDServerPort"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.opts.validate()
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func Test_collectOutputParameters(t *testing.T) {
	osReadFileFuncOrig := osReadFileFunc
	defer func() { osReadFileFunc = osReadFileFuncOrig }()

	happyReadFileFunc := func(filename string) ([]byte, error) {
		return []byte("My output parameter value!"), nil
	}

	var testExecutorInput = &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
				"myOutputParameter1": &pipelinespec.ExecutorInput_OutputParameter{
					OutputFile: "/path/to/output1",
				},
				"myOutputParameter2": &pipelinespec.ExecutorInput_OutputParameter{
					OutputFile: "/path/to/output2",
				},
			},
		},
	}

	tests := []struct {
		name           string
		executorInput  *pipelinespec.ExecutorInput
		executorOutput *pipelinespec.ExecutorOutput
		readFileFunc   func(filename string) ([]byte, error)
		expectedErr    error
	}{
		{
			name:          "happy path",
			executorInput: testExecutorInput,
			executorOutput: &pipelinespec.ExecutorOutput{
				ParameterValues: map[string]*structpb.Value{
					"myOutputParameter1": &structpb.Value{
						Kind: &structpb.Value_StringValue{
							StringValue: "My output parameter value!",
						},
					},
				},
			},
			readFileFunc: happyReadFileFunc,
			expectedErr:  nil,
		},
		{
			name:           "ParameterValues is nil",
			executorInput:  testExecutorInput,
			executorOutput: &pipelinespec.ExecutorOutput{},
			readFileFunc:   happyReadFileFunc,
			expectedErr:    nil,
		},
		{
			name: "component spec missing parameter",
			executorInput: &pipelinespec.ExecutorInput{
				Outputs: &pipelinespec.ExecutorInput_Outputs{
					Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
						"myMissingOutputParameter": &pipelinespec.ExecutorInput_OutputParameter{
							OutputFile: "/path/to/missingOutput",
						},
					},
				},
			},
			executorOutput: &pipelinespec.ExecutorOutput{},
			readFileFunc:   happyReadFileFunc,
			expectedErr:    errors.New("failed to find output parameter name=\"myMissingOutputParameter\" in component spec"),
		},
		{
			name: "failed to read file",
			executorInput: func() *pipelinespec.ExecutorInput {
				myExecutorInput := &pipelinespec.ExecutorInput{
					Outputs: &pipelinespec.ExecutorInput_Outputs{
						Parameters: map[string]*pipelinespec.ExecutorInput_OutputParameter{
							"myOutputParameter1": testExecutorInput.Outputs.Parameters["myOutputParameter1"],
						},
					},
				}
				return myExecutorInput
			}(),
			executorOutput: &pipelinespec.ExecutorOutput{},
			readFileFunc: func(filename string) ([]byte, error) {
				return nil, errors.New("failed to read file")
			},
			expectedErr: errors.New("failed to read output parameter name=\"myOutputParameter1\" type=\"STRING\" path=\"/path/to/output1\": failed to read file"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			osReadFileFunc = test.readFileFunc
			componentSpec := &pipelinespec.ComponentSpec{
				OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
					Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{
						"myOutputParameter1": &pipelinespec.ComponentOutputsSpec_ParameterSpec{
							ParameterType: pipelinespec.ParameterType_STRING,
						},
						"myOutputParameter2": &pipelinespec.ComponentOutputsSpec_ParameterSpec{
							ParameterType: pipelinespec.ParameterType_STRING,
						},
					},
				},
			}
			err := collectOutputParameters(test.executorInput, test.executorOutput, componentSpec)
			if test.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expectedErr.Error())
			}
		})
	}
}

func Test_uploadOutputArtifacts(t *testing.T) {
	testExecutorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"list1": &pipelinespec.ArtifactList{
					Artifacts: []*pipelinespec.RuntimeArtifact{
						{
							Name: "myArtifact",
							Uri:  "oci://path/to/myArtifact",
							Metadata: &structpb.Struct{
								Fields: make(map[string]*structpb.Value),
							},
							Type: &pipelinespec.ArtifactTypeSchema{
								Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{
									SchemaTitle: "MySchemaTitle",
								},
							},
						},
					},
				},
			},
		},
	}

	testExecutorOutput := &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"list1": &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{
					{
						Name: "myArtifact",
						Uri:  "oci://path/to/myArtifact",
						Metadata: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"myField": &structpb.Value{
									Kind: &structpb.Value_StringValue{
										StringValue: "My metadata value!",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	opts := uploadOutputArtifactsOptions{
		metadataClient: &metadata.FakeClient{},
	}

	tests := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "happy path",
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifacts, err := uploadOutputArtifacts(context.TODO(), testExecutorInput, testExecutorOutput, opts)
			assert.Equal(t, test.expectedErr, err)
			assert.NotEmpty(t, artifacts)
		})
	}
}
