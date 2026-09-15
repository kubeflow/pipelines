// Copyright 2025 The Kubeflow Authors
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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	commonmlflow "github.com/kubeflow/pipelines/backend/src/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins"
	"github.com/kubeflow/pipelines/backend/src/v2/common/plugins/mlflow"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
)

func Test_validateContainer(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil container spec returns error",
			opts: Options{
				Container: nil,
			},
			wantErr: true,
			errMsg:  "container spec is required",
		},
		{
			name: "missing pipeline name returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				PipelineName: "",
			},
			wantErr: true,
			errMsg:  "pipeline name is required",
		},
		{
			name: "missing run ID returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				PipelineName: "pipeline-1",
				RunID:        "",
			},
			wantErr: true,
			errMsg:  "KFP run ID is required",
		},
		{
			name: "missing component spec returns error",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
				PipelineName: "pipeline-1",
				RunID:        "run-1",
				Component:    nil,
			},
			wantErr: true,
			errMsg:  "component spec is required",
		},
		{
			name: "valid container options pass validation",
			opts: Options{
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image: "test-image",
				},
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
			err := validateContainer(test.opts)
			if test.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// MockMetadataClient manually mocks the gRPC service.
type MockMetadataClient struct {
	pb.MetadataStoreServiceClient

	GetArtifactsByIDFunc           func(ctx context.Context, in *pb.GetArtifactsByIDRequest, opts ...grpc.CallOption) (*pb.GetArtifactsByIDResponse, error)
	GetEventsByExecutionIDsFunc    func(ctx context.Context, in *pb.GetEventsByExecutionIDsRequest, opts ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error)
	GetContextsByExecutionFunc     func(ctx context.Context, in *pb.GetContextsByExecutionRequest, opts ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error)
	GetContextTypeFunc             func(ctx context.Context, in *pb.GetContextTypeRequest, opts ...grpc.CallOption) (*pb.GetContextTypeResponse, error)
	PutParentContextsFunc          func(ctx context.Context, in *pb.PutParentContextsRequest, opts ...grpc.CallOption) (*pb.PutParentContextsResponse, error)
	GetParentContextsByContextFunc func(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error)
	GetContextByTypeAndNameFunc    func(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error)
	GetExecutionsByIDFunc          func(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error)
	PutExecutionFunc               func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error)
	PutExecutionTypeFunc           func(ctx context.Context, in *pb.PutExecutionTypeRequest, opts ...grpc.CallOption) (*pb.PutExecutionTypeResponse, error)
	GetExecutionsByTypeAndNameFunc func(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error)
	GetExecutionByTypeAndNameFunc  func(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error)
}

func (m *MockMetadataClient) GetArtifactsByID(ctx context.Context, in *pb.GetArtifactsByIDRequest, opts ...grpc.CallOption) (*pb.GetArtifactsByIDResponse, error) {
	if m.GetArtifactsByIDFunc != nil {
		return m.GetArtifactsByIDFunc(ctx, in, opts...)
	}
	return &pb.GetArtifactsByIDResponse{}, nil
}

func (m *MockMetadataClient) GetExecutionByTypeAndName(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
	if m.GetExecutionByTypeAndNameFunc != nil {
		return m.GetExecutionByTypeAndNameFunc(ctx, in, opts...)
	}
	return &pb.GetExecutionByTypeAndNameResponse{}, nil
}

func (m *MockMetadataClient) PutExecutionType(ctx context.Context, in *pb.PutExecutionTypeRequest, opts ...grpc.CallOption) (*pb.PutExecutionTypeResponse, error) {
	if m.PutExecutionTypeFunc != nil {
		return m.PutExecutionTypeFunc(ctx, in, opts...)
	}
	return &pb.PutExecutionTypeResponse{}, nil
}

