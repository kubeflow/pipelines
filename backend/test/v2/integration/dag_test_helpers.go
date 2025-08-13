// Copyright 2025 The Kubeflow Authors
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

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

// DAGTestHelpers provides common helper methods for DAG status testing across test suites
type DAGTestHelpers struct {
	t          *testing.T
	mlmdClient pb.MetadataStoreServiceClient
}

// NewDAGTestHelpers creates a new DAGTestHelpers instance
func NewDAGTestHelpers(t *testing.T, mlmdClient pb.MetadataStoreServiceClient) *DAGTestHelpers {
	return &DAGTestHelpers{
		t:          t,
		mlmdClient: mlmdClient,
	}
}

// GetExecutionsForRun retrieves all executions for a specific run ID
func (h *DAGTestHelpers) GetExecutionsForRun(runID string) []*pb.Execution {
	contextsFilterQuery := util.StringPointer("name = '" + runID + "'")
	contexts, err := h.mlmdClient.GetContexts(context.Background(), &pb.GetContextsRequest{
		Options: &pb.ListOperationOptions{
			FilterQuery: contextsFilterQuery,
		},
	})
	require.NoError(h.t, err)
	require.NotNil(h.t, contexts)
	require.NotEmpty(h.t, contexts.Contexts)

	executionsByContext, err := h.mlmdClient.GetExecutionsByContext(context.Background(), &pb.GetExecutionsByContextRequest{
		ContextId: contexts.Contexts[0].Id,
	})
	require.NoError(h.t, err)
	require.NotNil(h.t, executionsByContext)
	require.NotEmpty(h.t, executionsByContext.Executions)

	return executionsByContext.Executions
}

// GetContextForRun retrieves the context for a specific run ID
func (h *DAGTestHelpers) GetContextForRun(runID string) *pb.Context {
	contextsFilterQuery := util.StringPointer("name = '" + runID + "'")
	contexts, err := h.mlmdClient.GetContexts(context.Background(), &pb.GetContextsRequest{
		Options: &pb.ListOperationOptions{
			FilterQuery: contextsFilterQuery,
		},
	})
	require.NoError(h.t, err)
	require.NotEmpty(h.t, contexts.Contexts)
	return contexts.Contexts[0]
}

// FilterDAGExecutions filters executions to only return DAG executions
func (h *DAGTestHelpers) FilterDAGExecutions(executions []*pb.Execution) []*pb.Execution {
	var dagExecutions []*pb.Execution
	for _, execution := range executions {
		if execution.GetType() == "system.DAGExecution" {
			dagExecutions = append(dagExecutions, execution)
		}
	}
	return dagExecutions
}

// FilterContainerExecutions filters executions to only return container executions
func (h *DAGTestHelpers) FilterContainerExecutions(executions []*pb.Execution) []*pb.Execution {
	var containerExecutions []*pb.Execution
	for _, execution := range executions {
		if execution.GetType() == "system.ContainerExecution" {
			containerExecutions = append(containerExecutions, execution)
		}
	}
	return containerExecutions
}

// GetExecutionProperty safely retrieves a property value from an execution
func (h *DAGTestHelpers) GetExecutionProperty(execution *pb.Execution, propertyName string) string {
	if props := execution.GetCustomProperties(); props != nil {
		if prop := props[propertyName]; prop != nil {
			return prop.GetStringValue()
		}
	}
	return ""
}

// GetExecutionIntProperty safely retrieves an integer property value from an execution
func (h *DAGTestHelpers) GetExecutionIntProperty(execution *pb.Execution, propertyName string) int64 {
	if props := execution.GetCustomProperties(); props != nil {
		if prop := props[propertyName]; prop != nil {
			return prop.GetIntValue()
		}
	}
	return 0
}

// GetTaskName retrieves the task_name property from an execution
func (h *DAGTestHelpers) GetTaskName(execution *pb.Execution) string {
	return h.GetExecutionProperty(execution, "task_name")
}

// GetParentDagID retrieves the parent_dag_id property from an execution
func (h *DAGTestHelpers) GetParentDagID(execution *pb.Execution) int64 {
	return h.GetExecutionIntProperty(execution, "parent_dag_id")
}

// GetTotalDagTasks retrieves the total_dag_tasks property from an execution
func (h *DAGTestHelpers) GetTotalDagTasks(execution *pb.Execution) int64 {
	return h.GetExecutionIntProperty(execution, "total_dag_tasks")
}

// GetIterationCount retrieves the iteration_count property from an execution
func (h *DAGTestHelpers) GetIterationCount(execution *pb.Execution) int64 {
	return h.GetExecutionIntProperty(execution, "iteration_count")
}

