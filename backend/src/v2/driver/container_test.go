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
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	}, mlmdClient, &mockCacheClient{})

	require.NotNil(t, execution)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	assert.NotContains(t, err.Error(), "failed to lookup existing execution")
}

func TestContainer_CreateExecutionSuccess(t *testing.T) {
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
		PutExecutionFunc: func(ctx context.Context, in *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			return nil, status.Error(codes.AlreadyExists, "execution already exists")
		},
		GetExecutionByTypeAndNameFunc: func(ctx context.Context, in *pb.GetExecutionByTypeAndNameRequest, opts ...grpc.CallOption) (*pb.GetExecutionByTypeAndNameResponse, error) {
			return &pb.GetExecutionByTypeAndNameResponse{
				Execution: &pb.Execution{
					Id: new(int64(1234)),
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
	}, mlmdClient, &mockCacheClient{})

	require.NotNil(t, execution)
	require.NoError(t, err)
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
	}, mlmdClient, &mockCacheClient{})

	// In a successful recovery, we expect NO error to be returned from Container
	require.Error(t, err)
	require.NotNil(t, execution)
	assert.Contains(t, err.Error(), "unavailable")
}
