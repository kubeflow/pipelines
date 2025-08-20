## Design Details

### New KFP Database Schema

See [schema_changes.sql] for the database schema additions and changes.

Note that a `Task` is a db model for a task node type as viewed in the Run Graph of the UI.

Note also that we will be dropping the `Task` table that exists today and recreating it. This is because it is rarely used within KFP, and where it is used, it is unnecessary (i.e. caching). This will require a migration strategy, addressed later in the proposal.

[schema_changes.sql]: ./schema_changes.sql

### KFP Server API

The KFP APIServer will now handle Artifacts, DAGs, input resolution, and other responsibilities previously managed by MLMD.

The Artifact service changes are detailed in [artifacts.proto]. The Driver and Launcher will introduce a
`v2beta1.ArtifactServiceClient` to interact with this API.

For the Driver and Launcher, a `v2beta1.RunServiceClient` obtained via `NewRunServiceClient()` in `backend/api/v2beta1`
will replace the MLMD client.

The additions to the RunService client are documented in [runs.proto].

An example of the updated run response format can be found in [runs.json]. 

[artifacts.proto]: ./protos/artifacts.proto
[runs.proto]: ./protos/runs.proto
[runs.json]: ./protos/runs.json

### MLMD Client replacement

This section explains how we will replace the MLMD client with new KFP Server API functionality in Driver/Launcher code.

The key changes and replacements are outlined below.
```go
package metadata

type DAG struct {
  Execution *Execution
}
type Execution struct {
  execution *pb.Execution
  pipeline  *Pipeline
}
// A pipeline context contains: create/update time, namespace, pipeline_root
// The pipelineCtx represents a Pipeline (not PipelineVersion) and is created once per pipeline
// This struct is primarily used for creating execution Associations and can most likely be discarded
type Pipeline struct {
  pipelineCtx    *pb.Context
  pipelineRunCtx *pb.Context
}

// These can be replaced with Artifacts from KFP artifacts.proto
type InputArtifact struct {
  Artifact *pb.Artifact
}
type OutputArtifact struct {
  Name     string
  Artifact *pb.Artifact
  Schema   string
}

// GetPipeline actually gets or creates a context if it doesn't already exist for this pipeline and pipelineRun (one context each)
func (c *Client) GetPipeline(ctx context.Context, pipelineName, runID, namespace, runResource, pipelineRoot, storeSessionInfo string) (*Pipeline, error)
func (c *Client) GetDAG(ctx context.Context, executionID int64) (*DAG, error)
func (c *Client) PublishExecution(ctx context.Context, execution *Execution, outputParameters map[string]*structpb.Value, outputArtifacts []*OutputArtifact, state pb.Execution_State) error
func (c *Client) CreateExecution(ctx context.Context, pipeline *Pipeline, config *ExecutionConfig) (*Execution, error)
// Creates execution, updating it with pod and status info
func (c *Client) PrePublishExecution(ctx context.Context, execution *Execution, config *ExecutionConfig) (*Execution, error)
func (c *Client) UpdateDAGExecutionsState(ctx context.Context, dag *DAG, pipeline *Pipeline) error
func (c *Client) PutDAGExecutionState(ctx context.Context, executionID int64, state pb.Execution_State) error
func (c *Client) GetExecutions(ctx context.Context, ids []int64) ([]*pb.Execution, error)
func (c *Client) GetExecution(ctx context.Context, id int64) (*Execution, error)
func (c *Client) GetPipelineFromExecution(ctx context.Context, id int64) (*Pipeline, error)
func (c *Client) GetExecutionsInDAG(ctx context.Context, dag *DAG, pipeline *Pipeline, filter bool) (executionsMap map[string]*Execution, err error)

func (c *Client) GetEventsByArtifactIDs(ctx context.Context, artifactIds []int64) ([]*pb.Event, error)
func (c *Client) GetArtifactName(ctx context.Context, artifactId int64) (string, error) // Not used
func (c *Client) GetArtifacts(ctx context.Context, ids []int64) ([]*pb.Artifact, error)
func (c *Client) GetOutputArtifactsByExecutionId(ctx context.Context, executionId int64) (map[string]*OutputArtifact, error)
func (c *Client) GetInputArtifactsByExecutionID(ctx context.Context, executionID int64) (inputs map[string]*pipelinespec.ArtifactList, err error)
func (c *Client) RecordArtifact(ctx context.Context, outputName, schema string, runtimeArtifact *pipelinespec.RuntimeArtifact, state pb.Artifact_State, bucketConfig *objectstore.Config) (*OutputArtifact, error)
func (c *Client) GetOrInsertArtifactType(ctx context.Context, schema string) (typeID int64, err error)
func (c *Client) FindMatchedArtifact(ctx context.Context, artifactToMatch *pb.Artifact, pipelineContextId int64) (matchedArtifact *pb.Artifact, err error)
```

