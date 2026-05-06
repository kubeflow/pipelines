// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
)

// mockMLMDClient embeds the MetadataStoreServiceClient interface and overrides
// only the methods exercised by UpdateDAGExecutionsState's call chain.
// Calling any non-overridden method panics (nil receiver on the embedded
// interface), which is intentional — it surfaces unexpected calls immediately.
type mockMLMDClient struct {
	pb.MetadataStoreServiceClient

	getExecutionsByContextFn func(ctx context.Context, req *pb.GetExecutionsByContextRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByContextResponse, error)
	getExecutionsByIDFn      func(ctx context.Context, req *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error)
	getContextsByExecutionFn func(ctx context.Context, req *pb.GetContextsByExecutionRequest, opts ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error)
	getContextTypeFn         func(ctx context.Context, req *pb.GetContextTypeRequest, opts ...grpc.CallOption) (*pb.GetContextTypeResponse, error)
	putExecutionFn           func(ctx context.Context, req *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error)
}

func (m *mockMLMDClient) GetExecutionsByContext(ctx context.Context, req *pb.GetExecutionsByContextRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByContextResponse, error) {
	return m.getExecutionsByContextFn(ctx, req, opts...)
}

func (m *mockMLMDClient) GetExecutionsByID(ctx context.Context, req *pb.GetExecutionsByIDRequest, opts ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
	return m.getExecutionsByIDFn(ctx, req, opts...)
}

func (m *mockMLMDClient) GetContextsByExecution(ctx context.Context, req *pb.GetContextsByExecutionRequest, opts ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
	return m.getContextsByExecutionFn(ctx, req, opts...)
}

func (m *mockMLMDClient) GetContextType(ctx context.Context, req *pb.GetContextTypeRequest, opts ...grpc.CallOption) (*pb.GetContextTypeResponse, error) {
	return m.getContextTypeFn(ctx, req, opts...)
}

func (m *mockMLMDClient) PutExecution(ctx context.Context, req *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
	return m.putExecutionFn(ctx, req, opts...)
}

// ---------- helpers ----------

func int64Ptr(v int64) *int64 { return &v }

func makeExecution(id int64, state pb.Execution_State, customProperties map[string]*pb.Value) *pb.Execution {
	return &pb.Execution{
		Id:               int64Ptr(id),
		LastKnownState:   state.Enum(),
		CustomProperties: customProperties,
	}
}

func makeDAG(execution *pb.Execution) *DAG {
	return &DAG{Execution: &Execution{Execution: execution}}
}

// newTestClient creates a Client backed by the given mock gRPC service.
func newTestClient(mock pb.MetadataStoreServiceClient) *Client {
	return &Client{svc: mock}
}

// buildMock wires up a mockMLMDClient with an execution store (keyed by ID)
// and configurable PutExecution behavior.
func buildMock(
	executionStore map[int64]*pb.Execution,
	contextsByExecution map[int64][]*pb.Context,
	executionsByContext map[int64][]*pb.Execution,
	putExecutionFn func(ctx context.Context, req *pb.PutExecutionRequest, opts ...grpc.CallOption) (*pb.PutExecutionResponse, error),
) *mockMLMDClient {
	contextTypeID := int64(100)

	return &mockMLMDClient{
		getExecutionsByContextFn: func(_ context.Context, req *pb.GetExecutionsByContextRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByContextResponse, error) {
			contextID := req.GetContextId()
			return &pb.GetExecutionsByContextResponse{
				Executions: executionsByContext[contextID],
			}, nil
		},
		getExecutionsByIDFn: func(_ context.Context, req *pb.GetExecutionsByIDRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
			var results []*pb.Execution
			for _, id := range req.GetExecutionIds() {
				if e, ok := executionStore[id]; ok {
					results = append(results, e)
				}
			}
			return &pb.GetExecutionsByIDResponse{Executions: results}, nil
		},
		getContextsByExecutionFn: func(_ context.Context, req *pb.GetContextsByExecutionRequest, _ ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
			executionID := req.GetExecutionId()
			return &pb.GetContextsByExecutionResponse{
				Contexts: contextsByExecution[executionID],
			}, nil
		},
		getContextTypeFn: func(_ context.Context, _ *pb.GetContextTypeRequest, _ ...grpc.CallOption) (*pb.GetContextTypeResponse, error) {
			return &pb.GetContextTypeResponse{
				ContextType: &pb.ContextType{Id: &contextTypeID},
			}, nil
		},
		putExecutionFn: putExecutionFn,
	}
}

