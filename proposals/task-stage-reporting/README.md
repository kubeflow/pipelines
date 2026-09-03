# Task Stage Reporting

This proposal adds the ability for running components to report their current stage back to the KFP API server, enabling visibility into long-running tasks.

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
3. API Server Load
    - Risk: Sequential processing of many queued updates under high concurrency could strain the API server.
    - Mitigation: If the sequential approach proves problematic, a future optimization could batch-write multiple updates in a single call.

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

The reader goroutine scans JSONL from the pipe and enqueues each valid update. Updates are processed sequentially — every update is sent to the API server via `UpdateTask`, none are dropped or overwritten. Malformed lines are silently discarded. On EOF (child exit), any remaining queued updates are flushed.

### Layer 2: SDK (Python)

A public function `kfp.set_task_stage(message, progress=None, **custom)` is exposed at the `kfp` package level. It opens FD 3 lazily, writes a JSON line per call, and flushes immediately. If FD 3 is not available (local mode, tests), it silently no-ops. If the launcher has already exited, `BrokenPipeError` is caught and suppressed.

### Wire Format

Each line written to FD 3 is a JSON object with the following fields:

- `message` (string, required): Human-readable stage description.
- `progress` (float, optional): Fractional progress in [0.0, 1.0].
- Additional keys are passed through as-is to `custom_properties`.

The launcher maps these to `PipelineTask.StatusMetadata.custom_properties` values using the existing proto `map<string, google.protobuf.Value>` field.

### Existing Infrastructure Reused

- **`StatusMetadata.custom_properties`** (`run.proto:467`): Already defined in the proto with the comment "can be used to provide additional status info for a given task during runtime." Already persisted by the task store.
- **`UpdateTask` RPC** (`run.proto:143`): Fully wired gRPC endpoint with authorization, merge-style partial updates, and automatic `state_history` population.
- **`TaskStore.UpdateTask`** (`task_store.go:1018`): Handles `StatusMetadata` marshaling at line 1119 with row-level locking for safe concurrent updates.

No new proto fields, RPCs, or database columns are needed.

## KFP Local Considerations

In local mode (SubprocessRunner / DockerRunner), FD 3 is not available. `set_task_stage()` detects this via the `OSError` from `os.fdopen(3)` and silently no-ops. Component code does not need conditional logic.

A future enhancement could have the local runner create the pipe and print stage updates to the terminal, but this is out of scope for this proposal.

## Test Plan

1. **Unit tests (Go)**: Test the pipe reader goroutine with mock pipes — valid JSONL, malformed lines, rapid writes (sequential queue ordering), and EOF handling.
2. **Unit tests (Python)**: Test `set_task_stage()` with a real pipe FD, with a closed FD (BrokenPipeError), and without FD 3 (local mode no-op).
3. **Integration test**: A pipeline with a component that calls `set_task_stage()`, asserting that `GetTask` returns the expected `custom_properties` values after the component completes.
4. **Load test**: Multiple concurrent tasks reporting status at high frequency to verify sequential queue processing handles sustained throughput.

## Delivery Plan

1. Add the pipe creation and reader goroutine to the launcher.
2. Add `kfp.set_task_stage()` to the SDK.
3. Add unit tests for both layers.
4. Add an integration test with a status-reporting component.