These will be replaced by calls to v2beta1.RunService instead:

```go
package run_client

// Replaces GetPipeline, additionally we will need to pass experiment ID to the Driver/Launcher
// It also replaces GetPipelineFromExecution, since Tasks have a RunID
func (c *RunServerClient) GetRun(ctx, runID, experimentID) (*apiv2beta1.Run, error)
// Replaces GetDAG (filter on task type), GetExecutions, GetExecution
func (c *RunServerClient) GetTask(ctx, taskID) (*apiv2beta1.PipelineTaskDetail, error)  // uses GetTask() in RunsAPI
func (c *RunServerClient) GetTasks(ctx, taskID) ([]*apiv2beta1.PipelineTaskDetail, error) // uses ListTasks() in RunsAPI

// Replaces PublishExecution, CreateExecution, PrePublishExecution, UpdateDAGExecutionsState
func (c *RunServerClient) CreateTask(ctx context.Context, task apiv2beta1.PipelineTaskDetail) (*apiv2beta1.PipelineTaskDetail, error)
func (c *RunServerClient) UpdateTask(ctx context.Context, task apiv2beta1.PipelineTaskDetail) (*apiv2beta1.PipelineTaskDetail, error)

// Replaces GetExecutionsInDAG
// Queries Run API's ListTasks() with run_id field 
func (c *RunServerClient) GetChildTasks(ctx context.Context, task apiv2beta1.PipelineTaskDetail) (map[string]*apiv2beta1.PipelineTaskDetail, error)
```

In a similar manner, the v2beta1 ArtifactService can be used to implement the following:

* `GetEventsByArtifactIDs` -> `GetArtifactTasks`, queries `ListArtifactTasks`
* `GetArtifacts` -> `ListArtifacts`
* `RecordArtifact` -> `CreateArtifact`
* `GetOutputArtifactsByExecutionId` -> `GetOutArtifactsByTaskID`, queries `ListArtifactTasks` and `ListArtifacts`
* `GetInputArtifactsByExecutionID` -> `GetInputArtifactsByTaskID`, queries `ListArtifactTasks` and `ListArtifacts`
* `GetOrInsertArtifactType` -> Use a combination of `GetArtifact` and `UpdateArtifact`
* `FindMatchedArtifact` -> Use `ListArtifacts` with `uri` filter

### Driver changes

Various portions of the Driver require adjustments to transition away from MLMD. The key change involves transitioning from creating Executions to creating Tasks.

In the pipeline, the `ROOT_DAG` Driver currently passes an `execution_id` flag to subsequent drivers. This needs to be
updated to pass `parent_task_id` instead. Downstream Driver tasks will continue to propagate their corresponding tasks as `parent_task_id` as well.

Multiple control flows involving execution creation and management exist. These are detailed in the sections below.

#### Control Flows

The KFP Driver component creates and manages different types of executions during pipeline execution. These executions follow two main patterns:

**DagExecution**

Manages pipeline control flow. Has two subtypes:

- **RootDag** (runs once per pipeline)
  - Creates the Pipeline and PipelineRun context in MLMD
  - Stores pipeline runtime input information

- **Dag** (runs for each task group)
  - Handles conditional logic (Condition, ConditionBranch, Loop, LoopIteration)
  - Resolves conditional expressions
  - Processes inputs for task groups
  - Calculates iteration counts for loops

**ContainerExecution**
  - Runs before every Launcher/executor pod
  - Handles caching decisions and input resolution
  - Generates the pod specification for the executor
  - Stores cache fingerprint in the task store
  - Download and Upload artifacts