// GetIterationIndex retrieves the iteration_index property from an execution
// Returns -1 if the property doesn't exist (indicating this is not an iteration DAG)
func (h *DAGTestHelpers) GetIterationIndex(execution *pb.Execution) int64 {
	if props := execution.GetCustomProperties(); props != nil {
		if prop := props["iteration_index"]; prop != nil {
			return prop.GetIntValue()
		}
	}
	return -1 // Not found
}

// FindRootDAG finds the root DAG execution (no parent_dag_id and empty task_name)
func (h *DAGTestHelpers) FindRootDAG(executions []*pb.Execution) *pb.Execution {
	dagExecutions := h.FilterDAGExecutions(executions)
	for _, execution := range dagExecutions {
		taskName := h.GetTaskName(execution)
		parentDagID := h.GetParentDagID(execution)
		
		// Root DAG has empty task name and no parent
		if taskName == "" && parentDagID == 0 {
			return execution
		}
	}
	return nil
}

// IsRecentExecution checks if an execution was created within the last 5 minutes
func (h *DAGTestHelpers) IsRecentExecution(execution *pb.Execution) bool {
	if execution.CreateTimeSinceEpoch == nil {
		return false
	}
	
	createdTime := *execution.CreateTimeSinceEpoch
	now := time.Now().UnixMilli()
	return now-createdTime < 5*60*1000 // Within 5 minutes
}

// LogExecutionSummary logs a summary of an execution for debugging
func (h *DAGTestHelpers) LogExecutionSummary(execution *pb.Execution, prefix string) {
	taskName := h.GetTaskName(execution)
	parentDagID := h.GetParentDagID(execution)
	totalDagTasks := h.GetTotalDagTasks(execution)
	
	h.t.Logf("%s Execution ID=%d, Type=%s, State=%s, TaskName='%s', ParentDAG=%d, TotalTasks=%d",
		prefix, execution.GetId(), execution.GetType(), execution.LastKnownState.String(),
		taskName, parentDagID, totalDagTasks)
}

// CategorizeExecutionsByType categorizes executions into DAGs and containers with root DAG identification
func (h *DAGTestHelpers) CategorizeExecutionsByType(executions []*pb.Execution) (dagExecutions []*pb.Execution, containerExecutions []*pb.Execution, rootDAGID int64) {
	h.t.Logf("=== Categorizing %d executions ===", len(executions))
	
	for _, execution := range executions {
		h.LogExecutionSummary(execution, "├──")
		
		if execution.GetType() == "system.DAGExecution" {
			taskName := h.GetTaskName(execution)
			
			// Identify the root DAG (has empty task name)
			if taskName == "" {
				rootDAGID = execution.GetId()
				h.t.Logf("Found root DAG ID=%d", rootDAGID)
			}
			
			dagExecutions = append(dagExecutions, execution)
		} else if execution.GetType() == "system.ContainerExecution" {
			containerExecutions = append(containerExecutions, execution)
		}
	}
	
	h.t.Logf("Summary: %d DAG executions, %d container executions, root DAG ID=%d", 
		len(dagExecutions), len(containerExecutions), rootDAGID)
	
	return dagExecutions, containerExecutions, rootDAGID
}

// GetAllDAGExecutions retrieves all DAG executions from the system (for cross-context searches)
func (h *DAGTestHelpers) GetAllDAGExecutions() []*pb.Execution {
	allDAGExecutions, err := h.mlmdClient.GetExecutionsByType(context.Background(), &pb.GetExecutionsByTypeRequest{
		TypeName: util.StringPointer("system.DAGExecution"),
	})
	require.NoError(h.t, err)
	require.NotNil(h.t, allDAGExecutions)
	
	return allDAGExecutions.Executions
}

// FindExecutionsByTaskNamePrefix finds executions with task names starting with the given prefix
func (h *DAGTestHelpers) FindExecutionsByTaskNamePrefix(executions []*pb.Execution, prefix string) []*pb.Execution {
	var matchingExecutions []*pb.Execution
	for _, execution := range executions {
		taskName := h.GetTaskName(execution)
		if len(taskName) > 0 && len(prefix) > 0 {
			if len(taskName) >= len(prefix) && taskName[:len(prefix)] == prefix {
				matchingExecutions = append(matchingExecutions, execution)
			}
		}
	}
	return matchingExecutions
}

// FindChildDAGExecutions finds all child DAG executions for a given parent DAG ID
func (h *DAGTestHelpers) FindChildDAGExecutions(allExecutions []*pb.Execution, parentDAGID int64) []*pb.Execution {
	var childDAGs []*pb.Execution
	dagExecutions := h.FilterDAGExecutions(allExecutions)
	
	for _, execution := range dagExecutions {
		if h.GetParentDagID(execution) == parentDAGID {
			childDAGs = append(childDAGs, execution)
		}
	}
	
	return childDAGs
}