func (m *MockMetadataClient) GetEventsByExecutionIDs(ctx context.Context, in *pb.GetEventsByExecutionIDsRequest, opts ...grpc.CallOption) (*pb.GetEventsByExecutionIDsResponse, error) {
	if m.GetEventsByExecutionIDsFunc != nil {
		return m.GetEventsByExecutionIDsFunc(ctx, in, opts...)
	}
	return &pb.GetEventsByExecutionIDsResponse{}, nil
}

func (m *MockMetadataClient) GetContextsByExecution(ctx context.Context, in *pb.GetContextsByExecutionRequest, opts ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
	if m.GetContextsByExecutionFunc != nil {
		return m.GetContextsByExecutionFunc(ctx, in, opts...)
	}
	return &pb.GetContextsByExecutionResponse{}, nil
}

func (m *MockMetadataClient) GetContextType(ctx context.Context, in *pb.GetContextTypeRequest, opts ...grpc.CallOption) (*pb.GetContextTypeResponse, error) {
	if m.GetContextTypeFunc != nil {
		return m.GetContextTypeFunc(ctx, in, opts...)
	}
	return &pb.GetContextTypeResponse{}, nil
}

func (m *MockMetadataClient) PutParentContexts(ctx context.Context, in *pb.PutParentContextsRequest, opts ...grpc.CallOption) (*pb.PutParentContextsResponse, error) {
	if m.PutParentContextsFunc != nil {
		return m.PutParentContextsFunc(ctx, in, opts...)
	}
	return &pb.PutParentContextsResponse{}, nil
}

func (m *MockMetadataClient) GetParentContextsByContext(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error) {
	if m.GetParentContextsByContextFunc != nil {
		return m.GetParentContextsByContextFunc(ctx, in, opts...)
	}
	return &pb.GetParentContextsByContextResponse{}, nil
}

func (m *MockMetadataClient) GetContextByTypeAndName(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
	if m.GetContextByTypeAndNameFunc != nil {
		return m.GetContextByTypeAndNameFunc(ctx, in, opts...)
	}
	// Return a safe default to prevent nil pointer panics in your real GetPipeline method
	return &pb.GetContextByTypeAndNameResponse{}, nil
}

func (m *MockMetadataClient) GetExecutionsByID(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
	if m.GetExecutionsByIDFunc != nil {
		return m.GetExecutionsByIDFunc(ctx, in, opts...)
	}
	return &pb.GetExecutionsByIDResponse{}, nil
}

func (m *MockMetadataClient) PutExecution(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
	if m.PutExecutionFunc != nil {
		return m.PutExecutionFunc(ctx, in, opts...)
	}
	return &pb.PutExecutionResponse{}, nil
}

func (m *MockMetadataClient) GetExecutionsByTypeAndName(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
	if m.GetExecutionsByTypeAndNameFunc != nil {
		return m.GetExecutionsByTypeAndNameFunc(ctx, in, opts...)
	}
	return &pb.GetExecutionByTypeAndNameResponse{}, nil
}

func TestContainer_CreateExecutionGeneralFailure(t *testing.T) {
	mockSvc := &MockMetadataClient{
		GetParentContextsByContextFunc: func(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error) {
			return &pb.GetParentContextsByContextResponse{}, nil
		},
		GetContextByTypeAndNameFunc: func(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
			return &pb.GetContextByTypeAndNameResponse{Context: &pb.Context{Id: new(int64(1234))}}, nil
		},
		GetExecutionsByIDFunc: func(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{Id: func() *int64 { i := int64(55); return &i }()}}}, nil
		},

		// Trigger a general error (e.g., Internal)
		PutExecutionFunc: func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			return nil, status.Error(codes.Internal, "database connection failed")
		},
	}

	mlmdClient := metadata.NewTestClient(mockSvc)

	execution, err := Container(context.Background(), Options{
		IterationIndex: -1,
		PipelineName:   "pipeline-1",
		RunID:          "run-1",
		TaskName:       "task-1",
		Component: &pipelinespec.ComponentSpec{
			Implementation:   &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "executor"},
			InputDefinitions: &pipelinespec.ComponentInputsSpec{Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{}},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{"output": {ParameterType: pipelinespec.ParameterType_STRING}},
			},
		},
		DAGExecutionID: 55,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "task-1"},
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true},
		},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
			Image:   "python:3.11",
			Command: []string{"python", "main.py"},
		},
		PluginDispatcher: plugins.NoOpDispatcher{},
	}, mlmdClient, &mockCacheClient{})

	require.NotNil(t, execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	assert.NotContains(t, err.Error(), "failed to lookup existing execution")
}

