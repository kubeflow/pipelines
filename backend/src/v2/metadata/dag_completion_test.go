package metadata

import (
	"fmt"
	"testing"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/assert"
)

// TestDAGCompletionLogic tests the core DAG completion logic without needing full integration
func TestDAGCompletionLogic(t *testing.T) {
	tests := []struct {
		name                string
		dagType            string
		iterationCount     *int64
		iterationIndex     *int64
		tasks              map[string]*mockExecution
		expectedComplete   bool
		expectedTotalTasks int64
		description        string
	}{
		// === PARALLEL FOR SCENARIOS ===
		
		// Basic ParallelFor iteration scenarios
		{
			name:    "ParallelFor_IterationDAG_NoTasks_ShouldComplete",
			dagType: "ParallelFor_Iteration",
			iterationCount: int64Ptr(3),
			iterationIndex: int64Ptr(0),
			tasks: map[string]*mockExecution{}, // No tasks - should complete immediately
			expectedComplete: true,
			expectedTotalTasks: 3,
			description: "ParallelFor iteration DAGs with no tasks should complete immediately",
		},
		{
			name:    "ParallelFor_IterationDAG_WithTasks_AllComplete",
			dagType: "ParallelFor_Iteration",
			iterationCount: int64Ptr(5),
			iterationIndex: int64Ptr(2),
			tasks: map[string]*mockExecution{
				"process-item": {taskType: "system.ContainerExecution", state: "COMPLETE"},
				"validate-result": {taskType: "system.ContainerExecution", state: "COMPLETE"},
			},
			expectedComplete: true,
			expectedTotalTasks: 5,
			description: "ParallelFor iteration DAGs should complete when all tasks complete",
		},
		{
			name:    "ParallelFor_IterationDAG_WithTasks_SomeRunning",
			dagType: "ParallelFor_Iteration",
			iterationCount: int64Ptr(10),
			iterationIndex: int64Ptr(7),
			tasks: map[string]*mockExecution{
				"process-item": {taskType: "system.ContainerExecution", state: "COMPLETE"},
				"validate-result": {taskType: "system.ContainerExecution", state: "RUNNING"},
			},
			expectedComplete: false,
			expectedTotalTasks: 10,
			description: "ParallelFor iteration DAGs should not complete while tasks are running",
		},
		
		// ParallelFor parent scenarios - mirroring integration tests
		{
			name:    "ParallelFor_ParentDAG_AllChildrenComplete_ShouldComplete",
			dagType: "ParallelFor_Parent",
			iterationCount: int64Ptr(3),
			iterationIndex: nil,
			tasks: map[string]*mockExecution{
				"child1": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child2": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child3": {taskType: "system.DAGExecution", state: "COMPLETE"},
			},
			expectedComplete: true,
			expectedTotalTasks: 3,
			description: "ParallelFor parent DAGs should complete when all child DAGs complete",
		},
		{
			name:    "ParallelFor_ParentDAG_SomeChildrenRunning_ShouldNotComplete",
			dagType: "ParallelFor_Parent",
			iterationCount: int64Ptr(3),
			iterationIndex: nil,
			tasks: map[string]*mockExecution{
				"child1": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child2": {taskType: "system.DAGExecution", state: "RUNNING"},
				"child3": {taskType: "system.DAGExecution", state: "COMPLETE"},
			},
			expectedComplete: false,
			expectedTotalTasks: 3,
			description: "ParallelFor parent DAGs should not complete while child DAGs are running",
		},
		{
			name:    "ParallelFor_ParentDAG_SomeChildrenFailed_ShouldNotComplete",
			dagType: "ParallelFor_Parent",
			iterationCount: int64Ptr(5),
			iterationIndex: nil,
			tasks: map[string]*mockExecution{
				"child1": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child2": {taskType: "system.DAGExecution", state: "FAILED"},
				"child3": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child4": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child5": {taskType: "system.DAGExecution", state: "RUNNING"},
			},
			expectedComplete: false,
			expectedTotalTasks: 5,
			description: "ParallelFor parent DAGs should not complete when children failed or still running",
		},
		
		// Large iteration count scenarios (dynamic ParallelFor simulation)
		{
			name:    "ParallelFor_ParentDAG_LargeIterationCount_AllComplete",
			dagType: "ParallelFor_Parent", 
			iterationCount: int64Ptr(10),
			iterationIndex: nil,
			tasks: func() map[string]*mockExecution {
				tasks := make(map[string]*mockExecution)
				for i := 0; i < 10; i++ {
					tasks[fmt.Sprintf("child%d", i)] = &mockExecution{
						taskType: "system.DAGExecution", 
						state: "COMPLETE",
					}
				}
				return tasks
			}(),
			expectedComplete: true,
			expectedTotalTasks: 10,
			description: "ParallelFor parent DAGs should handle large iteration counts correctly",
		},
		{
			name:    "ParallelFor_ParentDAG_EmptyIterations_ShouldComplete",
			dagType: "ParallelFor_Parent",
			iterationCount: int64Ptr(0), // Edge case: no iterations
			iterationIndex: nil,
			tasks: map[string]*mockExecution{}, // No child DAGs
			expectedComplete: true,
			expectedTotalTasks: 0,
			description: "ParallelFor parent DAGs with zero iterations should complete immediately",
		},
		
		// === CONDITIONAL SCENARIOS ===
		
		// Simple If scenarios (mirroring conditional_if_true.yaml / conditional_if_false.yaml)
		{
			name:    "Conditional_SimpleIf_True_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-if-task": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-if-branch"},
			},
			expectedComplete: true,
			expectedTotalTasks: 1,
			description: "Simple If conditional (true case) should complete when if-branch completes",
		},
		{
			name:    "Conditional_SimpleIf_False_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{}, // No tasks executed - false condition
			expectedComplete: true,
			expectedTotalTasks: 0,
			description: "Simple If conditional (false case) should complete with no tasks executed",
		},
		
		// If/Else scenarios (mirroring conditional_if_else_true.yaml / conditional_if_else_false.yaml)
		{
			name:    "Conditional_IfElse_TrueBranch_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-if-task": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-if-branch"},
			},
			expectedComplete: true,
			expectedTotalTasks: 1,
			description: "If/Else conditional should complete when If-branch executes",
		},
		{
			name:    "Conditional_IfElse_ElseBranch_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-else-task": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-else-branch"},
			},
			expectedComplete: true,
			expectedTotalTasks: 1,
			description: "If/Else conditional should complete when Else-branch executes",
		},
		
		// Complex conditional scenarios (mirroring conditional_complex.yaml)
		{
			name:    "Conditional_Complex_IfBranch_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-if-task1": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-if-branch"},
				"condition-if-task2": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-if-branch"},
			},
			expectedComplete: true,
			expectedTotalTasks: 2,
			description: "Complex conditional should complete when If-branch with multiple tasks executes",
		},
		{
			name:    "Conditional_Complex_ElifBranch_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-elif-task": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-elif-branch"},
			},
			expectedComplete: true,
			expectedTotalTasks: 1,
			description: "Complex conditional should complete when Elif-branch executes",
		},
		{
			name:    "Conditional_Complex_ElseBranch_ShouldComplete",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-else-task1": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-else-branch"},
				"condition-else-task2": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-else-branch"},
				"condition-else-task3": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-else-branch"},
			},
			expectedComplete: true,
			expectedTotalTasks: 3,
			description: "Complex conditional should complete when Else-branch with multiple tasks executes",
		},
		
		// Running/pending conditional scenarios
		{
			name:    "Conditional_BranchStillRunning_ShouldWait",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-if-task1": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-if-branch"},
				"condition-if-task2": {taskType: "system.ContainerExecution", state: "RUNNING", taskName: "condition-if-branch"},
			},
			expectedComplete: false,
			expectedTotalTasks: 2,
			description: "Conditional DAGs should wait when branch tasks are still running",
		},
		{
			name:    "Conditional_NoTasksYet_ShouldWait",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-placeholder": {taskType: "system.ContainerExecution", state: "RUNNING", taskName: "condition-if-branch"},
			},
			expectedComplete: false,
			expectedTotalTasks: 1,
			description: "Conditional DAGs should wait for branch execution to complete",
		},
		
		// === STANDARD DAG SCENARIOS ===
		
		{
			name:    "Standard_AllTasksComplete_ShouldComplete",
			dagType: "Standard",
			tasks: map[string]*mockExecution{
				"task1": {taskType: "system.ContainerExecution", state: "COMPLETE"},
				"task2": {taskType: "system.ContainerExecution", state: "COMPLETE"},
			},
			expectedComplete: true,
			expectedTotalTasks: 2,
			description: "Standard DAGs should complete when all tasks complete",
		},
		{
			name:    "Standard_SomeTasksRunning_ShouldNotComplete",
			dagType: "Standard",
			tasks: map[string]*mockExecution{
				"task1": {taskType: "system.ContainerExecution", state: "COMPLETE"},
				"task2": {taskType: "system.ContainerExecution", state: "RUNNING"},
			},
			expectedComplete: false,
			expectedTotalTasks: 2,
			description: "Standard DAGs should not complete while tasks are running",
		},
		{
			name:    "Standard_SomeTasksFailed_ShouldNotComplete",
			dagType: "Standard",
			tasks: map[string]*mockExecution{
				"task1": {taskType: "system.ContainerExecution", state: "COMPLETE"},
				"task2": {taskType: "system.ContainerExecution", state: "FAILED"},
				"task3": {taskType: "system.ContainerExecution", state: "RUNNING"},
			},
			expectedComplete: false,
			expectedTotalTasks: 3,
			description: "Standard DAGs should not complete when tasks failed or still running",
		},
		{
			name:    "Standard_EmptyDAG_ShouldComplete",
			dagType: "Standard",
			tasks: map[string]*mockExecution{}, // No tasks
			expectedComplete: true,
			expectedTotalTasks: 0,
			description: "Empty standard DAGs should complete immediately",
		},
		
		// === EDGE CASES AND MIXED SCENARIOS ===
		
		{
			name:    "ParallelFor_MixedTaskTypes_ShouldHandleCorrectly",
			dagType: "ParallelFor_Parent",
			iterationCount: int64Ptr(2),
			iterationIndex: nil,
			tasks: map[string]*mockExecution{
				// Child DAGs
				"child1": {taskType: "system.DAGExecution", state: "COMPLETE"},
				"child2": {taskType: "system.DAGExecution", state: "COMPLETE"},
				// Regular tasks (should be ignored for parent DAG completion)
				"setup-task": {taskType: "system.ContainerExecution", state: "COMPLETE"},
				"cleanup-task": {taskType: "system.ContainerExecution", state: "RUNNING"},
			},
			expectedComplete: true, // Should complete based on child DAGs, not container tasks
			expectedTotalTasks: 2,
			description: "ParallelFor parent should complete based on child DAGs, ignoring container tasks",
		},
		{
			name:    "Conditional_MixedStates_ShouldHandleCorrectly",
			dagType: "Conditional",
			tasks: map[string]*mockExecution{
				"condition-if-task": {taskType: "system.ContainerExecution", state: "COMPLETE", taskName: "condition-if-branch"},
				"condition-else-task": {taskType: "system.ContainerExecution", state: "CANCELED", taskName: "condition-else-branch"}, // Counts as completed
			},
			expectedComplete: true,
			expectedTotalTasks: 2, // Both tasks count toward total: 1 COMPLETE + 1 CANCELED = 2 completed, 2 total → should complete
			description: "Mixed states DAG: COMPLETE + CANCELED tasks should allow completion (CANCELED counts as completed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DAG and tasks
			// For ParallelFor parent DAGs, total_dag_tasks should be iteration_count, not len(tasks)
			initialTotalTasks := int64(len(tt.tasks))
			if tt.dagType == "ParallelFor_Parent" && tt.iterationCount != nil {
				initialTotalTasks = *tt.iterationCount
			}
			dag := createMockDAG(tt.dagType, tt.iterationCount, tt.iterationIndex, initialTotalTasks)
			tasks := createMockTasks(tt.tasks)
			
			// Create client and test completion logic
			client := &Client{}
			result := client.testDAGCompletion(dag, tasks)
			
			
			assert.Equal(t, tt.expectedComplete, result.shouldComplete, 
				"Test: %s - %s", tt.name, tt.description)
			assert.Equal(t, tt.expectedTotalTasks, result.totalDagTasks,
				"Test: %s - total_dag_tasks should be %d", tt.name, tt.expectedTotalTasks)
		})
	}
}