// defaultPutExecution applies the state change in-place and succeeds.
func defaultPutExecution(store map[int64]*pb.Execution) func(context.Context, *pb.PutExecutionRequest, ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
	return func(_ context.Context, req *pb.PutExecutionRequest, _ ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
		execution := req.GetExecution()
		if stored, ok := store[execution.GetId()]; ok {
			stored.LastKnownState = execution.LastKnownState
		}
		return &pb.PutExecutionResponse{}, nil
	}
}

// ---------- tests ----------

func TestUpdateDAGExecutionsState_RootDAGSkipped(t *testing.T) {
	// A root DAG has neither total_dag_tasks nor iteration_count set
	// (expectedTaskCount == 0). The function should return nil without
	// attempting any state write.
	rootExecution := makeExecution(1, pb.Execution_RUNNING, map[string]*pb.Value{})
	dag := makeDAG(rootExecution)

	store := map[int64]*pb.Execution{1: rootExecution}
	mock := buildMock(store, nil, map[int64][]*pb.Execution{}, nil)
	client := newTestClient(mock)

	err := client.UpdateDAGExecutionsState(context.Background(), dag, &Pipeline{})
	if err != nil {
		t.Fatalf("expected nil error for root DAG, got: %v", err)
	}
	if rootExecution.GetLastKnownState() != pb.Execution_RUNNING {
		t.Errorf("root DAG state should remain RUNNING, got %v", rootExecution.GetLastKnownState())
	}
}

func TestUpdateDAGExecutionsState_StateResolution(t *testing.T) {
	tests := []struct {
		name          string
		taskStates    []pb.Execution_State
		expectedTasks int64
		expectedState pb.Execution_State
	}{
		{
			name:          "all complete",
			taskStates:    []pb.Execution_State{pb.Execution_COMPLETE, pb.Execution_COMPLETE},
			expectedTasks: 2,
			expectedState: pb.Execution_COMPLETE,
		},
		{
			name:          "one failed",
			taskStates:    []pb.Execution_State{pb.Execution_COMPLETE, pb.Execution_FAILED},
			expectedTasks: 2,
			expectedState: pb.Execution_FAILED,
		},
		{
			name:          "still running",
			taskStates:    []pb.Execution_State{pb.Execution_COMPLETE, pb.Execution_RUNNING},
			expectedTasks: 2,
			expectedState: pb.Execution_RUNNING,
		},
		{
			name:          "cached and canceled count as complete",
			taskStates:    []pb.Execution_State{pb.Execution_COMPLETE, pb.Execution_CACHED, pb.Execution_CANCELED},
			expectedTasks: 3,
			expectedState: pb.Execution_COMPLETE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dagExecution := makeExecution(10, pb.Execution_RUNNING, map[string]*pb.Value{
				keyTotalDagTasks: intValue(tt.expectedTasks),
			})

			var tasks []*pb.Execution
			store := map[int64]*pb.Execution{10: dagExecution}
			for i, state := range tt.taskStates {
				id := int64(11 + i)
				task := makeExecution(id, state, map[string]*pb.Value{
					keyTaskName:    StringValue(fmt.Sprintf("task%d", i)),
					keyParentDagID: intValue(10),
				})
				tasks = append(tasks, task)
				store[id] = task
			}

			runCtxID := int64(500)
			pipeline := &Pipeline{pipelineRunCtx: &pb.Context{Id: &runCtxID}}
			dag := makeDAG(dagExecution)

			mock := buildMock(
				store,
				map[int64][]*pb.Context{10: {}},
				map[int64][]*pb.Execution{runCtxID: tasks},
				defaultPutExecution(store),
			)
			client := newTestClient(mock)

			err := client.UpdateDAGExecutionsState(context.Background(), dag, pipeline)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dagExecution.GetLastKnownState() != tt.expectedState {
				t.Errorf("expected DAG state %v, got %v", tt.expectedState, dagExecution.GetLastKnownState())
			}
		})
	}
}

