# DAG Status Propagation Issue - GitHub Issue #11979

## Problem Summary

Kubeflow Pipelines v2 has a critical bug where DAG (Directed Acyclic Graph) executions get stuck in `RUNNING` state and never transition to `COMPLETE`, causing pipeline runs to hang indefinitely. This affects two main constructs:

1. **ParallelFor Loops**: DAGs representing parallel iterations do not complete even when all iterations finish
2. **Conditional Constructs**: DAGs representing if/else branches do not complete, especially when conditions evaluate to false (resulting in 0 executed tasks)

## GitHub Issue

**Link**: https://github.com/kubeflow/pipelines/issues/11979

**Core Issue**: DAG status propagation failures in Kubeflow Pipelines v2 backend for ParallelFor and Conditional constructs, causing pipeline runs to hang in RUNNING state instead of completing.

## Observed Symptoms

### Integration Test Failures
- `/backend/test/integration/dag_status_parallel_for_test.go` - Tests fail because ParallelFor DAGs remain in RUNNING state
- `/backend/test/integration/dag_status_conditional_test.go` - Tests fail because Conditional DAGs remain in RUNNING state  
- `/backend/test/integration/dag_status_nested_test.go` - Tests fail because nested DAG structures don't complete properly

### Real-World Impact
- Pipeline runs hang indefinitely in RUNNING state
- Users cannot determine if pipelines have actually completed
- No automatic cleanup or resource release
- Affects both simple and complex pipeline structures

## Test Evidence

### ParallelFor Test Failures
From `dag_status_parallel_for_test.go`, we expect:
- `iteration_count=3, total_dag_tasks=3` ✅ (counting works)
- DAG state transitions from RUNNING → COMPLETE ❌ (stuck in RUNNING)

### Conditional Test Failures  
From `dag_status_conditional_test.go`, we expect:
- Simple If (false): 0 branches execute, DAG should complete ❌ (stuck in RUNNING)
- Simple If (true): 1 branch executes, DAG should complete ❌ (stuck in RUNNING)
- Complex conditionals: Executed branches complete, DAG should complete ❌ (stuck in RUNNING)

## Architecture Context

### Key Components
- **MLMD (ML Metadata)**: Stores execution state and properties
- **Persistence Agent**: Monitors workflow state and updates MLMD
- **DAG Driver**: Creates DAG executions and sets initial properties
- **API Server**: Orchestrates pipeline execution

### DAG Hierarchy
```
Pipeline Run
├── Root DAG (system.DAGExecution)
├── ParallelFor Parent DAG (system.DAGExecution)
│   ├── ParallelFor Iteration DAG 0 (system.DAGExecution)  
│   ├── ParallelFor Iteration DAG 1 (system.DAGExecution)
│   └── ParallelFor Iteration DAG 2 (system.DAGExecution)
└── Conditional DAG (system.DAGExecution)
    ├── Container Task 1 (system.ContainerExecution)
    └── Container Task 2 (system.ContainerExecution)
```

### Current DAG Completion Logic Location
Primary logic appears to be in `/backend/src/v2/metadata/client.go` in the `UpdateDAGExecutionsState` method.

## Development Environment

### Build Process
```bash
# Build images
KFP_REPO=/Users/hbelmiro/dev/opendatahub-io/data-science-pipelines TAG=latest docker buildx bake --push -f /Users/hbelmiro/dev/hbelmiro/kfp-parallel-image-builder/docker-bake.hcl

# Deploy to Kind cluster
h-kfp-undeploy && h-kfp-deploy

# Run integration tests
go test -v -timeout 10m -tags=integration -args -runIntegrationTests -isDevMode
```

### Test Strategy for Investigation
1. **Start with Integration Tests**: Run failing tests to understand current behavior
2. **Create Unit Tests**: Build focused unit tests for faster iteration (located in `dag_completion_test.go`)
3. **Verify Unit Tests**: Before running slow integration tests, ensure unit tests are comprehensive and pass
4. **Root Cause Analysis**: Identify why DAGs remain in RUNNING state
5. **Incremental Fixes**: Test changes against unit tests first, then integration tests

## Investigation Questions

1. **Where is DAG completion logic?** What determines when a DAG transitions from RUNNING → COMPLETE?
2. **How are ParallelFor DAGs supposed to complete?** What should trigger completion for parent vs iteration DAGs?
3. **How are Conditional DAGs supposed to complete?** What happens when 0, 1, or multiple branches execute?
4. **Status Propagation**: How should child DAG completion affect parent DAG state?
5. **Task Counting**: How is `total_dag_tasks` supposed to be calculated for different DAG types?

## Test Files Detailed Analysis

### ParallelFor Test (`dag_status_parallel_for_test.go`)
**Purpose**: Validates that ParallelFor DAG executions complete properly when all iterations finish.