Each execution type has distinct responsibilities and interacts with MLMD differently based on its role in the pipeline workflow.

##### ContainerExecution

Container Drivers will now create a task of type `Runtime`. When creating the PodSpecPatch, the Driver will pass the `--task_id` instead of the `execution_id` flag.

##### DagExecution — Loops

Loops in KFP today require two types of dags, there is either a dag that has an `iteration_count` or a dag that has an `iteration_index`.
We'll refer to these as `Loop` and `LoopIteration` respectively. A `Loop` is a task grouping of components that will run within this loop. It tracks the total count of iterations via `iteration_count`. A `LoopIteration` is a dag that tracks the current iteration for a given loop via `iteration_index`.

Each of these results in a `DagExecution`. Recall that, the components that run within a `LoopIteration` will continue to have their regular `Runtime` tasks.

Each of these loop types is used to resolve inputs/outputs and will need to be logged as Tasks into the Tasks table. The Tasks will be logged as `Loop` and `LoopIteration` respectively, and leverage the `RunServer` and `ArtifactServer` for input resolution.

##### DagExecution — Exit handler

Any task under the `dsl.Exithandler` group falls within a Dag execution. These tasks will now be grouped under a task of type `Exithandler`.

##### DagExecution — DSL If/Else/ElseIf

When working with Conditions in KFP, new nodes are introduced in the Pipeline Graph; they are prefixed with `condition-`
or `condition-branches-`.

1. **`Condition`** - Represents a conditional task group (i.e., per If/Else/ElseIf); in KFP it is represented by a DAG Driver that outputs a condition parameter which determines whether the underlying dag or components should execute.

2. **`ConditionBranch`** - Represents the branches that stem from a conditional statement (i.e., an If/Else wrapper).

Each of these results in a new dag execution. Instead of these executions, we will be switching to creating Tasks of types `ConditionBranch` and `Condition` respectively, and leverage the RunServer and ArtifactServer for input resolution.

##### Caching

###### Caching explained
To understand how caching should be handled in a post mlmd world, let's first review how caching in KFP works.

Caching has two parts. The first being Cache Fingerprint creations that happen in the Launcher, and the second is detecting Cache hit detections, which happens in Container Drivers.

1. At the end of Launcher `Execute()` procedure, there is a call to `l.clientManager.CacheClient().CreateExecutionCache(ctx, task)` which stores a `Task` with a `cache_fingerprint`. Underneath, this uses the `TaskServiceClient.CreateTaskV1` api, meaning this is execution data stored in the Task database table.

1. When the Container Driver runs, it does the following:

```go
if !opts.CacheDisabled {
    fingerPrint, cachedMLMDExecutionID, err := getFingerPrintsAndID(execution, &opts, cacheClient)
    if err != nil {
        return execution, err
    }
  ecfg.CachedMLMDExecutionID = cachedMLMDExecutionID
  ecfg.FingerPrint = fingerPrint
}
createdExecution, err := mlmd.CreateExecution(ctx, pipeline, ecfg)
```

The call to `getFingerPrintsAndID` makes a subsequent call to `TaskServiceClient.ListTasksV1` and fetches the execution ID for the Task with the fingerprint stored in the Launcher step. If such an execution ID is found, we assume there was a cache hit, and we don't run the next Launcher.

Notice also that we store the `ecfg.FingerPrint = fingerPrint` in the MLMD execution as well, this means the container execution also has the `cache_fingerprint` found in the task table.

###### Caching post mlmd removal

Much of the logic flow will stay the same, but instead of calls to `TaskServiceClient`'s v1 API, the v2 `RunService` api will be used.

When the Launcher finishes running `Execute()`, it `defers` an `UpdateDAGExecutionsState()` call, this can be replaced with the `UpdateTask` call using the v2 `RunServerClient`, providing the `cache_fingerprint` for this `Runtime` task. The fingerprint should only be provided upon a successful launcher execution.

