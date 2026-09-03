# Task Stage Reporting

This proposal adds the ability for running components to report their current stage back to the KFP API server, enabling visibility into long-running tasks.

> [!NOTE]
> This proposal is built on top of [mlmd removal](../../proposals/12147-mlmd-removal/README.md) and strictly depends on it. 


## Motivation

KFP components can only report status at completion — either success with outputs or failure with an error. For long-running components (large model training, data processing pipelines, multi-step ETL), users have no way to see what the component is currently doing. The only visible states are `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `CACHED`, `SKIPPED`, `CANCELING`, `CANCELED`, and `PAUSED` — none of which convey progress within a running task.

## User Stories

- As a Data Scientist, I want to see what stage my training component is at (e.g., "Epoch 7/50, loss=0.23") so I can decide whether to wait or cancel.
- As an ML Engineer, I want my multi-step data processing component to report which step it is on so I can estimate completion time.
- As a Platform Operator, I want visibility into long-running pipeline tasks without having to inspect pod logs.

## Risks and Mitigations

1. Pipe Lifecycle
    - Risk: The user process crashes or hangs without closing the pipe.
    - Mitigation: The pipe read-end returns EOF when the child process exits (OS closes the write-end FD). The launcher goroutine exits cleanly on EOF. If the launcher is killed, the write-end becomes a broken pipe and the SDK helper catches `BrokenPipeError` silently.
2. Malformed Input
    - Risk: User code writes arbitrary data to FD 3.
    - Mitigation: The launcher discards lines that fail JSON parsing. Only recognized fields are forwarded to `StatusMetadata.custom_properties`.
3. Failure Path Overwrites Stages
    - Risk: The launcher's `finalizeExecution` replaces `StatusMetadata` wholesale on failure — setting `StatusMetadata.Message` to the error string and discarding prior content, including accumulated stages.
    - Mitigation: Change `finalizeExecution` to merge rather than replace: read the existing `custom_properties` from the in-memory map, construct a new `StatusMetadata` with both the error `message` and the accumulated `custom_properties`, then write. This preserves stage history on failed tasks, which is the most useful case for debugging.
4. API Server Load
    - Risk: Each stage update triggers an `UpdateTask` call with the full accumulated `custom_properties` map.
    - Mitigation: The map is small (bounded by the number of stages, typically single digits). The `UpdateTask` write is also selective persisting only non-nil fields. If the per-update call proves problematic under high concurrency load, a future optimization could include batch-writing resulting in less write operations but delayed stages visibility in the GET API calls. 

## Design Details

### Mechanism: OS Pipe Between Launcher and User Process

The launcher (Go) creates an `os.Pipe()` before spawning the user process. The write-end is passed to the child via `exec.Cmd.ExtraFiles`, which makes it available as file descriptor 3 in the child process (the first entry in `ExtraFiles` is always FD 3, per Go's `os/exec` documentation). The launcher reads JSONL from the read-end in a background goroutine.

This approach:
- Avoids polluting stdout/stderr with control messages.
- Requires no file polling, sidecar containers, or additional environment variables.
- Cleans up automatically — the OS closes FDs when the process exits.
- Works within the existing pod structure (launcher and user code run in the same container).

### Layer 1: Launcher (Go)

In `LauncherV2.executeV2()`, before the user command starts, the launcher creates an `os.Pipe()` and passes the write-end to the child via `cmd.ExtraFiles`. After starting the command, it closes the write-end in the parent and starts a reader goroutine.

The reader goroutine scans JSONL from the pipe. It maintains an in-memory `map[string]*structpb.Value` that accumulates all stages reported so far. On each valid line, the goroutine upserts the stage entry (keyed by the `stage` field) into the map, then calls `UpdateTask` with the full accumulated map nested under `StatusMetadata.custom_properties["stages"]`. Malformed lines are silently discarded. On EOF (child exit), any remaining queued update is flushed.

During execution, no other code path writes to `StatusMetadata` — the launcher only touches it on failure (in `finalizeExecution`) or on plugin start (in the driver), both of which happen outside the user command's runtime window. The goroutine is the sole writer for the duration of the component.

### Layer 2: SDK (Python)

A public function `kfp.set_task_stage(stage, message, state="RUNNING", progress=None, **custom)` is exposed at the `kfp` package level. It opens FD 3 lazily, writes a JSON line per call, and flushes immediately. If FD 3 is not available (local mode, tests), it writes the stage-reporting lines to stdout along with the component logs. If the launcher has already exited, `BrokenPipeError` is caught and suppressed.

### Wire Format

Each line written to FD 3 is a JSON object with the following fields:

- `stage` (string, required): Stage identifier, used as the key in `custom_properties` (e.g., `"load_data"`, `"train"`).
- `message` (string, required): Human-readable stage description.
- `state` (string, required): One of `"RUNNING"`, `"COMPLETED"`, `"FAILED"`, `"SKIPPED"`.
- Additional keys are allowed and passed through as-is into the stage's value dict.

The launcher accumulates these into a `map[string]*structpb.Value` and writes it to `PipelineTask.StatusMetadata.custom_properties` via `UpdateTask` under a single `stages` key. The value of `custom_properties["stages"]` is a `Struct` where each key is a stage name and each value is a `Struct` containing `message`, `state`, and any additional keys.

### Existing Infrastructure Reused

- **`StatusMetadata.custom_properties`**: Already defined in the proto with the comment "can be used to provide additional status info for a given task during runtime." Already persisted by the task store.
- **`UpdateTask` RPC**: Fully wired gRPC endpoint with authorization, merge-style partial updates, and automatic `state_history` population.
- **`TaskStore.UpdateTask`**: Handles `StatusMetadata` marshaling with row-level locking for safe concurrent updates.

No new proto fields, RPCs, or database columns are needed.

### Stage Retrieval

Stages are retrieved through the existing `GET /apis/v2beta1/runs/{run_id}?view=FULL` endpoint. When called with `view=FULL`, the response includes the full `tasks` list with each task's `status_metadata.custom_properties` populated from the database. No additional endpoint or query parameter is needed.

It is to be discussed whether a new dedicated endpoint for getting stages for a particular task is required and viable to implement and maintain.

The response shape for a task with reported stages:

```json
{
  "tasks": [
    {
      "task_id": "abc-123",
      "display_name": "train-model",
      "state": "RUNNING",
      "status_metadata": {
        "custom_properties": {
          "stages": {
            "load_data": {
              "state": "COMPLETED",
              "message": "Loaded 50k rows"
            },
            "validate": {
              "state": "COMPLETED",
              "message": "Schema valid"
            },
            "train": {
              "state": "RUNNING",
              "message": "Epoch 7/50, loss=0.23",
              "progress": 0.14
            }
          }
        }
      }
    }
  ]
}
```

Tasks that never reported stages have no `stages` key in `custom_properties`. Consumers check for the presence of `custom_properties.stages` to determine whether a component reported any stages. The `UpdateTask` store method writes `StatusMetadata` as a selective column update — only the `StatusMetadata` column is touched, under a row-level lock (`SELECT ... FOR UPDATE`), so stage writes do not interfere with other task mutations.

## KFP Local Considerations

In local mode (SubprocessRunner / DockerRunner), FD 3 is not available. `set_task_stage()` detects this via the `OSError` from `os.fdopen(3)` and silently no-ops. Component code does not need conditional logic.

A future enhancement could have the local runner create the pipe and print stage updates to the terminal, but this is out of scope for this proposal.

## Test Plan

1. **Unit tests (Go)**: Test the pipe reader goroutine with mock pipes — valid JSONL, malformed lines, rapid writes (sequential queue ordering), and EOF handling.
2. **Unit tests (Python)**: Test `set_task_stage()` with a real pipe FD, with a closed FD (BrokenPipeError), and without FD 3 (local mode no-op).
3. **Integration test**: A pipeline with a component that calls `set_task_stage()` for multiple _parallel_ stages, asserting that `GetRun(view=FULL)` returns the expected `custom_properties` map on the task — with all stages present and correct states.
4. **Failure path test**: A component that reports stages then fails, asserting that the failed task's `status_metadata` contains both the error `message` and the accumulated `custom_properties` stages.

## Delivery Plan

1. Add the pipe creation and reader goroutine to the launcher, with in-memory stage accumulation and `UpdateTask` persistence.
2. Merge (not replace) `StatusMetadata` in `finalizeExecution` to preserve stages on failure.
3. Add `kfp.set_task_stage()` to the SDK.
4. Add unit tests for both layers, including the failure path merge.
5. Add an integration test with a multi-stage component, verifying retrieval via `GetRun(view=FULL)`.