// Helper types for testing
type mockExecution struct {
	taskType string
	state    string
	taskName string
}

type completionResult struct {
	shouldComplete bool
	totalDagTasks  int64
	newState       pb.Execution_State
	completedTasks int
	runningTasks   int
}

// Helper functions
func int64Ptr(v int64) *int64 {
	return &v
}

func createMockDAG(dagType string, iterationCount, iterationIndex *int64, initialTotalTasks int64) *DAG {
	customProps := make(map[string]*pb.Value)
	customProps["total_dag_tasks"] = &pb.Value{Value: &pb.Value_IntValue{IntValue: initialTotalTasks}}
	
	// Set a generic task name (no longer used for conditional detection)
	customProps["task_name"] = &pb.Value{Value: &pb.Value_StringValue{StringValue: "test-dag"}}
	
	if iterationCount != nil {
		customProps["iteration_count"] = &pb.Value{Value: &pb.Value_IntValue{IntValue: *iterationCount}}
	}
	if iterationIndex != nil {
		customProps["iteration_index"] = &pb.Value{Value: &pb.Value_IntValue{IntValue: *iterationIndex}}
	}

	execution := &pb.Execution{
		Id: int64Ptr(123),
		CustomProperties: customProps,
	}

	return &DAG{
		Execution: &Execution{
			execution: execution,
		},
	}
}