In the Driver, `getFingerPrintsAndID` will be updated to leverage `ListTasks` and its `filter` field to search by `cache_fingerprint` to detect a hit, much like how it uses `ListTasksV1` today. Note that unlike how the Driver works today, the `cache_fingerprint` should not be stored for an upcoming task that will be created in the Driver, it should instead be updated by the Launcher once an execution successfully completes.

**Migration Note**

Caching needs special consideration for migration. Since the `tasks` table will be dropped and re-populated using data from MLMD, caching will need to be handled carefully. When converting a `ContainerExecution` to a `Runtime` task, we will need to only store the fingerprint if the execution has a `COMPLETE` status. This avoids storing fingerprints that may be present in an execution, but where an executor pod never ran to success.

### Launcher changes

Like the Driver component, the Launcher will need to establish new client connections to access the Runs API via RunServerClient and the Artifacts API via ArtifactServiceClient.

The following flags in the launcher will need to be removed:

```text
--execution_id
--mlmd_server_address
--mlmd_server_port
```

And replaced with:

```text
--task_id               # note that unlike the Driver we call this `task_id` instead of `parent_task_id`
--kfp_server_address
--kfp_server_port
```

This is a good opportunity to also replace the endpoints used in `cacheDefaultEndpoint` to use this address/port value, instead of relying on hardcoded defaults.