func TestContainer_CreateExecutionSuccess(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	for _, test := range []struct {
		name           string
		dagID          int64
		taskName       string
		iterationIndex int
		executionName  string
	}{
		{"non-iteration", 55, "task-1", -1, "task/55/task-1/-1"},
		{"different DAG", 56, "task-1", -1, "task/56/task-1/-1"},
		{"different task", 55, "task-2", -1, "task/55/task-2/-1"},
		{"first iteration", 55, "task-1", 0, "task/55/task-1/0"},
		{"second iteration", 55, "task-1", 1, "task/55/task-1/1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mockSvc := &MockMetadataClient{
				GetParentContextsByContextFunc: func(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error) {
					return &pb.GetParentContextsByContextResponse{}, nil
				},
				GetContextByTypeAndNameFunc: func(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
					return &pb.GetContextByTypeAndNameResponse{Context: &pb.Context{Id: new(int64(1234))}}, nil
				},
				GetExecutionsByIDFunc: func(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
					inputs, err := structpb.NewStruct(map[string]any{"item": []any{"first", "second"}})
					require.NoError(t, err)
					return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{
						Id: &in.ExecutionIds[0],
						CustomProperties: map[string]*pb.Value{
							"inputs": {Value: &pb.Value_StructValue{StructValue: inputs}},
						},
					}}}, nil
				},
				PutExecutionFunc: func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
					require.Equal(t, test.executionName, in.GetExecution().GetName())
					return nil, status.Error(codes.AlreadyExists, "execution already exists")
				},
				GetExecutionByTypeAndNameFunc: func(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
					require.Equal(t, test.executionName, in.GetExecutionName())
					require.Equal(t, string(metadata.ContainerExecutionTypeName), in.GetTypeName())
					return &pb.GetExecutionByTypeAndNameResponse{
						Execution: &pb.Execution{
							Id: new(int64(1234)),
						},
					}, nil
				},
			}

			mlmdClient := metadata.NewTestClient(mockSvc)

			opts := Options{
				IterationIndex: test.iterationIndex,
				PipelineName:   "pipeline-1",
				RunID:          "run-1",
				TaskName:       test.taskName,
				Component: &pipelinespec.ComponentSpec{
					Implementation:   &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "executor"},
					InputDefinitions: &pipelinespec.ComponentInputsSpec{Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{}},
					OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
						Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{"output": {ParameterType: pipelinespec.ParameterType_STRING}},
					},
				},
				DAGExecutionID: test.dagID,
				Task: &pipelinespec.PipelineTaskSpec{
					TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: test.taskName},
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true},
				},
				Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
					Image:   "python:3.11",
					Command: []string{"python", "main.py"},
				},
				PluginDispatcher: plugins.NoOpDispatcher{},
			}

			if test.iterationIndex >= 0 {
				opts.Task.Iterator = &pipelinespec.PipelineTaskSpec_ParameterIterator{
					ParameterIterator: &pipelinespec.ParameterIteratorSpec{ItemInput: "item"},
				}
			}
			for attempt := 0; attempt < 2; attempt++ {
				execution, err := Container(context.Background(), opts, mlmdClient, &mockCacheClient{})
				require.NoError(t, err)
				require.NotNil(t, execution)
				assert.Equal(t, int64(1234), execution.ID)
				require.NotNil(t, execution.Cached)
				assert.False(t, *execution.Cached)
				assert.NotEmpty(t, execution.PodSpecPatch)
			}
		})
	}
}

