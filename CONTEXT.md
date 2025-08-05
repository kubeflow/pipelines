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

### ✅ **Major Breakthrough - Universal Detection Implemented** 
**Status**: Core infrastructure working, one edge case remaining

#### **Phase 1 Complete - Universal Detection Success**
**Implemented**: Replaced fragile task name detection with robust universal approach that works regardless of naming.

**Key Changes Made**:
1. **Replaced `isConditionalDAG()`** with `shouldApplyDynamicTaskCounting()` in `/backend/src/v2/metadata/client.go:979-1022`
2. **Universal Detection Logic**:
   - Skips ParallelFor DAGs (they have specialized logic)
   - Detects canceled tasks (non-executed branches)
   - Applies dynamic counting as safe default
   - No dependency on task names or user-controlled properties

3. **Simplified Completion Logic**:
   - Removed conditional-specific completion branch (lines 893-901)
   - Universal rule handles empty DAGs: `totalDagTasks == 0 && runningTasks == 0 → COMPLETE`
   - Standard logic handles dynamic counting results

#### **Test Results**
1. **✅ WORKING PERFECTLY**: 
   - **Simple conditionals with 0 executed branches**: `TestSimpleIfFalse` passes ✅
   - **Universal completion rule**: Empty DAGs complete immediately ✅
   - **Unit tests**: All 23 scenarios still passing ✅

2. **⚠️ ONE REMAINING ISSUE**:
   - **Conditional DAGs with executed branches**: Show `total_dag_tasks=0` instead of correct count
   - **Symptoms**: DAGs complete correctly (✅) but display wrong task count (❌)
   - **Example**: `expected_executed_branches=1, total_dag_tasks=0` should be `total_dag_tasks=1`

#### **Root Cause of Remaining Issue**
The dynamic task counting logic (lines 827-830) calculates the correct value but it's not being persisted or retrieved properly:
```go
if actualExecutedTasks > 0 {
    totalDagTasks = int64(actualExecutedTasks)  // ← Calculated correctly
    // But test shows total_dag_tasks=0 in MLMD
}
```

#### **Next Phase Required**
**Phase 2**: Fix the persistence/retrieval of updated `total_dag_tasks` values for conditional DAGs with executed branches.

## Next Phase Implementation Plan

### **Phase 1: Fix Conditional DAG Task Counting** ✅ **COMPLETED**
**Completed**: Universal detection implemented successfully. No longer depends on task names.

**What was accomplished**:
- ✅ Replaced fragile task name detection with universal approach  
- ✅ Empty conditional DAGs now complete correctly (`TestSimpleIfFalse` passes)
- ✅ Universal completion rule working
- ✅ All unit tests still passing

### **Phase 2: Fix Conditional Task Count Persistence** (High Priority) 🚧 **CURRENT**
**Issue**: Dynamic task counting calculates correct values but they don't persist to MLMD correctly

**Current Problem Analysis**:
- ✅ **DAG Completion**: Conditional DAGs complete correctly (some reach `COMPLETE` state)
- ❌ **Task Counting**: Shows `total_dag_tasks=0` instead of `expected_executed_branches=1`
- **Key Observation**: The dynamic task counting logic isn't finding executed container tasks in conditional DAGs

#### **Detailed Investigation Plan**

**Task 1: Debug Task Finding Logic** (30 min)
**Hypothesis**: `GetExecutionsInDAG()` may not be finding executed container tasks in conditional DAGs

**Steps**:
1. **Add comprehensive debug logging** to trace task counting flow:
   ```go
   glog.Infof("DAG %d: shouldApplyDynamic=%v, found %d tasks", dagID, shouldApplyDynamic, len(tasks))
   for taskName, task := range tasks {
       taskType := task.GetType()
       taskState := task.GetExecution().LastKnownState.String()
       glog.Infof("DAG %d: Task %s, type=%s, state=%s", dagID, taskName, taskType, taskState)
   }
   glog.Infof("DAG %d: actualExecutedTasks=%d, actualRunningTasks=%d", dagID, actualExecutedTasks, actualRunningTasks)
   ```

2. **Test with simple conditional**: `go test -run TestDAGStatusConditional/TestSimpleIfTrue`
3. **Verify task retrieval**: Check if container tasks from executed conditional branches are found

**Task 2: Debug MLMD Persistence** (30 min)
**Hypothesis**: Values calculated correctly but not persisted or retrieved properly

**Steps**:
1. **Add persistence debugging**:
   ```go
   // Before updating
   glog.Infof("DAG %d: Before update - totalDagTasks=%d", dagID, totalDagTasks)
   
   // After updating custom properties
   if shouldApplyDynamic && actualExecutedTasks > 0 {
       storedValue := dag.Execution.execution.CustomProperties["total_dag_tasks"].GetIntValue()
       glog.Infof("DAG %d: After update - stored value=%d", dagID, storedValue)
   }
   ```

2. **Check persistence across calls**: Verify value persists and test reads updated value

**Task 3: Fix Root Cause** (45 min)
**Based on findings, implement appropriate fix**:

- **Scenario A - Tasks Not Found**: Adjust `GetExecutionsInDAG()` query for conditional branches
- **Scenario B - Tasks Found But Not Counted**: Fix counting logic in lines 823-824
- **Scenario C - Counted But Not Persisted**: Add explicit `PutExecution` call:
  ```go
  if shouldApplyDynamic && stateChanged {
      _, err := c.svc.PutExecution(ctx, &pb.PutExecutionRequest{
          Execution: dag.Execution.execution,
      })
  }
  ```
- **Scenario D - Timing Issue**: Fix race condition or caching issue

**Task 4: Validate Fix** (30 min)
1. **Test single case**: `go test -run TestDAGStatusConditional/TestSimpleIfTrue`
2. **Verify both completion AND counting**: DAG reaches `COMPLETE` + correct `total_dag_tasks`
3. **No regression**: `TestSimpleIfFalse` continues to pass

#### **Implementation Strategy**
- **Phase 2A**: Debug & Investigate (1 hour)
- **Phase 2B**: Implement targeted fix (45 min)  
- **Phase 2C**: Validate (30 min)
- **Total**: ~2.25 hours

#### **Success Criteria for Phase 2**
- [ ] `TestSimpleIfTrue` passes with correct `total_dag_tasks=1`
- [ ] `TestSimpleIfFalse` continues to pass with `total_dag_tasks=0`
- [ ] Complex conditional scenarios show correct executed branch counts
- [ ] No regression in universal completion rule or ParallelFor logic

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
- [x] Universal detection system implemented (no dependency on task names)
- [x] **Conditional DAGs with 0 branches complete correctly** (`TestSimpleIfFalse` ✅)
- [x] **Universal completion rule working** (empty DAGs complete immediately)
- [ ] Conditional DAGs with executed branches show correct task count (Phase 2 target)
- [ ] ParallelFor DAGs complete when all iterations finish  
- [ ] Nested DAGs complete properly with correct task counting across hierarchy levels
- [ ] Status propagates correctly up DAG hierarchies
- [ ] No regression in existing functionality
- [ ] Pipeline runs complete instead of hanging indefinitely
- [ ] All three integration tests pass consistently

## Current Status: 🎯 **Major Progress Made**
- **Phase 1**: ✅ Universal detection system working
- **Phase 2**: 🚧 Fixing task count persistence (final edge case)
- **Phase 3**: ⏳ ParallelFor parent completion logic