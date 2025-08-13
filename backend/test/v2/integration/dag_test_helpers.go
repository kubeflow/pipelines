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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
)

// DAGTestUtil provides common helper methods for DAG status testing across test suites
type DAGTestUtil struct {
	t          *testing.T
	mlmdClient pb.MetadataStoreServiceClient
}

// NewDAGTestHelpers creates a new DAGTestUtil instance
func NewDAGTestHelpers(t *testing.T, mlmdClient pb.MetadataStoreServiceClient) *DAGTestUtil {
	return &DAGTestUtil{
		t:          t,
		mlmdClient: mlmdClient,
	}
}

// GetExecutionsForRun retrieves all executions for a specific run ID
func (h *DAGTestUtil) GetExecutionsForRun(runID string) []*pb.Execution {
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
func (h *DAGTestUtil) GetContextForRun(runID string) *pb.Context {
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
func (h *DAGTestUtil) FilterDAGExecutions(executions []*pb.Execution) []*pb.Execution {
	var dagExecutions []*pb.Execution
	for _, execution := range executions {
		if execution.GetType() == "system.DAGExecution" {
			dagExecutions = append(dagExecutions, execution)
		}
	}
	return dagExecutions
}

// FilterContainerExecutions filters executions to only return container executions
func (h *DAGTestUtil) FilterContainerExecutions(executions []*pb.Execution) []*pb.Execution {
	var containerExecutions []*pb.Execution
	for _, execution := range executions {
		if execution.GetType() == "system.ContainerExecution" {
			containerExecutions = append(containerExecutions, execution)
		}
	}
	return containerExecutions
}

// GetExecutionProperty safely retrieves a property value from an execution
func (h *DAGTestUtil) GetExecutionProperty(execution *pb.Execution, propertyName string) string {
	if props := execution.GetCustomProperties(); props != nil {
		if prop := props[propertyName]; prop != nil {
			return prop.GetStringValue()
		}
	}
	return ""
}

// GetExecutionIntProperty safely retrieves an integer property value from an execution
func (h *DAGTestUtil) GetExecutionIntProperty(execution *pb.Execution, propertyName string) int64 {
	if props := execution.GetCustomProperties(); props != nil {
		if prop := props[propertyName]; prop != nil {
			return prop.GetIntValue()
		}
	}
	return 0
}

// GetTaskName retrieves the task_name property from an execution
func (h *DAGTestUtil) GetTaskName(execution *pb.Execution) string {
	return h.GetExecutionProperty(execution, "task_name")
}

// GetParentDagID retrieves the parent_dag_id property from an execution
func (h *DAGTestUtil) GetParentDagID(execution *pb.Execution) int64 {
	return h.GetExecutionIntProperty(execution, "parent_dag_id")
}

// GetTotalDagTasks retrieves the total_dag_tasks property from an execution
func (h *DAGTestUtil) GetTotalDagTasks(execution *pb.Execution) int64 {
	return h.GetExecutionIntProperty(execution, "total_dag_tasks")
}

// GetIterationCount retrieves the iteration_count property from an execution
func (h *DAGTestUtil) GetIterationCount(execution *pb.Execution) int64 {
	return h.GetExecutionIntProperty(execution, "iteration_count")
}

// GetIterationIndex retrieves the iteration_index property from an execution
// Returns -1 if the property doesn't exist (indicating this is not an iteration DAG)
func (h *DAGTestUtil) GetIterationIndex(execution *pb.Execution) int64 {
	if props := execution.GetCustomProperties(); props != nil {
		if prop := props["iteration_index"]; prop != nil {
			return prop.GetIntValue()
		}
	}
	return -1 // Not found
}

// FindRootDAG finds the root DAG execution (no parent_dag_id and empty task_name)
func (h *DAGTestUtil) FindRootDAG(executions []*pb.Execution) *pb.Execution {
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
func (h *DAGTestUtil) IsRecentExecution(execution *pb.Execution) bool {
	if execution.CreateTimeSinceEpoch == nil {
		return false
	}

	createdTime := *execution.CreateTimeSinceEpoch
	now := time.Now().UnixMilli()
	return now-createdTime < 5*60*1000 // Within 5 minutes
}

// LogExecutionSummary logs a summary of an execution for debugging
func (h *DAGTestUtil) LogExecutionSummary(execution *pb.Execution, prefix string) {
	taskName := h.GetTaskName(execution)
	parentDagID := h.GetParentDagID(execution)
	totalDagTasks := h.GetTotalDagTasks(execution)

	h.t.Logf("%s Execution ID=%d, Type=%s, State=%s, TaskName='%s', ParentDAG=%d, TotalTasks=%d",
		prefix, execution.GetId(), execution.GetType(), execution.LastKnownState.String(),
		taskName, parentDagID, totalDagTasks)
}

// CategorizeExecutionsByType categorizes executions into DAGs and containers with root DAG identification
func (h *DAGTestUtil) CategorizeExecutionsByType(executions []*pb.Execution) (containerExecutions []*pb.Execution, rootDAGID int64) {
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

		} else if execution.GetType() == "system.ContainerExecution" {
			containerExecutions = append(containerExecutions, execution)
		}
	}

	h.t.Logf("Summary: %d container executions, root DAG ID=%d", len(containerExecutions), rootDAGID)

	return containerExecutions, rootDAGID
}

// GetAllDAGExecutions retrieves all DAG executions from the system (for cross-context searches)
func (h *DAGTestUtil) GetAllDAGExecutions() []*pb.Execution {
	allDAGExecutions, err := h.mlmdClient.GetExecutionsByType(context.Background(), &pb.GetExecutionsByTypeRequest{
		TypeName: util.StringPointer("system.DAGExecution"),
	})
	require.NoError(h.t, err)
	require.NotNil(h.t, allDAGExecutions)

	return allDAGExecutions.Executions
}

// FindExecutionsByTaskNamePrefix finds executions with task names starting with the given prefix
func (h *DAGTestUtil) FindExecutionsByTaskNamePrefix(executions []*pb.Execution, prefix string) []*pb.Execution {
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
func (h *DAGTestUtil) FindChildDAGExecutions(allExecutions []*pb.Execution, parentDAGID int64) []*pb.Execution {
	var childDAGs []*pb.Execution
	dagExecutions := h.FilterDAGExecutions(allExecutions)

	for _, execution := range dagExecutions {
		if h.GetParentDagID(execution) == parentDAGID {
			childDAGs = append(childDAGs, execution)
		}
	}

	return childDAGs
}

// ConditionalDAGValidationContext holds the context for conditional DAG validation
type ConditionalDAGValidationContext struct {
	ContainerExecutions   []*pb.Execution
	RootDAGID            int64
	AllConditionalDAGs   []*pb.Execution
	ActualConditionalDAGs []*pb.Execution
}

// GetConditionalDAGContext gets the complete context needed for conditional DAG validation
func (h *DAGTestUtil) GetConditionalDAGContext(runID string) *ConditionalDAGValidationContext {
	// Get executions for the run and categorize them
	executions := h.GetExecutionsForRun(runID)
	containerExecutions, rootDAGID := h.CategorizeExecutionsByType(executions)

	// Find all conditional DAGs related to this run (including cross-context)
	allConditionalDAGs := h.FindAllRelatedConditionalDAGs(rootDAGID)

	// Filter to actual conditional DAGs (exclude root DAG)
	actualConditionalDAGs := h.FilterToActualConditionalDAGs(allConditionalDAGs)

	return &ConditionalDAGValidationContext{
		ContainerExecutions:   containerExecutions,
		RootDAGID:            rootDAGID,
		AllConditionalDAGs:   allConditionalDAGs,
		ActualConditionalDAGs: actualConditionalDAGs,
	}
}

// FindAllRelatedConditionalDAGs searches for all conditional DAGs related to the run
func (h *DAGTestUtil) FindAllRelatedConditionalDAGs(rootDAGID int64) []*pb.Execution {
	if rootDAGID == 0 {
		return []*pb.Execution{}
	}

	h.t.Logf("Searching for conditional DAGs with parent_dag_id=%d", rootDAGID)

	// Get all DAG executions in the system
	allDAGExecutions := h.GetAllDAGExecutions()

	var conditionalDAGs []*pb.Execution
	for _, exec := range allDAGExecutions {
		if h.isConditionalDAGRelatedToRoot(exec, rootDAGID, allDAGExecutions) {
			taskName := h.GetTaskName(exec)
			parentDagID := h.GetParentDagID(exec)
			h.t.Logf("Found conditional DAG for current run: ID=%d, TaskName='%s', State=%s, ParentDAG=%d",
				exec.GetId(), taskName, exec.LastKnownState.String(), parentDagID)
			conditionalDAGs = append(conditionalDAGs, exec)
		}
	}

	h.t.Logf("=== Summary: Found %d total DAG executions ===", len(conditionalDAGs))
	return conditionalDAGs
}

// isConditionalDAGRelatedToRoot checks if a DAG execution is related to the root DAG
func (h *DAGTestUtil) isConditionalDAGRelatedToRoot(exec *pb.Execution, rootDAGID int64, allExecutions []*pb.Execution) bool {
	taskName := h.GetTaskName(exec)
	parentDagID := h.GetParentDagID(exec)

	// Find conditional DAGs that are children OR grandchildren of our root DAG
	isDirectChild := parentDagID == rootDAGID && strings.HasPrefix(taskName, "condition-")

	// Also check if this is a grandchild (parent is a child of root DAG)
	isGrandchild := false
	if strings.HasPrefix(taskName, "condition-") {
		// Find the parent DAG and check if its parent is our root DAG
		for _, parentExec := range allExecutions {
			if parentExec.GetId() == parentDagID && parentExec.GetType() == "system.DAGExecution" {
				if h.GetParentDagID(parentExec) == rootDAGID {
					isGrandchild = true
					break
				}
			}
		}
	}

	return isDirectChild || isGrandchild
}

// FilterToActualConditionalDAGs filters out root DAGs, keeping only conditional DAGs
func (h *DAGTestUtil) FilterToActualConditionalDAGs(dagExecutions []*pb.Execution) []*pb.Execution {
	actualConditionalDAGs := []*pb.Execution{}
	for _, dagExecution := range dagExecutions {
		taskName := h.GetTaskName(dagExecution)

		// Only validate conditional DAGs like "condition-1", "condition-2", "condition-branches-1", not root DAGs
		if taskName != "" && strings.HasPrefix(taskName, "condition-") {
			actualConditionalDAGs = append(actualConditionalDAGs, dagExecution)
		} else {
			h.t.Logf("Skipping root DAG ID=%d (TaskName='%s') - not a conditional branch DAG",
				dagExecution.GetId(), taskName)
		}
	}
	return actualConditionalDAGs
}

// ParallelForDAGValidationContext holds the context for ParallelFor DAG validation
type ParallelForDAGValidationContext struct {
	DAGHierarchy map[int64]*DAGNode
	RootDAG      *DAGNode
	ParallelForParents    []*DAGNode
	ParallelForIterations []*DAGNode
}

// DAGNode represents a node in the DAG hierarchy
type DAGNode struct {
	Execution *pb.Execution
	Parent    *DAGNode
	Children  []*DAGNode
}

// GetParallelForDAGContext gets the complete context needed for ParallelFor DAG validation
func (h *DAGTestUtil) GetParallelForDAGContext(runID string) *ParallelForDAGValidationContext {
	// Get all executions for the run
	executions := h.GetExecutionsForRun(runID)

	// Create DAG nodes from executions
	dagNodes := h.createDAGNodes(executions)

	// Build parent-child relationships
	rootDAG := h.buildParentChildRelationships(dagNodes)

	// Find and categorize DAG nodes
	parallelForParents, parallelForIterations := h.categorizeParallelForDAGs(dagNodes)

	return &ParallelForDAGValidationContext{
		DAGHierarchy:          dagNodes,
		RootDAG:              rootDAG,
		ParallelForParents:    parallelForParents,
		ParallelForIterations: parallelForIterations,
	}
}

// createDAGNodes creates DAGNode objects from executions
func (h *DAGTestUtil) createDAGNodes(executions []*pb.Execution) map[int64]*DAGNode {
	dagNodes := make(map[int64]*DAGNode)

	// Filter to only DAG executions
	dagExecutions := h.FilterDAGExecutions(executions)

	for _, execution := range dagExecutions {
		node := &DAGNode{
			Execution: execution,
			Children:  make([]*DAGNode, 0),
		}
		dagNodes[execution.GetId()] = node

		h.LogExecutionSummary(execution, "Found DAG execution")
	}

	return dagNodes
}

// buildParentChildRelationships establishes parent-child relationships between DAG nodes
func (h *DAGTestUtil) buildParentChildRelationships(dagNodes map[int64]*DAGNode) *DAGNode {
	var rootDAG *DAGNode

	for _, node := range dagNodes {
		parentID := h.GetParentDagID(node.Execution)
		if parentID != 0 {
			if parentNode, exists := dagNodes[parentID]; exists {
				parentNode.Children = append(parentNode.Children, node)
				node.Parent = parentNode
				h.t.Logf("DAG %d is child of DAG %d", node.Execution.GetId(), parentID)
			}
		} else {
			// This is the root DAG
			rootDAG = node
			h.t.Logf("DAG %d is the root DAG", node.Execution.GetId())
		}
	}

	return rootDAG
}

// categorizeParallelForDAGs separates parent and iteration ParallelFor DAGs
func (h *DAGTestUtil) categorizeParallelForDAGs(dagNodes map[int64]*DAGNode) ([]*DAGNode, []*DAGNode) {
	var parallelForParentDAGs []*DAGNode
	var parallelForIterationDAGs []*DAGNode

	for _, node := range dagNodes {
		iterationCount := h.GetIterationCount(node.Execution)
		if iterationCount > 0 {
			// Check if this is a parent DAG (no iteration_index) or iteration DAG (has iteration_index)
			iterationIndex := h.GetIterationIndex(node.Execution)
			if iterationIndex >= 0 {
				// Has iteration_index, so it's an iteration DAG
				parallelForIterationDAGs = append(parallelForIterationDAGs, node)
				h.t.Logf("Found ParallelFor iteration DAG: ID=%d, iteration_index=%d, state=%s",
					node.Execution.GetId(), iterationIndex, (*node.Execution.LastKnownState).String())
			} else {
				// No iteration_index, so it's a parent DAG
				parallelForParentDAGs = append(parallelForParentDAGs, node)
				h.t.Logf("Found ParallelFor parent DAG: ID=%d, iteration_count=%d, state=%s",
					node.Execution.GetId(), iterationCount, (*node.Execution.LastKnownState).String())
			}
		}
	}

	return parallelForParentDAGs, parallelForIterationDAGs
}

// NestedDAGValidationContext holds the context for nested DAG validation
type NestedDAGValidationContext struct {
	NestedDAGs []*pb.Execution
}

// GetNestedDAGContext gets the complete context needed for nested DAG validation
func (h *DAGTestUtil) GetNestedDAGContext(runID string, testScenario string) *NestedDAGValidationContext {
	// Get recent DAG executions and context-specific executions
	recentDAGs := h.getRecentDAGExecutions()
	contextDAGs := h.getContextSpecificDAGExecutions(runID)

	// Merge and deduplicate DAG executions
	nestedDAGs := h.mergeDAGExecutions(recentDAGs, contextDAGs)

	return &NestedDAGValidationContext{
		NestedDAGs: nestedDAGs,
	}
}

// getRecentDAGExecutions retrieves recent DAG executions from the system
func (h *DAGTestUtil) getRecentDAGExecutions() []*pb.Execution {
	// Get all DAG executions in the system
	allDAGExecutions := h.GetAllDAGExecutions()

	// Filter DAG executions that are recent (within last 5 minutes)
	var recentDAGs []*pb.Execution

	for _, execution := range allDAGExecutions {
		// Log all DAG executions for debugging
		h.LogExecutionSummary(execution, "Examining DAG execution")

		// Include DAG executions that are recent as potentially related
		if h.IsRecentExecution(execution) {
			recentDAGs = append(recentDAGs, execution)
			h.t.Logf("Including recent DAG execution ID=%d", execution.GetId())
		}
	}

	return recentDAGs
}

// getContextSpecificDAGExecutions retrieves DAG executions from the specific run context
func (h *DAGTestUtil) getContextSpecificDAGExecutions(runID string) []*pb.Execution {
	// Get all executions for the run
	executions := h.GetExecutionsForRun(runID)

	// Filter for DAG executions only
	contextDAGs := h.FilterDAGExecutions(executions)
	for _, execution := range contextDAGs {
		h.t.Logf("Adding context-specific DAG execution ID=%d", execution.GetId())
	}

	return contextDAGs
}

// mergeDAGExecutions merges and deduplicates DAG executions from different sources
func (h *DAGTestUtil) mergeDAGExecutions(recentDAGs, contextDAGs []*pb.Execution) []*pb.Execution {
	// Start with recent DAGs
	merged := make([]*pb.Execution, len(recentDAGs))
	copy(merged, recentDAGs)

	// Add context DAGs that aren't already present
	for _, contextDAG := range contextDAGs {
		found := false
		for _, existing := range merged {
			if existing.GetId() == contextDAG.GetId() {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, contextDAG)
		}
	}

	return merged
}