func TestContainer_CreateExecutionAlreadyExistsLookupReturnsNil(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	mockSvc := &MockMetadataClient{
		GetParentContextsByContextFunc: func(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error) {
			return &pb.GetParentContextsByContextResponse{}, nil
		},
		GetContextByTypeAndNameFunc: func(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
			return &pb.GetContextByTypeAndNameResponse{Context: &pb.Context{Id: new(int64(1234))}}, nil
		},
		GetExecutionsByIDFunc: func(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{Id: func() *int64 { i := int64(55); return &i }()}}}, nil
		},

		// Trigger the AlreadyExists path
		PutExecutionFunc: func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "execution already exists")
		},

		// Return a successful lookup, but with NO executions (translates to nil existing execution)
		GetExecutionByTypeAndNameFunc: func(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
			return nil, status.Error(codes.Internal, "simulated gRPC lookup failure")
		},
	}

	mlmdClient := metadata.NewTestClient(mockSvc)

	execution, err := Container(context.Background(), Options{
		IterationIndex: -1,
		PipelineName:   "pipeline-1",
		RunID:          "run-1",
		TaskName:       "task-1",
		Component: &pipelinespec.ComponentSpec{
			Implementation:   &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "executor"},
			InputDefinitions: &pipelinespec.ComponentInputsSpec{Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{}},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{"output": {ParameterType: pipelinespec.ParameterType_STRING}},
			},
		},
		DAGExecutionID: 55,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "task-1"},
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true},
		},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
			Image:   "python:3.11",
			Command: []string{"python", "main.py"},
		},
		PluginDispatcher: plugins.NoOpDispatcher{},
	}, mlmdClient, &mockCacheClient{})

	require.NotNil(t, execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to lookup existing execution")
	assert.Contains(t, err.Error(), "simulated gRPC lookup failure")
}

