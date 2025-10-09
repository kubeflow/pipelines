# Test Plan: KFP MLMD Removal

## Document Information
- **Feature**: Complete MLMD (ML Metadata) Removal
- **Version**: v2.0 (Major Refactoring)
- **Date Created**: 2025-10-06
- **Repository**: kubeflow/pipelines
- **Branch**: Current development branch
- **Total Files Changed**: 235+ files

---

## Table of Contents

1. [Overview](#overview)
2. [Summary of Changes](#summary-of-changes)
3. [Impact Analysis](#impact-analysis)
4. [Test Environment Requirements](#test-environment-requirements)
5. [Test Categories](#test-categories)
   - [Unit Tests](#unit-tests)
   - [Integration Tests](#integration-tests)
   - [API Contract Tests](#api-contract-tests)
   - [Storage Layer Tests](#storage-layer-tests)
   - [Driver and Launcher Tests](#driver-and-launcher-tests)
   - [Conditional Logic Tests](#conditional-logic-tests)
   - [Security Tests](#security-tests)
   - [Performance Tests](#performance-tests)
   - [Regression Tests](#regression-tests)
   - [Migration Tests](#migration-tests)
6. [Success Criteria](#success-criteria)
7. [Risk Assessment](#risk-assessment)

---

## Overview

This test plan covers comprehensive testing for the major refactoring of Kubeflow Pipelines (KFP) that includes:

1. **Complete MLMD Removal**: Removing all ML Metadata (MLMD) client dependencies from the driver, launcher, and backend
2. **New Storage Architecture**: Introduction of artifact_store, task_store, and run_store without MLMD
3. **OneOf Conditional Branching**: Implementation of new conditional logic for pipeline branching
4. **Parameter/Artifact Resolution**: Complete rework of input/output resolution logic
5. **API Changes**: Updates to v2beta1 protobuf definitions and generated clients

### Architecture Changes

**Before**:
- MLMD client for artifact and execution tracking
- MLMD-based context management (Pipeline, PipelineRun)
- MLMD executions for all task types
- Custom property storage in MLMD

**After**:
- KFP v2beta1 API Server for all metadata operations
- Direct database storage via artifact_store, task_store, run_store
- Task-based execution tracking with explicit types
- Dedicated metrics table (no longer stored as artifacts)

---

## Summary of Changes

### Core Components Modified

#### API Layer (66 files)
- **v2alpha1 protobuf**: Added oneOf support to pipeline_spec.proto
- **v2beta1 protobuf**: New artifact.proto service, extensive run.proto updates
  - **ArtifactService**: New service with ListArtifacts, GetArtifact, CreateArtifact, ListArtifactTasks, CreateArtifactTask
  - **RunService**: Enhanced with CreateTask, UpdateTask, GetTask, ListTasks endpoints
  - **Artifact types**: TYPE_UNSPECIFIED, Artifact, Model, Dataset, HTML, Markdown, Metric, ClassificationMetric, SlicedClassificationMetric
  - **Task types**: ROOT_DAG, DAG, RUNTIME, LOOP, LOOP_ITERATION, CONDITION, CONDITION_BRANCH, EXITHANDLER
- **Generated clients**: Complete regeneration of Go and Python HTTP clients
- **New models**: IOParameter, IOArtifact, IOProducer, ArtifactTask, PipelineTaskDetail enhancements

#### Backend Storage (20 files)
- **New stores**: artifact_store.go, artifact_task_store.go (with comprehensive tests)
- **Updated stores**: task_store.go, run_store.go (MLMD removal)
- **Database schema**: New tables for artifacts and artifact_tasks
- **Client manager**: Removed MLMD client initialization

#### Backend API Server (15 files)
- **New server**: artifact_server.go (handles artifact CRUD and queries)
- **Updated servers**: run_server.go (added task endpoints), report_server.go (removed MLMD reporting)
- **Converters**: api_converter.go refactored for new task models
- **Utilities**: list_request_util.go for unified listing logic

#### V2 Driver (40+ files)
- **Complete refactoring**: driver.go, container.go, dag.go, root_dag.go
- **New resolver package**: artifacts.go, parameters.go, resolve.go, util.go
- **Removed**: resolve.go (old monolithic resolver - 1099 lines deleted)
- **Test data**: 6 new test pipelines for oneOf, nested dags, loops, parameters

#### V2 Compiler (10 files)
- **Argo compiler**: Updates for oneOf and conditional branches
- **DAG handling**: New logic for ConditionBranch task types
- **Test data**: New test cases for multiple parallel loops

#### Metadata Client (5 files)
- **v2/metadata/client.go**: Removed MLMD-specific methods, added KFP API client integration
- **Fake client**: Updated for testing without MLMD

### New Features Implemented

#### OneOf Conditional Branching
- Enables mutually exclusive outputs from conditional branches
- Example: `dsl.OneOf(output_a, output_b, output_c)` returns the output from the executed branch
- Compiler generates ConditionBranch and Condition task types
- Driver resolves oneOf outputs through artifact selectors

#### Nested Naming Conflict Resolution
- Handles deeply nested pipeline components with same names
- Uses scope-based resolution (e.g., `pipeline-b/pipeline-c/task-a`)
- Test case: `nested_naming_conflicts.py` with 3 levels of nesting

#### Parameter Iterator and Collection
- Supports iterating over parameter lists from task outputs
- Collects outputs from loop iterations
- Example: `dsl.Collected()` gathers all iteration outputs into a list

#### Loop Artifact Passing
- Artifacts from loop iterations can be collected
- Supports both raw iterators and input parameter iterators
- Proper parent-child task relationships maintained

---

## Impact Analysis

### Criticality: CRITICAL

This is a **breaking architectural change** that affects:

1. **All pipeline executions** - Every pipeline run now uses new task and artifact storage
2. **Client compatibility** - Generated clients have breaking API changes
3. **Database schema** - Requires migration script for existing deployments
4. **Authentication** - New RBAC requirements for Driver/Launcher
5. **Frontend** - UI must transition from MLMD queries to Task API queries
6. **Caching** - Cache mechanism completely reworked
7. **Metrics** - Metrics no longer stored as artifacts (moved to dedicated table)

### Components with High Risk

| Component | Risk Level | Reason | Mitigation |
|-----------|-----------|---------|------------|
| Task Storage | **CRITICAL** | Complete replacement of MLMD-based execution tracking | Comprehensive storage layer tests + migration validation |
| Artifact Storage | **CRITICAL** | New artifact and artifact_task tables | Full CRUD test coverage + artifact linking tests |
| Driver | **CRITICAL** | 1099 lines deleted, complete resolver rewrite | Extensive unit tests + integration tests with real pipelines |
| Caching | **HIGH** | Different fingerprint storage and lookup mechanism | Cache hit/miss tests + performance benchmarks |
| Parameter/Artifact Resolution | **HIGH** | Complete rewrite of input resolution logic | Resolution tests for all parameter/artifact types |
| OneOf Logic | **HIGH** | New feature with complex conditional evaluation | Edge case testing + integration tests |
| Authentication | **MEDIUM** | New API endpoints need RBAC | Security tests with multi-tenant scenarios |
| Frontend | **MEDIUM** | UI must query new APIs | UI integration tests (manual verification) |

---

## Test Environment Requirements

### Development Environment
- **Go Version**: 1.21+
- **Python Version**: 3.9+
- **Database**: MySQL 8.0 or compatible
- **Container Runtime**: Docker or Podman
- **Kubernetes**: Kind cluster (v1.29.2 or v1.31.0)

### Test Cluster Configurations

#### 1. Local Development Cluster
```yaml
Kubernetes: Kind v1.29.2
KFP Mode: Standalone
Database: MySQL 8.0
Storage: MinIO (local S3-compatible)
RBAC: Disabled (for quick iteration)
Purpose: Unit test validation, quick smoke tests
```

#### 2. Standard KFP Cluster
```yaml
Kubernetes: Kind v1.31.0
KFP Mode: Standalone with RBAC
Database: MySQL 8.0
Storage: MinIO
RBAC: Enabled
Multi-tenancy: Single namespace
Purpose: Main integration testing, regression tests
```

#### 3. Multi-Tenant Cluster
```yaml
Kubernetes: Kind v1.31.0
KFP Mode: Multi-tenant
Database: MySQL 8.0
Storage: MinIO with namespace isolation
RBAC: Enabled with ServiceAccount tokens
Multi-tenancy: Multiple namespaces
Purpose: Security testing, RBAC validation
```

#### 4. Performance Testing Cluster
```yaml
Kubernetes: Kind v1.31.0 (or GKE/EKS for realistic perf)
KFP Mode: Standalone
Database: MySQL 8.0 (optimized)
Storage: MinIO (or cloud storage)
Resources: Higher CPU/memory allocation
Purpose: Load testing, performance regression
```

### Required Services
- **KFP API Server**: Running with new v2beta1 endpoints
- **MySQL Database**: With migrated schema (artifacts, artifact_tasks, tasks tables)
- **Object Storage**: MinIO or S3-compatible service
- **Kubernetes Cluster**: For end-to-end pipeline execution

### Test Data Requirements
- **Sample Pipelines**: 10+ test pipelines covering various scenarios
- **Migration Data**: Sample MLMD database dump for migration testing
- **Performance Baselines**: Pre-refactor performance metrics for comparison

---

## Test Categories

## Unit Tests

### Storage Layer Tests

#### Artifact Store Tests
**Location**: `backend/src/apiserver/storage/artifact_store_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Create artifact with valid data | <ol><li>Initialize artifact store with test DB</li><li>Create artifact with name, type, uri, metadata</li><li>Verify artifact is persisted</li></ol> | <ol><li>Artifact UUID is generated</li><li>Artifact is stored in DB</li><li>CreatedAtInSec and LastUpdateInSec are set</li></ol> | Yes |
| Get artifact by UUID | <ol><li>Create an artifact</li><li>Retrieve artifact by UUID</li><li>Verify all fields match</li></ol> | <ol><li>Artifact is retrieved successfully</li><li>All fields (name, type, uri, metadata) match original</li></ol> | Yes |
| List artifacts with filtering | <ol><li>Create multiple artifacts in different namespaces</li><li>List artifacts with namespace filter</li><li>Verify only matching artifacts are returned</li></ol> | <ol><li>Only artifacts in specified namespace are returned</li><li>Pagination works correctly</li><li>Total count is accurate</li></ol> | Yes |
| Create artifact with invalid metadata | <ol><li>Attempt to create artifact with malformed JSON metadata</li><li>Verify error is returned</li></ol> | <ol><li>Error is returned with clear message</li><li>No artifact is created in DB</li></ol> | Yes |
| List artifacts with pagination | <ol><li>Create 50 artifacts</li><li>List with page_size=10</li><li>Verify pagination token works</li></ol> | <ol><li>10 artifacts returned per page</li><li>next_page_token allows fetching next page</li><li>All 50 artifacts can be retrieved</li></ol> | Yes |

#### Artifact Task Store Tests
**Location**: `backend/src/apiserver/storage/artifact_task_store_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Create artifact task linking artifact to task | <ol><li>Create an artifact and a task</li><li>Create artifact_task entry linking them</li><li>Verify relationship is stored</li></ol> | <ol><li>ArtifactTask UUID is generated</li><li>Foreign keys to artifact and task are valid</li><li>IOType is correctly stored</li></ol> | Yes |
| List artifact tasks by task ID | <ol><li>Create task with multiple input/output artifacts</li><li>Create artifact_tasks for each</li><li>List artifact_tasks by task_id</li></ol> | <ol><li>All artifact_tasks for task are returned</li><li>Input and output artifacts are distinguishable</li></ol> | Yes |
| Batch create artifact tasks | <ol><li>Prepare list of artifact_task entries</li><li>Call BatchCreateArtifactTasks</li><li>Verify all are created</li></ol> | <ol><li>All artifact_tasks are created in single transaction</li><li>Rollback occurs if any fails</li></ol> | Yes |
| Create artifact task with iteration index | <ol><li>Create loop task</li><li>Create artifact_task with iteration_index=2</li><li>Verify iteration_index is stored</li></ol> | <ol><li>Iteration index is correctly stored</li><li>Can query by iteration index</li></ol> | Yes |
| Filter artifact tasks by IO type | <ol><li>Create tasks with INPUT, OUTPUT, ITERATOR_OUTPUT types</li><li>Query with io_type filter</li></ol> | <ol><li>Only matching io_type artifact_tasks returned</li></ol> | Yes |

#### Task Store Tests
**Location**: `backend/src/apiserver/storage/task_store_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Create runtime task | <ol><li>Initialize task store</li><li>Create task with type=RUNTIME</li><li>Verify task is persisted with all fields</li></ol> | <ol><li>Task UUID generated</li><li>Task type, status, timestamps correctly stored</li><li>Input/output parameters serialized correctly</li></ol> | Yes |
| Create DAG task with parent reference | <ol><li>Create parent task</li><li>Create child DAG task with parent_task_uuid</li><li>Verify parent-child relationship</li></ol> | <ol><li>ParentTaskUUID is stored correctly</li><li>Can query child tasks by parent ID</li></ol> | Yes |
| Update task status and outputs | <ol><li>Create task with RUNNING status</li><li>Update status to SUCCEEDED with output parameters</li><li>Verify update persisted</li></ol> | <ol><li>Status updated to SUCCEEDED</li><li>Output parameters stored correctly</li><li>FinishedInSec timestamp set</li></ol> | Yes |
| List tasks by parent ID | <ol><li>Create parent task with 5 child tasks</li><li>Call ListTasks with parent_id filter</li></ol> | <ol><li>All 5 child tasks returned</li><li>No tasks from other parents returned</li></ol> | Yes |
| Store and retrieve cache fingerprint | <ol><li>Create task with cache fingerprint</li><li>Query by fingerprint</li><li>Verify cache hit detection works</li></ol> | <ol><li>Fingerprint stored correctly</li><li>Can find task by fingerprint</li></ol> | Yes |
| Create task with loop iteration index | <ol><li>Create LOOP_ITERATION task with iteration_index=3</li><li>Verify iteration index stored in TypeAttrs</li></ol> | <ol><li>TypeAttrs JSON contains iteration_index</li><li>Can query tasks by iteration index</li></ol> | Yes |
| Get child tasks for DAG | <ol><li>Create DAG task</li><li>Create 3 child runtime tasks</li><li>Call GetChildTasks</li></ol> | <ol><li>All 3 child tasks returned</li><li>Tasks ordered by creation time</li></ol> | Yes |

#### Run Store Tests
**Location**: `backend/src/apiserver/storage/run_store_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Create run with runtime config | <ol><li>Create run with parameter values</li><li>Verify run stored with tasks relationship</li></ol> | <ol><li>Run UUID generated</li><li>Runtime config stored correctly</li><li>Can retrieve run with tasks</li></ol> | Yes |
| Update run state | <ol><li>Create run in RUNNING state</li><li>Update to SUCCEEDED state</li><li>Verify state change persisted</li></ol> | <ol><li>State updated correctly</li><li>FinishedAtInSec set</li><li>State history updated</li></ol> | Yes |
| List runs with filtering | <ol><li>Create multiple runs in different namespaces</li><li>Query with namespace filter</li></ol> | <ol><li>Only matching runs returned</li><li>Pagination works</li></ol> | Yes |

### API Server Tests

#### Artifact Server Tests
**Location**: `backend/src/apiserver/server/artifact_server_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| CreateArtifact API endpoint | <ol><li>Call CreateArtifact gRPC endpoint</li><li>Verify artifact created in DB</li><li>Check response contains UUID</li></ol> | <ol><li>Artifact created successfully</li><li>Response contains all artifact fields</li></ol> | Yes |
| GetArtifact API endpoint | <ol><li>Create artifact</li><li>Call GetArtifact with UUID</li><li>Verify response matches</li></ol> | <ol><li>Artifact retrieved successfully</li><li>All fields match original</li></ol> | Yes |
| ListArtifacts with filtering | <ol><li>Create artifacts with different types</li><li>Call ListArtifacts with type filter</li></ol> | <ol><li>Only matching artifacts returned</li><li>Pagination works correctly</li></ol> | Yes |
| CreateArtifactTask API endpoint | <ol><li>Create artifact and task</li><li>Call CreateArtifactTask to link them</li></ol> | <ol><li>ArtifactTask created successfully</li><li>Link can be queried</li></ol> | Yes |
| BatchCreateArtifactTasks API endpoint | <ol><li>Prepare multiple artifact_task requests</li><li>Call BatchCreateArtifactTasks</li></ol> | <ol><li>All tasks created atomically</li><li>Failure rolls back all</li></ol> | Yes |

#### Run Server Task Endpoints Tests
**Location**: `backend/src/apiserver/server/run_server_tasks_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| CreateTask API endpoint | <ol><li>Call CreateTask gRPC endpoint</li><li>Verify task created in DB</li></ol> | <ol><li>Task created with UUID</li><li>Response contains task details</li></ol> | Yes |
| GetTask API endpoint | <ol><li>Create task</li><li>Call GetTask with task_id</li></ol> | <ol><li>Task retrieved successfully</li><li>Includes inputs, outputs, status</li></ol> | Yes |
| UpdateTask API endpoint | <ol><li>Create task</li><li>Call UpdateTask to set status=SUCCEEDED</li></ol> | <ol><li>Task updated successfully</li><li>Timestamps updated</li></ol> | Yes |
| ListTasks with parent filter | <ol><li>Create parent task with children</li><li>Call ListTasks with parent_id filter</li></ol> | <ol><li>Only child tasks returned</li><li>Correct ordering</li></ol> | Yes |

#### API Converter Tests
**Location**: `backend/src/apiserver/server/api_converter_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Convert Task model to API PipelineTaskDetail | <ol><li>Create task model with all fields</li><li>Call ToApiTaskDetail converter</li><li>Verify all fields mapped correctly</li></ol> | <ol><li>All fields converted</li><li>JSON fields deserialized properly</li><li>Enums mapped correctly</li></ol> | Yes |
| Convert Artifact model to API Artifact | <ol><li>Create artifact model</li><li>Call ToApiArtifact converter</li></ol> | <ol><li>All fields converted correctly</li><li>Metadata JSON deserialized</li></ol> | Yes |
| Convert API request to Task model | <ol><li>Create API CreateTaskRequest</li><li>Convert to storage model</li></ol> | <ol><li>Fields mapped correctly</li><li>Validation applied</li></ol> | Yes |

### Driver Tests

#### Parameter Resolution Tests
**Location**: `backend/src/v2/driver/resolver/parameters.go` (with tests in `dag_test.go`)

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Resolve component input parameters | <ol><li>Create runtime config with parameters</li><li>Run driver to resolve component inputs</li><li>Verify ExecutorInput contains parameters</li></ol> | <ol><li>All component input parameters resolved</li><li>Types preserved (string, number, bool, list, struct)</li><li>IOType set to COMPONENT_INPUT</li></ol> | Yes |
| Resolve task output parameter reference | <ol><li>Create upstream task with output parameter</li><li>Create downstream task referencing it</li><li>Run driver to resolve input</li></ol> | <ol><li>Parameter value fetched from upstream task</li><li>IOType set to TASK_OUTPUT</li><li>Value matches upstream output</li></ol> | Yes |
| Resolve parameter iterator | <ol><li>Create task with parameter iterator (list)</li><li>Run driver for loop task</li><li>Verify iteration count calculated</li></ol> | <ol><li>Iteration count matches list length</li><li>Each iteration gets correct parameter value</li><li>IOType set to ITERATOR_INPUT</li></ol> | Yes |
| Resolve parameter from constant | <ol><li>Create task with constant parameter value</li><li>Run driver</li></ol> | <ol><li>Parameter resolved to constant value</li><li>IOType set to RUNTIME_VALUE_INPUT</li></ol> | Yes |
| Handle null parameter value | <ol><li>Create task with null parameter</li><li>Run driver</li></ol> | <ol><li>Null value handled correctly</li><li>No error thrown</li></ol> | Yes |

#### Artifact Resolution Tests
**Location**: `backend/src/v2/driver/resolver/artifacts.go` (with tests in `dag_test.go`)

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Resolve task output artifact reference | <ol><li>Create upstream task with output artifact</li><li>Create downstream task referencing it</li><li>Run driver to resolve input artifact</li></ol> | <ol><li>Artifact fetched from artifact_tasks table</li><li>URI, type, metadata populated</li><li>IOType set to TASK_OUTPUT</li></ol> | Yes |
| Resolve artifact iterator | <ol><li>Create task with artifact iterator (list)</li><li>Run driver for loop</li><li>Verify iteration count</li></ol> | <ol><li>Iteration count matches artifact list length</li><li>Each iteration gets single artifact</li></ol> | Yes |
| Resolve dsl.Collected() artifacts | <ol><li>Create loop with output artifacts</li><li>Create downstream task using dsl.Collected()</li><li>Run driver</li></ol> | <ol><li>All loop iteration artifacts collected</li><li>Artifact list passed to downstream task</li><li>IOType set to ITERATOR_OUTPUT</li></ol> | Yes |
| Resolve nested pipeline artifact output | <ol><li>Create nested pipeline with output artifact</li><li>Reference artifact in parent pipeline</li><li>Run driver</li></ol> | <ol><li>Artifact from nested pipeline resolved</li><li>Correct artifact retrieved via scope resolution</li></ol> | Yes |

#### Container Driver Tests
**Location**: `backend/src/v2/driver/container_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Container driver creates runtime task | <ol><li>Run container driver</li><li>Verify task created with type=RUNTIME</li><li>Check ExecutorInput generated</li></ol> | <ol><li>Task created in task_store</li><li>ExecutorInput contains resolved inputs</li><li>PodSpec generated correctly</li></ol> | Yes |
| Container driver handles caching | <ol><li>Run task twice with same inputs</li><li>Verify second run detects cache hit</li></ol> | <ol><li>First run stores fingerprint</li><li>Second run finds fingerprint</li><li>Cached task ID returned, no pod created</li></ol> | Yes |
| Container driver resolves all input types | <ol><li>Create task with parameters and artifacts</li><li>Run container driver</li></ol> | <ol><li>All parameters resolved</li><li>All artifacts resolved</li><li>ExecutorInput complete</li></ol> | Yes |

#### DAG Driver Tests
**Location**: `backend/src/v2/driver/dag_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| DAG driver creates DAG task | <ol><li>Run DAG driver for pipeline task</li><li>Verify task created with type=DAG</li></ol> | <ol><li>DAG task created</li><li>Parent-child relationships established</li></ol> | Yes |
| Loop DAG driver calculates iteration count | <ol><li>Run loop DAG with list input</li><li>Verify iteration count calculated</li></ol> | <ol><li>Iteration count matches list length</li><li>Loop task type=LOOP</li></ol> | Yes |
| Loop iteration driver sets iteration index | <ol><li>Run loop iteration driver for index 2</li><li>Verify iteration_index stored</li></ol> | <ol><li>Task type=LOOP_ITERATION</li><li>TypeAttrs contains iteration_index=2</li></ol> | Yes |
| Condition driver evaluates condition | <ol><li>Run condition driver with expression</li><li>Verify condition evaluated</li></ol> | <ol><li>Condition result stored</li><li>Task type=CONDITION</li></ol> | Yes |
| ConditionBranch driver for If/Elif/Else | <ol><li>Run condition branch driver</li><li>Verify branch task created</li></ol> | <ol><li>Task type=CONDITION_BRANCH</li><li>Child conditions grouped correctly</li></ol> | Yes |

#### Root DAG Driver Tests
**Location**: `backend/src/v2/driver/root_dag_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Root DAG creates root task | <ol><li>Run root DAG driver</li><li>Verify root task created</li></ol> | <ol><li>Root task type=ROOT_DAG</li><li>Runtime config stored</li><li>Run associated with task</li></ol> | Yes |
| Root DAG stores runtime parameters | <ol><li>Run with runtime parameter values</li><li>Verify parameters stored in root task</li></ol> | <ol><li>All runtime parameters stored</li><li>Can be retrieved by child tasks</li></ol> | Yes |

### Launcher Tests

#### Launcher V2 Tests
**Location**: `backend/src/v2/component/launcher_v2.go` (test file needs creation)

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Launcher updates task with output parameters | <ol><li>Run launcher that outputs parameters</li><li>Verify task updated with output values</li></ol> | <ol><li>Task outputs contain parameters</li><li>UpdateTask API called successfully</li></ol> | Partial |
| Launcher creates output artifacts | <ol><li>Run launcher that outputs artifacts</li><li>Verify artifacts created via CreateArtifact API</li></ol> | <ol><li>Artifacts created with correct URI</li><li>ArtifactTasks link artifacts to task</li></ol> | Partial |
| Launcher updates cache fingerprint on success | <ol><li>Run launcher to completion</li><li>Verify fingerprint stored in task</li></ol> | <ol><li>Task updated with fingerprint</li><li>Fingerprint only stored on success</li></ol> | Partial |
| Launcher uploads artifacts to object store | <ol><li>Run launcher with artifact outputs</li><li>Verify artifacts uploaded to S3/MinIO</li></ol> | <ol><li>Artifacts uploaded to correct URI</li><li>Artifact metadata contains URI</li></ol> | Partial |

### Compiler Tests

#### Argo Compiler Tests
**Location**: `backend/src/v2/compiler/argocompiler/argo_test.go`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Compile pipeline with oneOf | <ol><li>Compile pipeline using dsl.OneOf</li><li>Verify Argo workflow generated correctly</li></ol> | <ol><li>ConditionBranch tasks created</li><li>Artifact selectors configured for oneOf</li></ol> | Yes |
| Compile nested pipelines | <ol><li>Compile multi-level nested pipeline</li><li>Verify DAG tasks created for each level</li></ol> | <ol><li>Correct number of DAG tasks</li><li>Parent-child relationships correct</li></ol> | Yes |
| Compile loop with parameter iterator | <ol><li>Compile pipeline with for-loop over parameters</li><li>Verify loop tasks generated</li></ol> | <ol><li>LOOP and LOOP_ITERATION tasks in spec</li><li>Iterator configuration correct</li></ol> | Yes |

---

## Integration Tests

### End-to-End Pipeline Execution Tests

#### Basic Pipeline Execution
**Test Data**: `backend/src/v2/driver/test_data/componentInput_level_1_test.py.yaml`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Execute simple 2-task pipeline | <ol><li>Submit pipeline with 2 sequential tasks</li><li>Monitor execution via GetRun API</li><li>Verify both tasks complete successfully</li></ol> | <ol><li>Pipeline completes with SUCCEEDED state</li><li>Both tasks visible in run graph</li><li>Task outputs stored correctly</li></ol> | Yes |
| Execute pipeline with artifact passing | <ol><li>Submit pipeline where task B consumes artifact from task A</li><li>Verify artifact resolution</li></ol> | <ol><li>Artifact created by task A</li><li>Task B receives artifact as input</li><li>Artifact metadata correct</li></ol> | Yes |
| Execute pipeline with parameter passing | <ol><li>Submit pipeline with parameter flow</li><li>Verify parameter values propagated</li></ol> | <ol><li>Output parameters from task A</li><li>Input parameters to task B match</li></ol> | Yes |

#### Loop Pipeline Execution
**Test Data**: `backend/src/v2/driver/test_data/loop_collected_raw_Iterator.py.yaml`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Execute for-loop with raw list iterator | <ol><li>Submit pipeline with for-loop over [1,2,3]</li><li>Verify 3 iterations execute</li><li>Check artifacts collected</li></ol> | <ol><li>LOOP task created</li><li>3 LOOP_ITERATION tasks created</li><li>Each iteration completes</li><li>Artifacts collected correctly</li></ol> | Yes |
| Execute for-loop with parameter iterator | <ol><li>Submit pipeline with for-loop over task output list</li><li>Verify iterations match output length</li></ol> | <ol><li>Iteration count calculated correctly</li><li>Each iteration gets correct parameter value</li><li>Parameters collected via dsl.Collected()</li></ol> | Yes |
| Execute nested for-loops | <ol><li>Submit pipeline with loop inside loop</li><li>Verify correct nesting and iteration counts</li></ol> | <ol><li>Outer loop creates inner loop for each iteration</li><li>Total iterations = outer × inner</li><li>Parent-child relationships correct</li></ol> | Partial |

#### Conditional Pipeline Execution
**Test Data**: `backend/src/v2/driver/test_data/oneof_simple.yaml`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Execute If/Elif/Else pipeline with If branch taken | <ol><li>Submit pipeline with condition evaluating to true for If</li><li>Verify only If branch executes</li></ol> | <ol><li>CONDITION_BRANCH task created</li><li>CONDITION task for If created and succeeds</li><li>Elif and Else conditions not executed</li><li>Correct branch artifact output</li></ol> | Yes |
| Execute If/Elif/Else pipeline with Elif branch taken | <ol><li>Submit pipeline with condition true for Elif</li><li>Verify only Elif branch executes</li></ol> | <ol><li>If condition fails</li><li>Elif condition succeeds</li><li>Elif branch tasks execute</li><li>OneOf resolves to Elif output</li></ol> | Yes |
| Execute If/Elif/Else with Else branch taken | <ol><li>Submit pipeline with all conditions false</li><li>Verify Else branch executes</li></ol> | <ol><li>If and Elif fail</li><li>Else branch executes as default</li></ol> | Yes |
| Execute pipeline with dsl.OneOf() output | <ol><li>Submit pipeline using dsl.OneOf to select branch output</li><li>Verify downstream task receives correct output</li></ol> | <ol><li>OneOf resolves to executed branch's output</li><li>Downstream task receives artifact</li><li>IOType = ONEOF_OUTPUT</li></ol> | Yes |

#### Nested Pipeline Execution
**Test Data**: `backend/src/v2/driver/test_data/nested_naming_conflicts.py.yaml`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Execute 3-level nested pipeline | <ol><li>Submit pipeline with pipeline_a → pipeline_b → pipeline_c</li><li>Verify all DAG tasks created</li><li>Check artifact passing through levels</li></ol> | <ol><li>3 DAG tasks created (pipeline_b, pipeline_c, and their nesting)</li><li>Artifacts passed correctly between levels</li><li>Naming conflicts resolved via scope</li></ol> | Yes |
| Verify nested naming conflict resolution | <ol><li>Submit nested pipeline with same task names in different levels</li><li>Verify correct task is resolved for artifact inputs</li></ol> | <ol><li>Task "a" in pipeline_c gets artifact from task "b" in pipeline_c, not pipeline_b</li><li>Scope-based resolution works correctly</li></ol> | Yes |

### API Integration Tests

#### Artifact Service Integration
**Location**: New test file needed

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Create artifact via API and verify in DB | <ol><li>Call CreateArtifact gRPC endpoint</li><li>Query DB directly to verify storage</li></ol> | <ol><li>Artifact exists in DB</li><li>All fields match API request</li></ol> | No |
| Create artifact task and query via ListArtifactTasks | <ol><li>Create artifact_task linking artifact to task</li><li>Call ListArtifactTasks API</li></ol> | <ol><li>ArtifactTask returned</li><li>Correct artifact and task IDs</li></ol> | No |
| Batch create artifacts and verify atomicity | <ol><li>Create batch with 1 invalid artifact</li><li>Verify entire batch fails</li></ol> | <ol><li>No artifacts created</li><li>Transaction rolled back</li></ol> | No |

#### Run Service Task Integration
**Location**: New test file needed

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Create task via API and retrieve via GetTask | <ol><li>Call CreateTask API</li><li>Call GetTask with returned task_id</li></ol> | <ol><li>Task retrieved successfully</li><li>All fields match</li></ol> | No |
| Update task status and verify state transition | <ol><li>Create task with RUNNING status</li><li>Update to SUCCEEDED</li><li>Verify state history</li></ol> | <ol><li>Status updated</li><li>Timestamps set correctly</li></ol> | No |
| List tasks with complex filters | <ol><li>Create tasks with various attributes</li><li>Query with filters (parent_id, status, type)</li></ol> | <ol><li>Only matching tasks returned</li><li>Pagination works</li></ol> | No |

---

## API Contract Tests

### Protobuf Validation Tests

#### v2beta1 Artifact Proto Tests
**Location**: `backend/api/v2beta1/artifact.proto`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| CreateArtifact request validation | <ol><li>Create request with all required fields</li><li>Validate proto serialization</li></ol> | <ol><li>Proto serializes correctly</li><li>All fields accessible</li></ol> | Yes |
| CreateArtifact request with missing required fields | <ol><li>Create request without namespace</li><li>Attempt to process</li></ol> | <ol><li>Validation error returned</li></ol> | Yes |
| ArtifactTask proto with iteration_index | <ol><li>Create ArtifactTask with iteration_index</li><li>Verify field serialization</li></ol> | <ol><li>iteration_index field present</li><li>Can be null for non-loop tasks</li></ol> | Yes |

#### v2beta1 Run Proto Tests
**Location**: `backend/api/v2beta1/run.proto`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| PipelineTaskDetail proto completeness | <ol><li>Create PipelineTaskDetail with all task types</li><li>Verify all fields serialize</li></ol> | <ol><li>All task types (RUNTIME, DAG, LOOP, etc.) supported</li><li>Inputs/outputs correctly structured</li></ol> | Yes |
| IOParameter proto with all value types | <ol><li>Create IOParameter with string, number, bool, list, struct</li><li>Verify serialization</li></ol> | <ol><li>All value types supported</li><li>JSON structs handled correctly</li></ol> | Yes |
| IOArtifact proto with metadata | <ol><li>Create IOArtifact with custom metadata</li><li>Verify metadata JSON</li></ol> | <ol><li>Metadata field accepts arbitrary JSON</li></ol> | Yes |

### Generated Client Tests

#### Go HTTP Client Tests
**Location**: `backend/api/v2beta1/go_http_client/`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| ArtifactServiceClient CreateArtifact call | <ol><li>Initialize client</li><li>Call CreateArtifact method</li><li>Verify HTTP request formed correctly</li></ol> | <ol><li>HTTP POST to /v2beta1/artifacts</li><li>Request body matches proto</li></ol> | Partial |
| RunServiceClient GetTask call | <ol><li>Call GetTask with task_id</li><li>Verify HTTP GET request</li></ol> | <ol><li>HTTP GET to /v2beta1/runs/{run_id}/tasks/{task_id}</li></ol> | Partial |

#### Python HTTP Client Tests
**Location**: `backend/api/v2beta1/python_http_client/test/`

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| ArtifactServiceApi create_artifact call | <ol><li>Initialize Python client</li><li>Call create_artifact</li><li>Verify request</li></ol> | <ol><li>Correct HTTP request formed</li><li>Response deserialized to V2beta1Artifact</li></ol> | Yes |
| RunServiceApi list_tasks call | <ol><li>Call list_tasks with filters</li><li>Verify query parameters</li></ol> | <ol><li>Query params include filters</li><li>Response is V2beta1ListTasksResponse</li></ol> | Yes |

---

## Conditional Logic Tests

### OneOf Implementation Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| OneOf with single output artifact | <ol><li>Submit pipeline with If/Else using dsl.OneOf for artifact</li><li>Verify only executed branch output is returned</li></ol> | <ol><li>OneOf artifact selector matches executed branch</li><li>Downstream task receives correct artifact</li><li>IOType = ONEOF_OUTPUT</li></ol> | Yes |
| OneOf with multiple possible outputs | <ol><li>Submit pipeline with If/Elif/Elif/Else with dsl.OneOf</li><li>Verify correct output selected</li></ol> | <ol><li>Only one output artifact created</li><li>Matches executed branch</li></ol> | Yes |
| OneOf with nested conditions | <ol><li>Submit pipeline with nested If inside If with dsl.OneOf</li><li>Verify resolution through nesting</li></ol> | <ol><li>Correct nested branch output selected</li><li>Artifact resolution correct</li></ol> | Partial |
| Multiple OneOf in same pipeline | <ol><li>Submit pipeline with 2 separate dsl.OneOf outputs</li><li>Verify both resolve independently</li></ol> | <ol><li>Each OneOf resolves to correct branch</li><li>No interference between them</li></ol> | Partial |

### Condition Evaluation Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| String equality condition | <ol><li>Create condition: "value" == task.output</li><li>Run with matching and non-matching values</li></ol> | <ol><li>Condition correctly evaluates true/false</li><li>Branch execution matches condition</li></ol> | Yes |
| Numeric comparison condition | <ol><li>Create condition: task.output > 10</li><li>Run with values above and below threshold</li></ol> | <ol><li>Numeric comparison works</li><li>Correct branch executes</li></ol> | Yes |
| Boolean condition | <ol><li>Create condition based on bool output</li><li>Run with true and false values</li></ol> | <ol><li>Boolean condition evaluated correctly</li></ol> | Yes |
| Complex condition with AND/OR | <ol><li>Create condition: (a > 5) AND (b == "test")</li><li>Test various combinations</li></ol> | <ol><li>Complex expressions evaluate correctly</li></ol> | Partial |

### Conditional Artifact Passing Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Pass artifact from If branch to downstream task | <ol><li>If branch creates artifact</li><li>Downstream task consumes via dsl.OneOf</li></ol> | <ol><li>Artifact passed correctly</li><li>Type and metadata preserved</li></ol> | Yes |
| Pass multiple artifacts from conditional branch | <ol><li>Branch creates 2 artifacts</li><li>Downstream tasks consume both</li></ol> | <ol><li>Both artifacts passed</li><li>Correct artifact resolution</li></ol> | Partial |
| Conditional artifact with different types in branches | <ol><li>If branch outputs Dataset, Else outputs Model</li><li>Downstream task accepts Artifact (base type)</li></ol> | <ol><li>Type compatibility validated</li><li>Artifact passed correctly</li></ol> | Partial |

---

## Security Tests

### Authentication Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Driver authenticates with Pipeline Runner SA token | <ol><li>Configure Pipeline Runner SA</li><li>Run driver</li><li>Verify auth header contains SA token</li></ol> | <ol><li>Driver sends Authorization header</li><li>Token is valid SA token</li></ol> | No |
| Launcher authenticates with Pipeline Runner SA token | <ol><li>Run launcher</li><li>Verify API calls include auth token</li></ol> | <ol><li>Launcher authenticated correctly</li></ol> | No |
| API server rejects unauthenticated requests | <ol><li>Call CreateArtifact without auth header</li><li>Verify rejection</li></ol> | <ol><li>401 Unauthorized returned</li></ol> | No |

### Authorization Tests (RBAC)

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| User with run RBAC can access artifacts in run | <ol><li>Grant user RBAC on run resource</li><li>User calls ListArtifacts for run's artifacts</li></ol> | <ol><li>User can access artifacts</li><li>No explicit artifact RBAC needed</li></ol> | No |
| User without run RBAC cannot access run's artifacts | <ol><li>User has no RBAC on run</li><li>User calls GetArtifact for run's artifact</li></ol> | <ol><li>403 Forbidden returned</li></ol> | No |
| Pipeline Runner SA has namespace-level artifact access | <ol><li>Configure Pipeline Runner SA RBAC</li><li>Driver/Launcher creates artifacts</li></ol> | <ol><li>Artifacts created successfully</li><li>SA has sufficient permissions</li></ol> | No |
| Cross-namespace artifact access denied | <ol><li>User tries to access artifact from different namespace</li><li>Verify denial</li></ol> | <ol><li>403 Forbidden</li><li>Namespace isolation enforced</li></ol> | No |

### Multi-Tenant Isolation Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Artifacts isolated by namespace | <ol><li>Create artifacts in namespace-a and namespace-b</li><li>List artifacts in namespace-a</li></ol> | <ol><li>Only namespace-a artifacts returned</li><li>No cross-namespace leakage</li></ol> | No |
| Tasks isolated by namespace | <ol><li>Create tasks in multiple namespaces</li><li>Query tasks with namespace filter</li></ol> | <ol><li>Only matching namespace tasks returned</li></ol> | No |
| Run isolation by namespace and experiment | <ol><li>Create runs in different namespaces</li><li>Verify isolation</li></ol> | <ol><li>Runs properly isolated</li></ol> | No |

---

## Performance Tests

### Baseline Performance Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Measure task creation latency | <ol><li>Create 100 tasks sequentially</li><li>Measure avg latency per CreateTask call</li><li>Compare to baseline</li></ol> | <ol><li>Avg latency < 100ms</li><li>No regression vs. MLMD-based system</li></ol> | Partial |
| Measure artifact creation latency | <ol><li>Create 100 artifacts</li><li>Measure avg latency</li></ol> | <ol><li>Avg latency < 50ms</li></ol> | Partial |
| Measure task update latency | <ol><li>Update 100 tasks</li><li>Measure latency</li></ol> | <ol><li>Avg latency < 100ms</li></ol> | Partial |

### Load Testing

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Concurrent pipeline executions | <ol><li>Submit 50 pipelines concurrently</li><li>Monitor API server CPU/memory</li><li>Verify all complete successfully</li></ol> | <ol><li>All pipelines complete</li><li>No API server crashes</li><li>Resource usage within limits</li></ol> | No |
| High-frequency task updates | <ol><li>Simulate 100 launcher pods updating tasks concurrently</li><li>Monitor DB connections and latency</li></ol> | <ol><li>All updates succeed</li><li>DB connection pool not exhausted</li><li>No deadlocks</li></ol> | No |
| Large artifact metadata | <ol><li>Create artifacts with 10KB metadata JSON</li><li>Verify no performance degradation</li></ol> | <ol><li>Large metadata handled</li><li>No significant latency increase</li></ol> | No |

### Database Performance Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Task query performance with large task count | <ol><li>Create run with 1000 tasks</li><li>Query ListTasks for run</li><li>Measure query time</li></ol> | <ol><li>Query completes in < 1 second</li><li>Pagination works efficiently</li></ol> | No |
| Artifact query performance with joins | <ol><li>Create 1000 artifacts linked to tasks</li><li>Query ListArtifactTasks</li><li>Measure time</li></ol> | <ol><li>Query optimized with proper indexes</li><li>Completes in < 1 second</li></ol> | No |

### Cache Performance Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Cache hit detection speed | <ol><li>Run pipeline twice with caching</li><li>Measure fingerprint lookup time on 2nd run</li></ol> | <ol><li>Fingerprint lookup < 50ms</li><li>Cache hit detected correctly</li></ol> | Partial |
| Cache with large fingerprint dataset | <ol><li>Create 10,000 cached tasks</li><li>Attempt cache lookup</li></ol> | <ol><li>Lookup remains fast</li><li>Index on fingerprint column effective</li></ol> | No |

---

## Regression Tests

### Full E2E Regression Suite

#### Standard RHOAI Cluster Testing

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Execute all KFP sample pipelines | <ol><li>Deploy KFP to standard cluster</li><li>Submit all official sample pipelines</li><li>Verify all complete successfully</li></ol> | <ol><li>All samples pass</li><li>Outputs match expected</li><li>No crashes or errors</li></ol> | Partial |
| Execute pipelines with all component types | <ol><li>Run pipelines using container, importer, condition, loop components</li><li>Verify all work correctly</li></ol> | <ol><li>All component types supported</li><li>Backward compatibility maintained</li></ol> | Partial |
| Recurring runs continue to work | <ol><li>Create recurring run schedule</li><li>Wait for multiple executions</li><li>Verify each execution succeeds</li></ol> | <ol><li>Scheduled runs execute</li><li>Each run completes successfully</li></ol> | Partial |

#### Caching Regression Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Previously cached pipelines still use cache | <ol><li>Migrate DB with cached tasks</li><li>Run same pipeline post-migration</li><li>Verify cache hit</li></ol> | <ol><li>Cache fingerprint found</li><li>Task reused, not re-executed</li></ol> | No |
| New cache entries work correctly | <ol><li>Run new pipeline twice with caching enabled</li><li>Verify cache hit on 2nd run</li></ol> | <ol><li>Fingerprint stored correctly</li><li>Cache mechanism functional</li></ol> | Partial |

#### Metrics Regression Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| System metrics logged to metrics table | <ol><li>Run pipeline outputting system.Metrics</li><li>Verify metrics stored in metrics table</li></ol> | <ol><li>Metrics in metrics table, not artifacts table</li><li>Values correct</li></ol> | Partial |
| Classification metrics stored correctly | <ol><li>Output system.ClassificationMetrics</li><li>Verify JSON storage</li></ol> | <ol><li>JSON metrics stored</li><li>Can be retrieved and displayed</li></ol> | Partial |

#### Artifact Regression Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Artifact download from object store works | <ol><li>Pipeline creates artifact</li><li>Use UI to download artifact</li></ol> | <ol><li>Pre-signed URL generated</li><li>Artifact downloaded correctly</li></ol> | No |
| Artifact visualization in UI | <ol><li>Create pipeline with visualizations</li><li>View in UI</li></ol> | <ol><li>Visualizations render</li><li>Data correct</li></ol> | No |

---

## Migration Tests

### Database Migration Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Migration script runs without errors | <ol><li>Prepare MLMD database dump</li><li>Run migration script</li><li>Verify completion without errors</li></ol> | <ol><li>Script completes successfully</li><li>All MLMD executions converted to tasks</li><li>All MLMD artifacts converted to KFP artifacts</li></ol> | No |
| Task table dropped and recreated | <ol><li>Check task table schema before migration</li><li>Run migration</li><li>Verify new schema</li></ol> | <ol><li>Old task table dropped</li><li>New schema with correct columns</li></ol> | No |
| MLMD executions converted to tasks | <ol><li>Count MLMD executions before migration</li><li>Run migration</li><li>Count tasks after migration</li></ol> | <ol><li>Task count >= execution count</li><li>All execution types mapped correctly (Container→RUNTIME, DAG→DAG, etc.)</li></ol> | No |
| MLMD artifacts converted to KFP artifacts | <ol><li>Count MLMD artifacts</li><li>Run migration</li><li>Count KFP artifacts</li></ol> | <ol><li>Non-metric artifacts migrated to artifacts table</li><li>Metric artifacts migrated to metrics table</li></ol> | No |
| Cache fingerprints migrated correctly | <ol><li>Verify MLMD executions with fingerprints</li><li>Migrate</li><li>Verify fingerprints in tasks</li></ol> | <ol><li>Only COMPLETE executions have fingerprints</li><li>Fingerprints match MLMD values</li></ol> | No |
| Artifact relationships preserved | <ol><li>Check MLMD events for artifact-execution links</li><li>Migrate</li><li>Verify artifact_tasks table</li></ol> | <ol><li>All artifact-task relationships preserved</li><li>Input/output types correct</li></ol> | No |

### Post-Migration Validation Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| API server starts successfully post-migration | <ol><li>Complete migration</li><li>Start API server</li><li>Verify no startup errors</li></ol> | <ol><li>Server starts without errors</li><li>Health check passes</li></ol> | No |
| Existing runs visible in UI | <ol><li>Migrate DB with completed runs</li><li>Open UI</li><li>Verify runs listed</li></ol> | <ol><li>All pre-migration runs visible</li><li>Run details accessible</li></ol> | No |
| Can execute new pipelines post-migration | <ol><li>Complete migration</li><li>Submit new pipeline</li><li>Verify execution</li></ol> | <ol><li>New pipeline executes successfully</li><li>Uses new task/artifact storage</li></ol> | No |
| Pre-migration artifacts accessible | <ol><li>Migrate DB</li><li>Access artifact from old run</li><li>Download artifact</li></ol> | <ol><li>Artifact metadata retrieved</li><li>Artifact downloadable</li></ol> | No |

### Rollback Testing

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Database backup can be restored | <ol><li>Backup DB before migration</li><li>Run migration</li><li>Restore backup</li><li>Verify MLMD-based system works</li></ol> | <ol><li>Backup restores successfully</li><li>Old system functional</li></ol> | No |

---

## Task Type-Specific Tests

Based on the proposal design document, the new system defines specific task types to replace MLMD executions. Each task type requires specific testing.

### Task Type Definitions from Proposal

| Task Type | Purpose | MLMD Equivalent |
|-----------|---------|-----------------|
| ROOT_DAG | Root pipeline execution | Pipeline context |
| DAG | Task group execution | DAG execution |
| RUNTIME | Container executor execution | Container execution |
| LOOP | For-loop group | Loop DAG execution |
| LOOP_ITERATION | Single loop iteration | Loop iteration DAG execution |
| CONDITION | Conditional evaluation | Condition DAG execution |
| CONDITION_BRANCH | If/Elif/Else branch grouping | Condition branch DAG execution |
| EXITHANDLER | Exit handler task grouping | Exit handler DAG execution |

### ROOT_DAG Task Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Root DAG stores pipeline runtime config | <ol><li>Submit pipeline with runtime parameters</li><li>Verify ROOT_DAG task created</li><li>Check runtime config stored in task</li></ol> | <ol><li>ROOT_DAG task type set correctly</li><li>Runtime config JSON stored in task inputs</li><li>Can be retrieved by child tasks</li></ol> | Yes |
| Root DAG associates with run | <ol><li>Create ROOT_DAG task</li><li>Verify run_id association</li><li>Check parent_task_uuid is null</li></ol> | <ol><li>Task linked to run correctly</li><li>No parent task (root level)</li></ol> | Yes |
| Root DAG passes execution_id to child drivers | <ol><li>ROOT_DAG driver creates task</li><li>Child drivers receive parent_task_id</li></ol> | <ol><li>Child drivers get parent_task_id instead of execution_id</li></ol> | Yes |

### RUNTIME Task Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Container driver creates RUNTIME task | <ol><li>Run container driver</li><li>Verify RUNTIME task created</li><li>Check executor inputs populated</li></ol> | <ol><li>Task type set to RUNTIME</li><li>ExecutorInput contains resolved parameters/artifacts</li><li>PodSpec generated with --task_id flag</li></ol> | Yes |
| RUNTIME task stores cache fingerprint on success | <ol><li>Complete launcher execution successfully</li><li>Verify task updated with fingerprint</li></ol> | <ol><li>Fingerprint stored only on successful completion</li><li>Can be found for cache hit detection</li></ol> | Yes |
| RUNTIME task updates with output artifacts | <ol><li>Launcher creates output artifacts</li><li>Verify task updated with outputs</li></ol> | <ol><li>Task outputs contain artifact references</li><li>ArtifactTasks created linking artifacts to task</li></ol> | Yes |

### LOOP and LOOP_ITERATION Task Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| LOOP task calculates iteration count | <ol><li>Create for-loop with list input [1,2,3,4,5]</li><li>Verify LOOP task created</li><li>Check iteration_count in TypeAttrs</li></ol> | <ol><li>Task type set to LOOP</li><li>TypeAttrs JSON contains iteration_count=5</li><li>Child LOOP_ITERATION tasks created</li></ol> | Yes |
| LOOP_ITERATION task sets iteration index | <ol><li>Run loop iteration driver for index 2</li><li>Verify LOOP_ITERATION task created</li><li>Check iteration_index stored</li></ol> | <ol><li>Task type set to LOOP_ITERATION</li><li>TypeAttrs contains iteration_index=2</li><li>Parent task is LOOP task</li></ol> | Yes |
| Loop artifact collection works | <ol><li>Each LOOP_ITERATION creates output artifact</li><li>Downstream task uses dsl.Collected()</li><li>Verify all artifacts collected</li></ol> | <ol><li>All iteration artifacts linked to downstream task</li><li>IOType set to ITERATOR_OUTPUT</li></ol> | Yes |

### CONDITION and CONDITION_BRANCH Task Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| CONDITION task evaluates condition expression | <ol><li>Create If condition with expression "param > 5"</li><li>Run condition driver with param=10</li><li>Verify condition result</li></ol> | <ol><li>Task type set to CONDITION</li><li>Condition result stored in task outputs</li><li>Child tasks execute based on result</li></ol> | Yes |
| CONDITION_BRANCH task groups If/Elif/Else | <ol><li>Create If/Elif/Else structure</li><li>Verify CONDITION_BRANCH task created</li><li>Check child CONDITION tasks</li></ol> | <ol><li>Task type set to CONDITION_BRANCH</li><li>Multiple child CONDITION tasks created</li><li>OneOf resolution works correctly</li></ol> | Yes |
| Nested conditions create proper task hierarchy | <ol><li>Create nested If inside If</li><li>Verify task parent-child relationships</li></ol> | <ol><li>Outer CONDITION_BRANCH contains inner CONDITION_BRANCH</li><li>Proper task nesting preserved</li></ol> | Partial |

### EXITHANDLER Task Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Exit handler tasks grouped under EXITHANDLER | <ol><li>Create pipeline with dsl.ExitHandler</li><li>Verify EXITHANDLER task created</li><li>Check exit handler tasks are children</li></ol> | <ol><li>Task type set to EXITHANDLER</li><li>Exit handler components are child tasks</li><li>Executes regardless of pipeline success/failure</li></ol> | Partial |
| Exit handler execution name parsing | <ol><li>Migration script encounters execution with name "exit-handler-*"</li><li>Verify converted to EXITHANDLER task</li></ol> | <ol><li>Execution name parsed correctly</li><li>Task type set to EXITHANDLER</li></ol> | No |

---

## Metrics Handling Tests

The proposal specifies that metrics are no longer stored as artifacts but in a dedicated metrics table. This requires comprehensive testing.

### Metrics Storage Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| system.Metrics stored in metrics table | <ol><li>Create component that outputs system.Metrics</li><li>Run pipeline</li><li>Verify metrics stored in metrics table, not artifacts</li></ol> | <ol><li>No artifact created for metrics</li><li>Entry created in metrics table</li><li>Key-value pairs stored correctly</li></ol> | Yes |
| system.ClassificationMetrics stored as JSON | <ol><li>Output system.ClassificationMetrics</li><li>Verify JSON storage in metrics table</li></ol> | <ol><li>Classification metrics stored as JSON</li><li>Can be retrieved for UI display</li></ol> | Yes |
| system.SlicedClassificationMetrics stored as JSON | <ol><li>Output system.SlicedClassificationMetrics</li><li>Verify JSON storage</li></ol> | <ol><li>Sliced metrics stored as JSON</li><li>Structure preserved</li></ol> | Yes |
| Metrics have no URI | <ol><li>Create metrics</li><li>Verify no URI field populated</li><li>Check UI doesn't show download link</li></ol> | <ol><li>Metrics have no URI</li><li>UI shows metrics data inline</li><li>No artifact download option</li></ol> | Partial |

### Metrics Resolution Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Driver resolves metrics as input artifacts | <ol><li>Task A outputs system.Metrics</li><li>Task B consumes metrics as input</li><li>Verify resolution works</li></ol> | <ol><li>Driver queries metrics table</li><li>Converts metrics to RuntimeArtifact format</li><li>Downstream task receives metrics</li></ol> | Yes |
| GetOutputMetricsByTaskID function works | <ol><li>Create task with metrics output</li><li>Call GetOutputMetricsByTaskID</li><li>Verify metrics returned</li></ol> | <ol><li>Function queries task's output_metrics</li><li>Returns map[string]*OutputArtifact</li></ol> | Yes |
| Metrics schema type determination | <ol><li>Driver determines artifact schema type from ComponentInputSpec</li><li>Verify metrics vs artifacts handled correctly</li></ol> | <ol><li>SchemaTitle correctly identified</li><li>Metrics routed to metrics table</li><li>Artifacts routed to artifacts table</li></ol> | Yes |

### Metrics Migration Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| MLMD metrics artifacts migrated to metrics table | <ol><li>MLMD DB contains system.Metrics artifacts</li><li>Run migration script</li><li>Verify metrics moved to metrics table</li></ol> | <ol><li>Metrics artifacts not in artifacts table</li><li>Entries created in metrics table</li><li>Custom properties converted to JSON</li></ol> | No |
| Non-metrics artifacts remain in artifacts table | <ol><li>MLMD DB contains Model/Dataset artifacts</li><li>Run migration</li><li>Verify only metrics moved</li></ol> | <ol><li>Model/Dataset artifacts in artifacts table</li><li>Only metrics moved to metrics table</li></ol> | No |

---

## Authentication and Authorization Tests (Enhanced)

Based on the proposal's specific RBAC requirements for Driver/Launcher and new API endpoints.

### Driver/Launcher Authentication Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Driver uses Pipeline Runner SA token for RunService | <ol><li>Configure Pipeline Runner SA with tokens</li><li>Run driver</li><li>Verify RunService calls include auth header</li></ol> | <ol><li>Authorization header contains SA token</li><li>RunService accepts authenticated requests</li></ol> | No |
| Driver uses Pipeline Runner SA token for ArtifactService | <ol><li>Driver creates artifacts</li><li>Verify ArtifactService calls authenticated</li></ol> | <ol><li>CreateArtifact calls include auth header</li><li>ArtifactService validates SA token</li></ol> | No |
| Launcher uses Pipeline Runner SA token for all APIs | <ol><li>Launcher updates task status</li><li>Launcher creates artifacts</li><li>Verify both calls authenticated</li></ol> | <ol><li>UpdateTask includes auth header</li><li>CreateArtifact includes auth header</li></ol> | No |
| Pipeline Runner SA has required RBAC permissions | <ol><li>Configure minimal RBAC for Pipeline Runner SA</li><li>Attempt pipeline execution</li><li>Verify success</li></ol> | <ol><li>SA can create/update tasks</li><li>SA can create artifacts</li><li>SA has namespace-level access</li></ol> | No |

### SubjectAccessReview Integration Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| RunService endpoints use SubjectAccessReview | <ol><li>Call CreateTask with valid token but no RBAC</li><li>Verify resourceManager.IsAuthorized called</li><li>Check 403 returned</li></ol> | <ol><li>SubjectAccessReview performed</li><li>Request denied due to insufficient RBAC</li></ol> | No |
| ArtifactService endpoints use SubjectAccessReview | <ol><li>Call CreateArtifact without proper RBAC</li><li>Verify authorization check</li></ol> | <ol><li>Authorization checked against "artifacts" resource</li><li>403 Forbidden returned</li></ol> | No |

### Artifact Access Control Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Artifacts in runs accessible with run RBAC | <ol><li>User has RBAC on run resource</li><li>User calls GetArtifact for artifact in that run</li><li>Verify access granted</li></ol> | <ol><li>Access granted based on run RBAC</li><li>No separate artifact RBAC needed</li></ol> | No |
| Get/List artifacts requires artifacts resource RBAC | <ol><li>User calls ListArtifacts globally</li><li>Verify requires "artifacts" resource permission</li></ol> | <ol><li>RBAC checked against "artifacts" resource</li><li>Only artifacts user has access to returned</li></ol> | No |
| Reimport=false artifacts require originating namespace RBAC | <ol><li>Artifact with Reimport=false from namespace-a</li><li>User in namespace-b tries to access</li><li>Verify denied</li></ol> | <ol><li>Access denied due to originating namespace</li><li>RBAC checked against artifact's namespace</li></ol> | No |

### Pre-signed URL Download Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| UI artifact download uses pre-signed URLs | <ol><li>User views artifact in UI</li><li>Click download link</li><li>Verify pre-signed URL generated</li></ol> | <ol><li>KFP server generates pre-signed URL</li><li>No direct UI download from object store</li><li>RBAC enforced for URL generation</li></ol> | No |
| Pre-signed URL authorization via RBAC | <ol><li>User without artifact access tries download</li><li>Verify URL generation denied</li></ol> | <ol><li>URL generation requires artifact access</li><li>403 returned if no access</li></ol> | No |

---

## Frontend Changes Tests

The proposal specifies specific UI changes to transition from MLMD queries to Task API queries.

### UI Data Fetching Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Run Details page uses Task API instead of MLMD | <ol><li>Open run details page</li><li>Verify API calls made</li><li>Check data displayed correctly</li></ol> | <ol><li>No MLMD API calls (getKfpV2RunContext, getExecutionsFromContext)</li><li>Uses run.run_details.task_details from Run object</li><li>Task information displayed correctly</li></ol> | No |
| RuntimeNodeDetailsV2 uses tasks instead of executions | <ol><li>Click on task node in run graph</li><li>Verify task details sidebar</li><li>Check all information present</li></ol> | <ol><li>Task details fetched from tasks field</li><li>All execution info available as task info</li><li>No missing information</li></ol> | No |
| Artifact fetching uses fetchArtifactsFromTasks | <ol><li>View artifacts in run details</li><li>Verify artifact API calls</li></ol> | <ol><li>Artifacts fetched via Task relationship</li><li>ListArtifactTasks API used</li><li>All artifacts displayed</li></ol> | No |
| Execution page removed | <ol><li>Verify execution page no longer accessible</li><li>All execution info available in task nodes</li></ol> | <ol><li>Execution page returns 404 or redirects</li><li>Task node details contain all needed info</li></ol> | No |

### Metrics Visualization Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Metrics fetched from Task response instead of Artifacts | <ol><li>View task with metrics output</li><li>Check Visualization nav</li><li>Verify metrics displayed</li></ol> | <ol><li>Metrics data from task.output_metrics</li><li>No MLMD artifact query for metrics</li><li>Visualization renders correctly</li></ol> | No |
| Metrics artifacts show no URI in UI | <ol><li>View metrics in artifact list</li><li>Verify no download link</li></ol> | <ol><li>Metrics show as data, not downloadable files</li><li>No URI displayed</li></ol> | No |
| Classification metrics JSON displayed correctly | <ol><li>View ClassificationMetrics in UI</li><li>Verify JSON data rendered</li></ol> | <ol><li>JSON structure preserved</li><li>Data displayed in appropriate format</li></ol> | No |

### Compare UI Tests

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| CompareV2 uses Task API instead of MLMD | <ol><li>Compare multiple runs</li><li>Verify API calls made</li><li>Check comparison data correct</li></ol> | <ol><li>No MLMD calls (getKfpV2RunContext, getExecutionsFromContext)</li><li>Uses Task API for all run data</li><li>Comparison functionality preserved</li></ol> | No |

### Store Session Info Tests

The proposal removes store_session_info as a custom property and handles it directly in launcher.

| Test Case Summary | Test Steps | Expected Result | Automated? |
|-------------------|------------|-----------------|------------|
| Artifact download works without store_session_info property | <ol><li>Create artifact without store_session_info</li><li>Download via UI</li><li>Verify download succeeds</li></ol> | <ol><li>Server builds session info from launcher config</li><li>Download works correctly</li><li>No dependency on custom property</li></ol> | No |
| Server builds session info from pipeline root | <ol><li>Server handles artifact download request</li><li>Verify cfg.GetStoreSessionInfo(pipelineRoot) called</li></ol> | <ol><li>Session info built server-side</li><li>Same logic as root driver</li></ol> | No |
| Launcher uses session info directly | <ol><li>Launcher uploads artifacts</li><li>Verify session info used from launcher config</li><li>Check no custom property stored</li></ol> | <ol><li>Session info used directly in launcher</li><li>No store_session_info custom property</li></ol> | Partial |

---

### Continuous Integration

**Automated Test Execution**:
- Unit tests run on every commit (via GitHub Actions)
- Integration tests run on PR merge to master
- Nightly full regression suite
- Performance tests run weekly

**GitHub Workflow Integration**:
- `.github/workflows/kfp-sdk-tests.yml` - SDK tests
- `.github/workflows/api-server-tests.yml` - API server tests
- `.github/workflows/e2e-test.yml` - End-to-end tests
- New workflow needed: `.github/workflows/mlmd-removal-tests.yml` - Specialized tests for this refactoring

### Manual Testing Requirements

**Manual Verification Needed**:
1. **UI Testing**: Frontend changes cannot be fully automated
   - Verify run details page shows tasks correctly
   - Check artifact visualization
   - Validate metrics display
   - Test artifact download links

2. **Migration Testing**: Requires real MLMD database
   - Migration script execution
   - Post-migration validation
   - Rollback testing

3. **Performance Testing**: Realistic load testing
   - Deploy to larger cluster (GKE/EKS)
   - Run at scale (100+ concurrent pipelines)
   - Monitor resource usage

4. **Security Testing**: Multi-user scenarios
   - Multiple users with different RBAC roles
   - Cross-namespace access attempts
   - Token expiration scenarios

---

## Success Criteria

### Functional Success Criteria

1. **All Unit Tests Pass**: 100% of new unit tests pass, 80%+ code coverage
2. **All Integration Tests Pass**: End-to-end pipelines execute successfully
3. **New Features Work**: OneOf, nested pipelines, parameter iterators function correctly
4. **No Regressions**: All existing sample pipelines continue to work
5. **Migration Validated**: Migration script successfully converts MLMD data to new schema

### Performance Success Criteria

1. **No Significant Latency Regression**: Task creation/update latency within 20% of baseline
2. **Scalability Maintained**: Can handle 100+ concurrent pipeline executions
3. **Database Performance**: Queries complete in < 1 second for typical workloads
4. **Cache Performance**: Cache hit detection < 50ms

### Security Success Criteria

1. **RBAC Enforced**: All API endpoints protected by RBAC
2. **Namespace Isolation**: No cross-namespace data leakage
3. **Authentication Required**: No unauthenticated access possible
4. **Audit Trail**: All operations logged appropriately

### Operational Success Criteria

1. **API Server Stable**: No crashes during testing
2. **Database Integrity**: No data corruption
3. **Migration Reversible**: Can rollback migration if needed
4. **Documentation Complete**: Migration guide, API docs, release notes

---

## Risk Assessment

### Critical Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Migration script fails on customer data | Medium | Critical | Extensive testing with diverse MLMD databases; provide dry-run mode |
| Performance regression unacceptable | Low | High | Early performance testing; optimization before release |
| Data loss during migration | Low | Critical | Mandatory backup before migration; validation script |
| Incompatibility with existing pipelines | Medium | High | Extensive regression testing; backward compatibility checks |
| Security vulnerabilities in new APIs | Low | Critical | Security audit; penetration testing |

### Medium Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| UI doesn't display all information | Medium | Medium | Manual UI testing; user acceptance testing |
| Cache mechanism doesn't work as expected | Low | Medium | Dedicated cache tests; performance validation |
| Generated clients have bugs | Low | Medium | Client integration tests; sample code validation |
| Metrics display broken in UI | Medium | Medium | Manual verification; automated screenshot tests if possible |

### Low Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Documentation incomplete | Medium | Low | Documentation review process |
| Edge cases not covered | Low | Low | Comprehensive test case design |

---

## Test Data and Resources

### Test Pipelines

1. **componentInput_level_1_test.py**: Component inputs and runtime constants
2. **loop_collected_raw_Iterator.py**: Loop with raw list iterator
3. **loop_collected_InputParameter_Iterator.py**: Loop with parameter iterator
4. **nested_naming_conflicts.py**: 3-level nested pipelines with naming conflicts
5. **oneof_simple.py**: OneOf with If/Elif/Else branches
6. **taskOutputArtifact_test.py**: Artifact output from task
7. **taskOutputParameter_test.py**: Parameter output from task

### Required Infrastructure

- **MySQL Test Database**: Fresh instance for each test run
- **MinIO Server**: Local S3-compatible storage
- **Kind Clusters**: Multiple clusters for different test scenarios
- **GitHub Actions Runners**: For CI/CD test execution

### Test Artifacts

- **Performance Baselines**: JSON file with latency/throughput metrics
- **MLMD Database Dumps**: Sample databases for migration testing
- **Expected Outputs**: Golden files for regression testing

---

## Appendix

### Test Environment Setup

#### Local Development Setup

```bash
# 1. Set up database
docker run -d --name kfp-mysql -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=kfp -p 3306:3306 mysql:8.0

# 2. Set up MinIO
docker run -d --name kfp-minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minio -e MINIO_ROOT_PASSWORD=minio123 \
  minio/minio server /data --console-address ":9001"

# 3. Create Kind cluster
kind create cluster --name kfp-test --config=test-cluster-config.yaml

# 4. Deploy KFP
kubectl apply -k manifests/kustomize/env/platform-agnostic

# 5. Run tests
cd backend/src/apiserver/storage
go test -v ./...
```

#### CI Test Cluster Config

```yaml
# test-cluster-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
```

### Glossary

- **MLMD**: ML Metadata - the metadata store being removed
- **OneOf**: Conditional branching construct that selects one output from multiple possible branches
- **DAG**: Directed Acyclic Graph - represents pipeline structure
- **IOType**: Input/Output Type - categorizes parameter/artifact sources (e.g., COMPONENT_INPUT, TASK_OUTPUT)
- **Artifact Task**: Join table linking artifacts to tasks with metadata
- **Cache Fingerprint**: Hash of task inputs used for cache hit detection
- **Iteration Index**: Index of current loop iteration (0-based)

---

## Document Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-06 | KFP Test Team | Initial test plan creation |

---

**End of Test Plan**