**Key Scenarios**:
- Creates a ParallelFor construct with 3 iterations 
- Each iteration should run independently and complete
- Parent ParallelFor DAG should complete when all child iteration DAGs finish
- Tests `iteration_count=3, total_dag_tasks=3` calculation correctness
- **Current Bug**: DAGs remain stuck in RUNNING state instead of transitioning to COMPLETE

### Conditional Test (`dag_status_conditional_test.go`)  
**Purpose**: Validates that Conditional DAG executions complete properly for different branch scenarios.

**Key Scenarios**:
- **Simple If (true)**: Condition evaluates to true, if-branch executes, DAG should complete
- **Simple If (false)**: Condition evaluates to false, no branches execute, DAG should complete with 0 tasks
- **If/Else (true)**: Condition true, if-branch executes, else-branch skipped, DAG completes  
- **If/Else (false)**: Condition false, if-branch skipped, else-branch executes, DAG completes
- **Complex conditionals**: Multiple branches (if/elif/else), only executed branches count toward completion
- **Current Bug**: DAGs remain stuck in RUNNING state regardless of branch execution outcomes

### Nested Test (`dag_status_nested_test.go`)
**Purpose**: Validates that nested DAG structures (pipelines within pipelines) update status correctly across hierarchy levels.

**Key Scenarios**:
- **Simple Nested**: Parent pipeline contains child pipeline, both should complete properly
- **Nested ParallelFor**: Parent pipeline with nested ParallelFor constructs, completion should propagate up
- **Nested Conditional**: Parent pipeline with nested conditional constructs, status should update correctly  
- **Deep Nesting**: Multiple levels of nesting, status propagation should work through all levels
- **Current Bug**: Parent DAGs don't account for nested child pipeline tasks in `total_dag_tasks` calculation, causing completion logic failures

**Expected Behavior**: 
- Child pipeline DAGs complete correctly (have proper task counting)
- Parent DAGs should include nested child pipeline tasks in their completion calculations
- Status updates should propagate up the DAG hierarchy when child structures complete
- Test expects parent DAGs to have `total_dag_tasks >= 5` (parent tasks + child pipeline tasks)

## Current Progress (as of 2025-01-05)

### ✅ **Major Fixes Implemented**
**Location**: `/backend/src/v2/metadata/client.go` in `UpdateDAGExecutionsState()` method (lines 776-929)

1. **Enhanced DAG Completion Logic**:
   - **Conditional DAG detection**: `isConditionalDAG()` function (lines 979-1007)
   - **ParallelFor logic**: Separate handling for iteration vs parent DAGs (lines 854-886)
   - **Universal completion rule**: DAGs with no tasks and nothing running complete immediately (lines 858-861)
   - **Status propagation**: `propagateDAGStateUp()` method for recursive hierarchy updates (lines 931-975)

2. **Task Counting Fixes**:
   - **Conditional adjustment**: Lines 819-842 adjust `total_dag_tasks` for executed branches only
   - **ParallelFor parent completion**: Based on child DAG completion count, not container tasks

3. **Comprehensive Testing**:
   - **Unit tests**: 23 scenarios in `/backend/src/v2/metadata/dag_completion_test.go` ✅ **ALL PASSING**
   - **Integration test infrastructure**: Fully working with proper port forwarding setup

### ⚠️ **Remaining Issues** 
**Status**: Partial fixes working, edge cases need refinement

1. **Conditional DAG Task Counting**:
   - ✅ **Working**: Simple conditional DAGs with 0 executed branches
   - ❌ **Broken**: Conditional DAGs with 1+ executed branches show `total_dag_tasks=0` instead of correct count
   - **Root cause**: Task counting adjustment may not be persisting to MLMD correctly

2. **ParallelFor Parent DAG Completion**:
   - ✅ **Working**: Individual iteration DAGs complete correctly  
   - ❌ **Broken**: Parent DAGs remain stuck in RUNNING state even when all child DAGs complete
   - **Root cause**: Parent completion logic not correctly detecting child DAG states

## Next Phase Implementation Plan

### **Phase 1: Fix Conditional DAG Task Counting** (High Priority)
**Issue**: DAGs complete but `total_dag_tasks=0` when should be 1+ for executed branches

#### **Root Cause Analysis**
From test output: `task_name:string_value:""` - DAG task names are empty, breaking conditional detection.

**Key Insight**: We can rely on task names IF they are backend-controlled (not user-controlled):
- ✅ **Backend-controlled names**: DAG task names, system-generated names like `"for-loop-1"`, `"condition-dag"`
- ❌ **User-controlled names**: Pipeline names, component display names, user parameters

**Current Problem**: The `isConditionalDAG()` function (lines 979-1007) relies on task name patterns like `"condition-"`, `"-condition"`, but task names are empty in test output.

#### **Investigation Required**
**Primary Question**: Are DAG task names supposed to be set by the backend but aren't (bug), or are they intentionally empty (design)?