func TestContainer_CreateExecutionDoesNotExistGenericError(t *testing.T) {
	mockSvc := &MockMetadataClient{
		GetParentContextsByContextFunc: func(ctx context.Context, in *pb.GetParentContextsByContextRequest, opts ...grpc.CallOption) (*pb.GetParentContextsByContextResponse, error) {
			return &pb.GetParentContextsByContextResponse{}, nil
		},
		GetContextByTypeAndNameFunc: func(ctx context.Context, in *pb.GetContextByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
			return &pb.GetContextByTypeAndNameResponse{Context: &pb.Context{Id: new(int64(1234))}}, nil
		},
		GetExecutionsByIDFunc: func(ctx context.Context, in *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{Id: func() *int64 { i := int64(55); return &i }()}}}, nil
		},

		// Trigger an error that is NOT AlreadyExists
		PutExecutionFunc: func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			return nil, status.Error(codes.Unavailable, "unavailable")
		},

		// Return a valid execution to simulate finding it successfully
		GetExecutionByTypeAndNameFunc: func(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
			return &pb.GetExecutionByTypeAndNameResponse{
				Execution: &pb.Execution{
					Id: new(int64(999)),
				},
			}, nil
		},
	}

	mlmdClient := metadata.NewTestClient(mockSvc)

	execution, err := Container(context.Background(), Options{
		IterationIndex: -1,
		PipelineName:   "pipeline-1",
		RunID:          "run-1",
		TaskName:       "task-1",
		Component: &pipelinespec.ComponentSpec{
			Implementation:   &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "executor"},
			InputDefinitions: &pipelinespec.ComponentInputsSpec{Parameters: map[string]*pipelinespec.ComponentInputsSpec_ParameterSpec{}},
			OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
				Parameters: map[string]*pipelinespec.ComponentOutputsSpec_ParameterSpec{"output": {ParameterType: pipelinespec.ParameterType_STRING}},
			},
		},
		DAGExecutionID: 55,
		Task: &pipelinespec.PipelineTaskSpec{
			TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "task-1"},
			CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true},
		},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{
			Image:   "python:3.11",
			Command: []string{"python", "main.py"},
		},
		PluginDispatcher: plugins.NoOpDispatcher{},
	}, mlmdClient, &mockCacheClient{})

	// In a successful recovery, we expect NO error to be returned from Container
	require.Error(t, err)
	require.NotNil(t, execution)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestContainer_ReusedExecutionRestoresMLflow(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	for _, test := range []struct {
		name              string
		sameAttempt       bool
		cached            bool
		startFails        bool
		cleanupFails      bool
		missingProperties bool
	}{
		{name: "driver restart"},
		{name: "cached completion", cached: true},
		{name: "failed repeated start", cached: true, startFails: true},
		{name: "failed redundant run cleanup", cleanupFails: true},
		{name: "RPC retry keeps current run", sameAttempt: true},
		{name: "missing persisted state", missingProperties: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mutex sync.Mutex
			starts := 0
			updates := map[string]string{}
			var loggedRuns []string
			var canceledRuns []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mutex.Lock()
				defer mutex.Unlock()
				switch r.URL.Path {
				case "/api/2.0/mlflow/runs/create":
					starts++
					if test.startFails && starts == 2 {
						http.Error(w, "start failed", http.StatusInternalServerError)
						return
					}
					fmt.Fprintf(w, `{"run":{"info":{"run_id":"run-%d"}}}`, starts)
				case "/api/2.0/mlflow/runs/update":
					var update struct {
						RunID  string `json:"run_id"`
						Status string `json:"status"`
					}
					if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					updates[update.RunID] = update.Status
					if update.Status == "KILLED" {
						canceledRuns = append(canceledRuns, update.RunID)
					}
					if test.cleanupFails && update.RunID == "run-2" {
						http.Error(w, "cleanup failed", http.StatusInternalServerError)
						return
					}
					fmt.Fprint(w, `{}`)
				case "/api/2.0/mlflow/runs/log-batch":
					var batch commonmlflow.LogBatchRequest
					if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					loggedRuns = append(loggedRuns, batch.RunID)
					fmt.Fprint(w, `{}`)
				default:
					t.Errorf("unexpected MLflow request: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			newDispatcher := func() plugins.TaskPluginDispatcher {
				handler, err := mlflow.NewMLflowTaskHandler(&commonmlflow.MLflowRuntimeConfig{
					Endpoint: server.URL, ParentRunID: "parent-run", ExperimentID: "experiment",
					AuthType: commonmlflow.AuthTypeNone, Timeout: "5s", InjectUserEnvVars: true,
				})
				require.NoError(t, err)
				dispatcher, err := plugins.NewTaskPluginDispatcherImpl([]plugins.TaskPluginHandler{handler})
				require.NoError(t, err)
				return dispatcher
			}

			var stored *pb.Execution
			failRead := !test.sameAttempt
			svc := &MockMetadataClient{
				GetContextByTypeAndNameFunc: func(context.Context, *pb.GetContextByTypeAndNameRequest, ...grpc.CallOption) (*pb.GetContextByTypeAndNameResponse, error) {
					return &pb.GetContextByTypeAndNameResponse{Context: &pb.Context{Id: proto.Int64(10)}}, nil
				},
				PutExecutionFunc: func(_ context.Context, req *pb.PutExecutionRequest, _ ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
					if req.GetExecution().GetId() != 0 {
						stored = proto.Clone(req.GetExecution()).(*pb.Execution)
						return &pb.PutExecutionResponse{ExecutionId: stored.Id}, nil
					}
					if stored != nil {
						require.Equal(t, stored.GetName(), req.GetExecution().GetName())
						return nil, status.Error(codes.AlreadyExists, "execution already exists")
					}
					stored = proto.Clone(req.GetExecution()).(*pb.Execution)
					stored.Id = proto.Int64(100)
					if test.sameAttempt {
						// Model a committed RPC whose response was lost and whose retry collided.
						return nil, status.Error(codes.AlreadyExists, "execution already exists")
					}
					return &pb.PutExecutionResponse{ExecutionId: stored.Id}, nil
				},
				GetExecutionsByIDFunc: func(_ context.Context, req *pb.GetExecutionsByIDRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
					id := req.ExecutionIds[0]
					if id == 100 {
						if failRead {
							failRead = false
							return nil, status.Error(codes.Unavailable, "read failed after commit")
						}
						return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{proto.Clone(stored).(*pb.Execution)}}, nil
					}
					return &pb.GetExecutionsByIDResponse{Executions: []*pb.Execution{{Id: &id}}}, nil
				},
				GetExecutionByTypeAndNameFunc: func(_ context.Context, req *pb.GetExecutionByTypeAndNameRequest, _ ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
					require.Equal(t, stored.GetName(), req.GetExecutionName())
					return &pb.GetExecutionByTypeAndNameResponse{Execution: proto.Clone(stored).(*pb.Execution)}, nil
				},
			}
			client := metadata.NewTestClient(svc)
			cache := &mockCacheClient{}
			if test.cached {
				cache.getExecutionCacheFunc = func(string, string, string) (string, error) { return "999", nil }
			}
			opts := Options{
				PipelineName: "pipeline", RunID: "run", TaskName: "notify", DAGExecutionID: 55, IterationIndex: -1,
				Component: &pipelinespec.ComponentSpec{},
				Task: &pipelinespec.PipelineTaskSpec{
					TaskInfo:       &pipelinespec.PipelineTaskInfo{Name: "notify"},
					CachingOptions: &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: test.cached},
				},
				Container:        &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11", Command: []string{"python"}},
				PluginDispatcher: newDispatcher(),
			}
			ctx := context.Background()
			if !test.sameAttempt {
				_, err := Container(ctx, opts, client, cache)
				require.ErrorContains(t, err, "read failed after commit")
				// A restarted driver has fresh handlers; persisted execution state must win.
				opts.PluginDispatcher = newDispatcher()
				if test.missingProperties {
					delete(stored.CustomProperties, "plugins.mlflow.run_id")
				}
			}
			execution, err := Container(ctx, opts, client, cache)
			if test.missingProperties {
				require.ErrorContains(t, err, "persisted plugin properties are missing")
				require.Empty(t, execution.PodSpecPatch)
				mutex.Lock()
				defer mutex.Unlock()
				assert.Equal(t, []string{"run-2"}, canceledRuns)
				return
			}
			require.NoError(t, err)
			require.Equal(t, int64(100), execution.ID)
			require.Equal(t, "run-1", stored.GetCustomProperties()["plugins.mlflow.run_id"].GetStringValue())
			if test.cached {
				require.True(t, *execution.Cached)
			} else {
				var pod corev1.PodSpec
				require.NoError(t, json.Unmarshal([]byte(execution.PodSpecPatch), &pod))
				require.NotEmpty(t, pod.Containers)
				assert.Contains(t, pod.Containers[0].Env, corev1.EnvVar{Name: "MLFLOW_RUN_ID", Value: "run-1"})
				launcher := newDispatcher()
				launcher.ApplyCustomProperties(metadata.ExtractPluginCustomProperties(metadata.NewExecution(stored)))
				require.NoError(t, launcher.OnTaskEnd(ctx, &plugins.TaskInfo{Name: "notify", RunStatus: "COMPLETE", ScalarMetrics: map[string]float64{"accuracy": 1}}))
			}
			mutex.Lock()
			defer mutex.Unlock()
			assert.Equal(t, "FINISHED", updates["run-1"])
			if !test.cached {
				assert.NotEmpty(t, loggedRuns)
			}
			for _, runID := range loggedRuns {
				assert.Equal(t, "run-1", runID)
			}
			if test.sameAttempt {
				assert.Equal(t, 1, starts)
				assert.Len(t, updates, 1)
				assert.Empty(t, canceledRuns)
			} else {
				assert.Equal(t, 2, starts)
				if test.startFails {
					assert.NotContains(t, updates, "run-2")
					assert.Empty(t, canceledRuns)
				} else {
					assert.Equal(t, "KILLED", updates["run-2"])
					for _, runID := range canceledRuns {
						assert.Equal(t, "run-2", runID)
					}
				}
			}
		})
	}
}
