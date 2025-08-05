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

## Success Criteria

- [ ] ParallelFor DAGs complete when all iterations finish
- [ ] Conditional DAGs complete when branches finish (including 0-task cases)  
- [ ] Nested DAGs complete properly with correct task counting across hierarchy levels
- [ ] Status propagates correctly up DAG hierarchies
- [ ] No regression in existing functionality
- [ ] Pipeline runs complete instead of hanging indefinitely
- [ ] All three integration tests pass consistently