func TestUpdateDAGExecutionsState_ExpectedTaskCount(t *testing.T) {
	tests := []struct {
		name          string
		dagProperties map[string]*pb.Value
		taskCount     int
		expectedState pb.Execution_State
	}{
		{
			name: "iteration_count takes priority over total_dag_tasks",
			dagProperties: map[string]*pb.Value{
				keyTotalDagTasks:  intValue(1),
				keyIterationCount: intValue(3),
			},
			taskCount:     3,
			expectedState: pb.Execution_COMPLETE,
		},
		{
			name: "iteration_count not met remains running",
			dagProperties: map[string]*pb.Value{
				keyTotalDagTasks:  intValue(1),
				keyIterationCount: intValue(3),
			},
			taskCount:     2,
			expectedState: pb.Execution_RUNNING,
		},
		{
			name: "iteration_count zero completes immediately",
			dagProperties: map[string]*pb.Value{
				keyTotalDagTasks:  intValue(0),
				keyIterationCount: intValue(0),
			},
			taskCount:     0,
			expectedState: pb.Execution_COMPLETE,
		},
		{
			name: "falls back to total_dag_tasks when no iteration_count",
			dagProperties: map[string]*pb.Value{
				keyTotalDagTasks: intValue(1),
			},
			taskCount:     1,
			expectedState: pb.Execution_COMPLETE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dagExecution := makeExecution(10, pb.Execution_RUNNING, tt.dagProperties)

			var tasks []*pb.Execution
			store := map[int64]*pb.Execution{10: dagExecution}
			for i := 0; i < tt.taskCount; i++ {
				id := int64(11 + i)
				task := makeExecution(id, pb.Execution_COMPLETE, map[string]*pb.Value{
					keyTaskName:    StringValue(fmt.Sprintf("iter-%d", i)),
					keyParentDagID: intValue(10),
				})
				tasks = append(tasks, task)
				store[id] = task
			}

			runCtxID := int64(500)
			pipeline := &Pipeline{pipelineRunCtx: &pb.Context{Id: &runCtxID}}
			dag := makeDAG(dagExecution)

			mock := buildMock(
				store,
				map[int64][]*pb.Context{10: {}},
				map[int64][]*pb.Execution{runCtxID: tasks},
				defaultPutExecution(store),
			)
			client := newTestClient(mock)

			err := client.UpdateDAGExecutionsState(context.Background(), dag, pipeline)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dagExecution.GetLastKnownState() != tt.expectedState {
				t.Errorf("expected DAG state %v, got %v", tt.expectedState, dagExecution.GetLastKnownState())
			}
		})
	}
}