**Tasks**:
1. **Audit DAG Creation Logic** (30 min)
   - Find where DAG executions are created in the backend
   - Check if task names should be set but aren't being assigned
   - Identify backend naming standards for different DAG types
   ```bash
   # Search commands
   grep -r "CreateExecution.*DAG" backend/src/v2/
   grep -A 10 -B 5 "TaskName.*=.*" backend/src/v2/
   find backend/src/v2 -name "*.go" -exec grep -l "condition\|conditional" {} \;
   ```

2. **Choose Detection Strategy** (15 min)
   - **Option A**: If names should be set → Fix the name generation bug
   - **Option B**: If names are intentionally empty → Use alternative backend properties
   
3. **Expected Backend Task Name Patterns** (if Option A):
   ```go
   // ParallelFor DAGs
   "for-loop-{iteration-index}"     // For iteration DAGs
   "parallel-for-{component-name}"  // For parent DAGs
   
   // Conditional DAGs  
   "condition-{component-name}"     // For conditional DAGs
   "if-branch-{component-name}"     // For if branches
   "else-branch-{component-name}"   // For else branches
   
   // Standard DAGs
   "dag-{component-name}"           // For regular DAGs
   ```

4. **Alternative Detection Approaches** (if Option B):
   - **Use DAG properties**: Check for `condition_result`, `conditional_task` properties
   - **Use execution patterns**: Detect CANCELED tasks (non-executed branches)
   - **Use hierarchy relationships**: Parent-child DAG relationships
   - **Universal dynamic counting**: Adjust `total_dag_tasks` based on actual executed tasks

#### **Implementation Strategy**
1. **Audit Phase**: Determine if empty task names are a bug or intentional
2. **Fix Phase**: Either fix task name generation OR implement robust alternative detection
3. **Test Phase**: Validate detection works for all conditional scenarios

### **Phase 2: Fix ParallelFor Parent DAG Completion** (High Priority)  
**Issue**: Parent DAGs remain RUNNING even when all child iteration DAGs complete

**Tasks**:
1. **Debug ParallelFor parent completion logic**
   - Check `isParallelForParentDAG()` function in `client.go:1017-1023`
   - Review parent completion logic in lines 870-886
   
2. **Verify child DAG state detection**
   - Ensure parent DAGs correctly count completed child DAG executions
   - Check if `task.GetType() == "system.DAGExecution"` is working properly
   
3. **Test parent-child relationship queries**
   - Verify `GetExecutionsInDAG()` returns child DAGs for parent DAGs
   - May need to adjust filtering logic

### **Phase 3: Comprehensive Testing** (Medium Priority)
**Tasks**:
1. **Run focused tests** after each fix:
   ```bash
   # Test conditionals
   go test -run TestDAGStatusConditional/TestComplexConditional
   
   # Test ParallelFor  
   go test -run TestDAGStatusParallelFor/TestSimpleParallelForSuccess
   ```

2. **Full regression testing**:
   ```bash
   # All DAG status tests
   go test -run TestDAGStatus
   ```

3. **Verify unit tests still pass**:
   ```bash
   cd backend/src/v2/metadata && go test -run TestDAGCompletionLogic
   ```

## Implementation Strategy

### **Development Workflow**
1. **Build images with changes**:
   ```bash
   KFP_REPO=/Users/hbelmiro/dev/opendatahub-io/data-science-pipelines TAG=latest docker buildx bake --push -f /Users/hbelmiro/dev/hbelmiro/kfp-parallel-image-builder/docker-bake.hcl
   ```

2. **Deploy to Kind cluster**:
   ```bash
   h-kfp-undeploy && h-kfp-deploy
   ```

3. **Setup port forwarding**:
   ```bash
   nohup kubectl port-forward -n kubeflow svc/ml-pipeline 8888:8888 > /dev/null 2>&1 &
   nohup kubectl port-forward -n kubeflow svc/metadata-grpc-service 8080:8080 > /dev/null 2>&1 &
   ```

4. **Run targeted tests**:
   ```bash
   cd backend/test/integration
   go test -v -timeout 10m -tags=integration -run TestDAGStatusConditional -args -runIntegrationTests -isDevMode
   ```

## Success Criteria

- [x] Unit tests comprehensive and passing
- [x] Integration test infrastructure working  
- [x] Basic DAG completion logic implemented
- [x] Status propagation framework in place
- [ ] Conditional DAGs complete when branches finish (including 0-task cases)  
- [ ] ParallelFor DAGs complete when all iterations finish
- [ ] Nested DAGs complete properly with correct task counting across hierarchy levels
- [ ] Status propagates correctly up DAG hierarchies
- [ ] No regression in existing functionality
- [ ] Pipeline runs complete instead of hanging indefinitely
- [ ] All three integration tests pass consistently