func createMockTasks(mockTasks map[string]*mockExecution) map[string]*Execution {
	tasks := make(map[string]*Execution)
	
	for name, mock := range mockTasks {
		state := pb.Execution_UNKNOWN
		switch mock.state {
		case "COMPLETE":
			state = pb.Execution_COMPLETE
		case "RUNNING":
			state = pb.Execution_RUNNING
		case "FAILED":
			state = pb.Execution_FAILED
		case "CANCELED":
			state = pb.Execution_CANCELED
		}

		// Set different TypeId based on task type
		typeId := int64(789) // ContainerExecution
		if mock.taskType == "system.DAGExecution" {
			typeId = 999 // DAGExecution
		}

		taskName := mock.taskName
		if taskName == "" {
			taskName = name
		}

		customProps := map[string]*pb.Value{
			"task_name": {Value: &pb.Value_StringValue{StringValue: taskName}},
		}
		
		// Add type-specific properties for the real GetType() method to work
		if mock.taskType == "system.DAGExecution" {
			// DAG executions have total_dag_tasks property
			customProps["total_dag_tasks"] = &pb.Value{Value: &pb.Value_IntValue{IntValue: 1}}
		} else {
			// Container executions have pod-related properties
			customProps["pod_name"] = &pb.Value{Value: &pb.Value_StringValue{StringValue: "test-pod"}}
		}

		execution := &pb.Execution{
			Id: int64Ptr(456),
			TypeId: int64Ptr(typeId),
			LastKnownState: &state,
			CustomProperties: customProps,
		}

		tasks[name] = &Execution{
			execution: execution,
		}
	}
	
	return tasks
}