func TestUpdateDAGExecutionsState_RecursivePropagation(t *testing.T) {
	tests := []struct {
		name          string
		leafState     pb.Execution_State
		expectedState pb.Execution_State
	}{
		{
			name:          "complete propagates up",
			leafState:     pb.Execution_COMPLETE,
			expectedState: pb.Execution_COMPLETE,
		},
		{
			name:          "failed propagates up",
			leafState:     pb.Execution_FAILED,
			expectedState: pb.Execution_FAILED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCtxID := int64(500)
			pipelineCtxTypeID := int64(100)
			runCtxTypeID := int64(101)

			grandparentExecution := makeExecution(1, pb.Execution_RUNNING, map[string]*pb.Value{
				keyTotalDagTasks: intValue(1),
			})
			parentExecution := makeExecution(2, pb.Execution_RUNNING, map[string]*pb.Value{
				keyTotalDagTasks: intValue(1),
				keyParentDagID:   intValue(1),
				keyTaskName:      StringValue("parent-task"),
			})
			leafExecution := makeExecution(3, tt.leafState, map[string]*pb.Value{
				keyTaskName:    StringValue("leaf-task"),
				keyParentDagID: intValue(2),
			})

			store := map[int64]*pb.Execution{
				1: grandparentExecution,
				2: parentExecution,
				3: leafExecution,
			}

			pipelineCtx := &pb.Context{Id: int64Ptr(600), TypeId: &pipelineCtxTypeID}
			runCtx := &pb.Context{Id: &runCtxID, TypeId: &runCtxTypeID}
			sharedContexts := []*pb.Context{pipelineCtx, runCtx}

			callCount := 0
			mock := &mockMLMDClient{
				getExecutionsByContextFn: func(_ context.Context, _ *pb.GetExecutionsByContextRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByContextResponse, error) {
					callCount++
					switch callCount {
					case 1:
						return &pb.GetExecutionsByContextResponse{Executions: []*pb.Execution{leafExecution}}, nil
					case 2:
						return &pb.GetExecutionsByContextResponse{Executions: []*pb.Execution{parentExecution}}, nil
					default:
						return &pb.GetExecutionsByContextResponse{}, nil
					}
				},
				getExecutionsByIDFn: func(_ context.Context, req *pb.GetExecutionsByIDRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByIDResponse, error) {
					var results []*pb.Execution
					for _, id := range req.GetExecutionIds() {
						if e, ok := store[id]; ok {
							results = append(results, e)
						}
					}
					return &pb.GetExecutionsByIDResponse{Executions: results}, nil
				},
				getContextsByExecutionFn: func(_ context.Context, _ *pb.GetContextsByExecutionRequest, _ ...grpc.CallOption) (*pb.GetContextsByExecutionResponse, error) {
					return &pb.GetContextsByExecutionResponse{Contexts: sharedContexts}, nil
				},
				getContextTypeFn: func(_ context.Context, _ *pb.GetContextTypeRequest, _ ...grpc.CallOption) (*pb.GetContextTypeResponse, error) {
					return &pb.GetContextTypeResponse{
						ContextType: &pb.ContextType{Id: &pipelineCtxTypeID},
					}, nil
				},
				putExecutionFn: defaultPutExecution(store),
			}

			client := newTestClient(mock)
			parentDAG := makeDAG(parentExecution)
			pipeline := &Pipeline{
				pipelineCtx:    pipelineCtx,
				pipelineRunCtx: runCtx,
			}

			err := client.UpdateDAGExecutionsState(context.Background(), parentDAG, pipeline)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parentExecution.GetLastKnownState() != tt.expectedState {
				t.Errorf("expected parent DAG %v, got %v", tt.expectedState, parentExecution.GetLastKnownState())
			}
			if grandparentExecution.GetLastKnownState() != tt.expectedState {
				t.Errorf("expected grandparent DAG %v, got %v", tt.expectedState, grandparentExecution.GetLastKnownState())
			}
		})
	}
}

func TestUpdateDAGExecutionsState_PropagationStopsAtRootDAG(t *testing.T) {
	// parentDAG (no parent_dag_id) → task(COMPLETE)
	// Should complete parentDAG but not attempt further propagation.
	dagExecution := makeExecution(10, pb.Execution_RUNNING, map[string]*pb.Value{
		keyTotalDagTasks: intValue(1),
	})
	task := makeExecution(11, pb.Execution_COMPLETE, map[string]*pb.Value{
		keyTaskName:    StringValue("task1"),
		keyParentDagID: intValue(10),
	})

	runCtxID := int64(500)
	pipeline := &Pipeline{pipelineRunCtx: &pb.Context{Id: &runCtxID}}
	dag := makeDAG(dagExecution)

	store := map[int64]*pb.Execution{10: dagExecution, 11: task}

	putCallCount := 0
	mock := buildMock(
		store,
		map[int64][]*pb.Context{10: {}},
		map[int64][]*pb.Execution{runCtxID: {task}},
		func(_ context.Context, req *pb.PutExecutionRequest, _ ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
			putCallCount++
			execution := req.GetExecution()
			if stored, ok := store[execution.GetId()]; ok {
				stored.LastKnownState = execution.LastKnownState
			}
			return &pb.PutExecutionResponse{}, nil
		},
	)
	client := newTestClient(mock)

	err := client.UpdateDAGExecutionsState(context.Background(), dag, pipeline)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if putCallCount != 1 {
		t.Errorf("expected exactly 1 PutExecution call (no further propagation), got %d", putCallCount)
	}
	if dagExecution.GetLastKnownState() != pb.Execution_COMPLETE {
		t.Errorf("expected DAG state COMPLETE, got %v", dagExecution.GetLastKnownState())
	}
}