Other changes that will be required in Launcher are mentioned elsewhere in the proposal (see [Caching](#caching), and [Metrics](#metrics) sections).

### Nested Pipelines

There is no direct way to infer whether a Driver run is for a Nested execution, to accommodate this, there is a generic `DAG` task type provided to fit such cases.
Alternatively, we could provide an SDK update to declare a task type in a field on a `ComponentSpec` `dag` field.

### StoreSessionInfo

Currently, Artifact object storage credential info is stored as a custom property, and is called `store_session_info`. Storing such system level info as a custom property is an anti-pattern and should be avoided. In this effort, we will not port over this functionality, and we will instead remove the following from `root_dag.go`:

```go
storeSessionInfo, err = cfg.GetStoreSessionInfo(pipelineRoot)
```

And use it directly in `launcher_v2.go`, replacing:

```go
storeSessionInfo, err := objectstore.GetSessionInfoFromString(execution.GetPipeline().GetStoreSessionInfo())
```

Removing this property from the Artifact will also require Frontend changes, the server will now need to also parse the Launcher config, instead of the client code sending this as part of the call to:

```typescript
  // Apis.ts
  public static readFile({
    path,
    providerInfo,
    namespace,
    peek,
  }: { }
```

The server in [artifacts.tx] will instead need to build this object similar to how the root Driver does it in `cfg.GetStoreSessionInfo(pipelineRoot)`.

[artifacts.tx]: https://github.com/kubeflow/pipelines/blob/2c91fb797ed5e95bb51ae80c4daa2c6b9334b51b/frontend/server/handlers/artifacts.ts#L102

### Metrics

Metrics in KFP today are stored as Artifacts, they have the following Artifact Types:

* system.Metrics - Regular Key and NumberValue pair
* system.ClassificationMetrics - Key and JSON pair
* system.SlicedClassificationMetrics - Key and JSON pair

The values for these Metrics Artifacts are stored as `CustomProperties`; unlike other Artifacts, they are not stored in the object store.  Therefore, it is questionable that they are treated as Artifacts to begin with. Instead of porting this behavior, we'll instead leverage the Metrics table in KFP which is currently unused.

We will log the Metrics in this table when such artifact types are encountered in the Launcher. These can be addressed in `launcher_v2.go` when `uploadOutputArtifact` is called. During this invocation we can check for an artifacts type via::

```go
	schemaTitle := runtimeArtifact.Type.GetSchemaTitle()
	switch schemaTitle {
	case "system.Metrics":  // Handles Metric type, do something similar for ClassificationMetrics & SlicedClassificationMetrics
		err := LogMetric(...)
		...
    case "system.Artifact":
        err := RecordArtifact()
		...
```

In the executor Input we can abstain from storing a URI since this does not apply to Metrics.

The Python SDK will continue to interpret Metrics as artifacts, this maintains backwards compatibility. The Driver will need to ensure when it is creating the Artifacts list during the call to `resolveInputs -> resolveInputArtifact -> resolveUpstreamArtifacts() -> artifact.ToRuntimeArtifact()`, we are converting Metrics to output Artifacts. The updated pseudocode in `resolveUpstreamArtifacts` will be something like:

```go
package driver

func resolveUpstreamArtifacts(cfg resolveUpstreamOutputsConfig) (*pipelinespec.ArtifactList, error) {
  for {
    ...
  } else {
    // use the Component *pipelinespec.ComponentSpec.ComponentInputsSpec from Options in driver.go to determine 
	// artifact schema type, 
    schemaTitle := determineArtifactSchema(ComponentInputSpec, TaskSpec)
    switch schemaTitle {
    case "system.Metrics":  // Handles Metric type, do something similar for ClassificationMetrics & SlicedClassificationMetrics
	  // GetOutputMetricsByTaskID can fetch the Task via GetTask (if we don't already have the task), 
	  // and can parse the `output_metrics` to return map[string]*OutputArtifact or just the *OutputArtifact
      outputs, err := GetOutputMetricsByTaskID(cfg.ctx, taskID)
    case "system.Artifact":
      outputs, err := GetOutArtifactsByTaskID(cfg.ctx, taskID)
  }
}
```

### Frontend Changes

There are three primary pages in the UI where MLMD encounters happen, the Run Details, Artifacts, and Executions pages.

The Executions page can be removed entirely since all information will be present in the run task nodes in the run graph. The FrontEnd will need to be updated to ensure all relevant information will continue to be surfaced in a Task Node's details sidebar. 

On the Run Details page, MLMD data is fetched in `RuntimeNodeDetailsV2.tsx` via:

```typescript
const context = await getKfpV2RunContext(runId);
const executions = await getExecutionsFromContext(context);
const artifacts = await getArtifactsFromContext(context);
const events = await getEventsByExecutions(executions);
```

These can be replaced by the following new implementations:

```typescript
// context no longer needed, use the Run object which often readily available wherever context is required
const tasks = run.run_details.task_details;  // run is a V2beta1Run
const artifacts = await fetchArtifactsFromTasks(tasks); // the information is now available in the task.
// a separate call for this may not needed, as the required info may already be present in `tasks`
const events = await getArtifactEventsByTasks(tasks); // uses ListArtifactEvents()
```

The `Visualization` Nav in `RuntimeNodeDetailsV2.tsx` will also need to be updated to take `Metrics` fetched from `Task` response object, instead of `Artifacts` from MLMD.

The Artifact Node in the UI should also no longer display an `Artifact URI` for metrics, as this is not applicable.

The `CompareV2.tsx` also makes various calls to MLMD, much like `RuntimeNodeDetailsV2`:

```typescript
      Promise.all(
        runIds.map(async runId => {
          const context = await getKfpV2RunContext(runId);
          const executions = await getExecutionsFromContext(context);
          const artifacts = await getArtifactsFromContext(context);
          const events = await getEventsByExecutions(executions);
          return {
            executions,
            artifacts,
            events,
          } as MlmdPackage;
        }),
      )
```

This and underlying code will also need to be updated to leverage Tasks retrieved via the Runs API server.

#### Run Reporting

The Persistence Agent calls the KFP API Server's [report_server.go] for updating the Run metadata in the DB. This includes updates to the Task metadata in the DB as well.

Because we are relying on Driver/Launcher to create/update tasks, we will no longer require Persistence Agent to report on task details, and we will need to get rid of this portion of the code.

This is the key piece of code from `report-server.go`:

```go
_, err = s.reportTasksFromExecution(newExecSpec, runId)
```
[report_server.go]: ../../backend/src/apiserver/server/report_server.go

### Task States 

We will follow the following method for handling Task states: 

RuntimeStates
* Tasks will always be in a subset of the [RuntimeStates](../../backend/api/v2beta1/run.proto)
* When the parent run is in a terminal state, don't allow updates to the underlying tasks associated with that run
  * Terminal States are: `SUCCEEDED`, `FAILED`, `CANCELED`

StorageStates
* Tasks don't need a storage state 
* If a Run is deleted, all tasks should be deleted

### Auth Considerations

The Driver/Launcher will be introducing a new `RunServerClient` and `ArtifactServerClient` using the `v2beta1`. All calls to this endpoint must be protected via SubjectAccessReview. The new server implementations can simply use `resourceManager.IsAuthorized(ctx, resourceAttributes)`, which is the standard everywhere else in KFP. All tasks/artifacts/metrics endpoints will be doing SAR on the `run` resource. If a user makes a REST request with a `verb` that matches their permissions on the `Run` KFP resource, they will be authorized to perform that action.

For example, if a user makes a request to `ListArtifactRequest`, they require `list` verb on the `Run` resource for that particular namespace.

A few more notes: 
* the Driver/Launcher communicates with the KFP API Server via the CacheClient. This has no auth mechanism today and will need to be updated.
* the Driver/Launcher will provide the Pipeline Runner's Service Account token in the auth header for authorization. 
  * As such, the Pipeline Runner SA will need the appropriate namespace-level access to such resources for the Driver & Launcher to communicate with the API Server.

### Manifests

The following changes will need to be made:

* For Driver/Launcher authentication purpose, the Pipeline Runner SA rbac will need to be updated accordingly to support basic verbs on the Artifact resource.
* Envoy manifests will be removed
* MLMD manifests will need to be removed
* Any configmaps, env vars, or such fields referencing MLMD will need to be removed

### Migration

This change will come with some drastic changes to the DB schema, namely the `Tasks` table. We will be dropping this table entirely. The only usage this table sees is described in the [caching](#caching) section. As noted there, all information that is relevant already exists in MLMD. 

To accommodate the transition, the KFP release containing this change will provide a migration script for users to apply to their DB. MLMD will be required so that the script may use the mlmd client. The script will do the following: 

* Drop the Tasks table and recreate it
* Drop the Metrics table (it is not used at all)
* Scan MLMD executions, converting them to their Task counterparts.
  * When encountering ContainerExecutions with `cache_fingerprints`, the fingerprint should only be stored if the execution has a `COMPLETE` state.
  * To detect exit handler dags, the execution name will need to be parsed for `exit-handler-*` prefixed, as there's no other declarative way to determine this type.
* Scan all `Artifacts` and recreate in the KFP artifact table. 
* In the case of metrics, artifacts will need to be logged to the `Metrics` table instead of `Artifacts`.
* Validation Step

Due to the nature of the change, we will require users to opt in to this upgrade by running this script. If the API Server detects the new fields are not present, KFP will assume the migration script has not been executed, and thus the server will fail to start up, logging a meaningful message to the user.

#### Migration Alternative 

An alternative to this migration strategy is to have the KFP server perform the migration. We can enable opt-in by having a one time API Server config option `mlmdMigrate=true`.

The benefit of this approach is a more seamless migration that's automated. However, if a user wants to have more granular control of their migration, they may prefer the script method which they can adjust as needed.

There is also the option of doing a hybrid approach at the cost of more overhead. 

### Testing

1. Unit Tests
- Driver/Launcher unit tests will need to be updated to use KFP server instead of MLMD
- API server unit tests for new task and artifact endpoints
- Frontend unit tests for updated components using task data instead of MLMD

2. Integration Tests
- Existing integration tests will verify no regressions
- With the change to metrics handling, we may need more rigorous testing if current sample pipelines are insufficient
- Similarly, more testing around data passing might be required

3. Migration Tests (manual but accompanied by supporting evidence)
- Testing of a migration script
- Previous and post upgrade mysql dumps should be provided
- Verify old cached pipelines continue to be cached post-upgrade
- Verify recurring runs pre-upgrade and post-upgrade continue to work
- Test running old and new pipelines before and after migration

4. Security Tests (requires multi-user mode)
- Test authentication/authorization for new API endpoints
- Verify proper RBAC enforcement for task/artifact operations
- Test multi-tenant isolation of tasks and artifacts

5. Frontend Verification Tests
- Verify frontend reporting of metrics in the "Artifact Info" and "Visualization" navs in run details
- Verify the frontend comparison UI, confirming artifacts and metrics are fetched accordingly 

6. Performance Testing 
- Monitor and assess changes in CI times
- Load testing on Kubernetes clusters before/after mlmd removal