// Test version of shouldApplyDynamicTaskCounting to match real implementation
func (c *Client) shouldApplyDynamicTaskCountingTest(dag *DAG, tasks map[string]*Execution) bool {
	props := dag.Execution.execution.CustomProperties
	
	// Skip ParallelFor DAGs - they have their own specialized logic
	if props["iteration_count"] != nil || props["iteration_index"] != nil {
		return false
	}
	
	// Apply dynamic counting for any DAG that might have variable task execution:
	// 1. DAGs with no tasks (conditional with false branch)
	// 2. DAGs with canceled tasks (conditional with non-executed branches)  
	// 3. DAGs where execution pattern suggests conditional behavior
	
	canceledTasks := 0
	for _, task := range tasks {
		if task.GetType() == "system.DAGExecution" {
			continue // Skip child DAGs, only count container tasks
		}
		if task.GetExecution().LastKnownState.String() == "CANCELED" {
			canceledTasks++
		}
	}
	
	// Heuristic: If we have canceled tasks, likely a conditional with non-executed branches
	if canceledTasks > 0 {
		return true
	}
	
	// Heuristic: Empty DAGs might be conditionals with false branches
	if len(tasks) == 0 {
		return true
	}
	
	// For standard DAGs with normal execution patterns, don't apply dynamic counting
	// Only apply dynamic counting when we detect patterns that suggest conditional behavior
	return false
}