func TestUpdateDAGExecutionsState_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() *mockMLMDClient
	}{
		{
			name: "PutExecution error",
			setupMock: func() *mockMLMDClient {
				dagExecution := makeExecution(10, pb.Execution_RUNNING, map[string]*pb.Value{
					keyTotalDagTasks: intValue(1),
				})
				task := makeExecution(11, pb.Execution_COMPLETE, map[string]*pb.Value{
					keyTaskName:    StringValue("task1"),
					keyParentDagID: intValue(10),
				})
				store := map[int64]*pb.Execution{10: dagExecution, 11: task}
				runCtxID := int64(500)
				return buildMock(
					store,
					map[int64][]*pb.Context{10: {}},
					map[int64][]*pb.Execution{runCtxID: {task}},
					func(_ context.Context, _ *pb.PutExecutionRequest, _ ...grpc.CallOption) (*pb.PutExecutionResponse, error) {
						return nil, fmt.Errorf("simulated MLMD write failure")
					},
				)
			},
		},
		{
			name: "GetExecutionsByContext error",
			setupMock: func() *mockMLMDClient {
				return &mockMLMDClient{
					getExecutionsByContextFn: func(_ context.Context, _ *pb.GetExecutionsByContextRequest, _ ...grpc.CallOption) (*pb.GetExecutionsByContextResponse, error) {
						return nil, fmt.Errorf("simulated context query failure")
					},
				}
			},
		},
		{
			name: "parent DAG not found during recursion",
			setupMock: func() *mockMLMDClient {
				dagExecution := makeExecution(10, pb.Execution_RUNNING, map[string]*pb.Value{
					keyTotalDagTasks: intValue(1),
					keyParentDagID:   intValue(99),
				})
				task := makeExecution(11, pb.Execution_COMPLETE, map[string]*pb.Value{
					keyTaskName:    StringValue("task1"),
					keyParentDagID: intValue(10),
				})
				store := map[int64]*pb.Execution{10: dagExecution, 11: task}
				runCtxID := int64(500)
				return buildMock(
					store,
					map[int64][]*pb.Context{10: {}},
					map[int64][]*pb.Execution{runCtxID: {task}},
					defaultPutExecution(store),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			client := newTestClient(mock)

			dagExecution := makeExecution(10, pb.Execution_RUNNING, map[string]*pb.Value{
				keyTotalDagTasks: intValue(1),
			})
			// For the GetExecutionsByContext error case, we use the dag from here.
			// For the other cases, the mock's internal store has the real dag.
			// We need to use the mock's store dag if it exists.
			if mock.getExecutionsByIDFn != nil {
				resp, _ := mock.getExecutionsByIDFn(context.Background(), &pb.GetExecutionsByIDRequest{ExecutionIds: []int64{10}}, nil)
				if resp != nil && len(resp.Executions) > 0 {
					dagExecution = resp.Executions[0]
				}
			}

			runCtxID := int64(500)
			pipeline := &Pipeline{pipelineRunCtx: &pb.Context{Id: &runCtxID}}
			dag := makeDAG(dagExecution)

			err := client.UpdateDAGExecutionsState(context.Background(), dag, pipeline)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