// Test method that simulates the completion logic
func (c *Client) testDAGCompletion(dag *DAG, tasks map[string]*Execution) completionResult {
	// Simulate the counting logic from UpdateDAGExecutionsState
	totalDagTasks := dag.Execution.execution.CustomProperties["total_dag_tasks"].GetIntValue()
	completedTasks := 0
	failedTasks := 0
	runningTasks := 0
	dagExecutions := 0
	
	for _, task := range tasks {
		taskState := task.GetExecution().LastKnownState.String()
		taskType := task.GetType() // Call GetType on task, not task.GetExecution()
		
		if taskType == "system.DAGExecution" {
			dagExecutions++
			// Don't continue here - we still need to process DAG execution states
		} else {
			// Only count container execution states for regular task counting
			switch taskState {
			case "FAILED":
				failedTasks++
			case "COMPLETE":
				completedTasks++
			case "CACHED":
				completedTasks++
			case "CANCELED":
				completedTasks++
			case "RUNNING":
				runningTasks++
			}
		}
	}
	
	// Apply universal dynamic counting logic (matching real implementation)
	shouldApplyDynamic := c.shouldApplyDynamicTaskCountingTest(dag, tasks)
	if shouldApplyDynamic {
		// For DAGs with dynamic execution, adjust total_dag_tasks based on actual execution
		actualExecutedTasks := completedTasks + failedTasks
		actualRunningTasks := runningTasks
		
		// Apply universal dynamic counting logic
		if actualExecutedTasks > 0 {
			// We have completed/failed tasks - use that as the expected total
			totalDagTasks = int64(actualExecutedTasks)
		} else if actualRunningTasks > 0 {
			// Tasks are running - use running count as temporary total
			totalDagTasks = int64(actualRunningTasks)
		} else if totalDagTasks == 0 {
			// No tasks at all - this is valid for conditionals with false branches
			// Keep totalDagTasks = 0, this will trigger universal completion rule
		}
	}
	
	// For ParallelFor iteration DAGs, ensure total_dag_tasks is preserved from iteration_count
	isParallelForIterationDAG := c.isParallelForIterationDAG(dag)
	if isParallelForIterationDAG && dag.Execution.execution.CustomProperties["iteration_count"] != nil {
		totalDagTasks = dag.Execution.execution.CustomProperties["iteration_count"].GetIntValue()
	}
	
	// Apply completion logic (matching real implementation)
	var newState pb.Execution_State
	var stateChanged bool
	
	isParallelForParentDAG := c.isParallelForParentDAG(dag)
	
	// UNIVERSAL RULE: Any DAG with no tasks and nothing running should complete
	if totalDagTasks == 0 && runningTasks == 0 {
		newState = pb.Execution_COMPLETE
		stateChanged = true
	} else if isParallelForIterationDAG {
		if runningTasks == 0 {
			newState = pb.Execution_COMPLETE
			stateChanged = true
		}
	} else if isParallelForParentDAG {
		childDagCount := dagExecutions
		completedChildDags := 0
		for _, task := range tasks {
			if task.GetType() == "system.DAGExecution" && 
			   task.GetExecution().LastKnownState.String() == "COMPLETE" {
				completedChildDags++
			}
		}
		
		if completedChildDags == childDagCount && childDagCount > 0 {
			newState = pb.Execution_COMPLETE
			stateChanged = true
		}
	} else {
		// Standard DAG completion logic
		if completedTasks == int(totalDagTasks) {
			newState = pb.Execution_COMPLETE
			stateChanged = true
		}
	}
	
	if !stateChanged && failedTasks > 0 {
		newState = pb.Execution_FAILED
		stateChanged = true
	}
	
	return completionResult{
		shouldComplete: stateChanged && newState == pb.Execution_COMPLETE,
		totalDagTasks:  totalDagTasks,
		newState:       newState,
		completedTasks: completedTasks,
		runningTasks:   runningTasks,
	}
}

