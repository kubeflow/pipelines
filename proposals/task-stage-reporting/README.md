# Task Stage Reporting — Implementation Plan (code-grounded)

This document augments [README.original.md](./README.original.md) with a design that has been
reconciled against the **current** state of the codebase. The README captures
the intent and user-facing contract; this file records what actually exists
today, where the README's assumptions diverge from the code, and the concrete,
phased plan to implement the feature.

> [!NOTE]
> The feature is **buildable today**. The [mlmd removal](../12147-mlmd-removal/README.md)
> work this proposal strictly depends on has already landed: the native task
> store, the `UpdateTask` RPC, the `StatusMetadata.custom_properties` field, and
> the `GetRun?view=FULL` path that returns full task details are all present. No new protos, RPCs, or
> database columns are required.

## Dependency status: mlmd removal has landed

Evidence in-tree:

- `StatusMetadata.custom_properties` exists in the canonical proto
  [`backend/api/v2/run.proto:466`](../../backend/api/v2/run.proto#L466-L475),
  typed `map<string, google.protobuf.Value>`, with the comment *"this can be used
  to provide additional status info for a given task during runtime … This is
  currently not utilized by KFP backend."*
- The launcher already uses the post-mlmd flags `task_id`, `parent_task_id`,
  `run_id`, `ml_pipeline_server_address`/`ml_pipeline_server_port`
  ([`backend/src/v2/cmd/launcher-v2/main.go:36`](../../backend/src/v2/cmd/launcher-v2/main.go#L36-L45)),
  not the old `execution_id`/`mlmd_server_address`.
- The task/run RPCs (`CreateTask`, `UpdateTask`, `UpdateTasksBulk`, `GetTask`,
  `ListTasks`, `FindCachedTask`) are implemented on `RunService`
  ([`backend/src/apiserver/server/run_server.go`](../../backend/src/apiserver/server/run_server.go)).
- `GetRun(view=FULL)` loads and returns the full task list including
  `status_metadata.custom_properties`
  ([`run_server.go:328`](../../backend/src/apiserver/server/run_server.go#L328-L359)).

## What is reused as-is

| Mechanism | Location | Notes |
|---|---|---|
| `StatusMetadata.custom_properties` | [run.proto:466-475](../../backend/api/v2/run.proto#L466-L475) | Go type is `map[string]*structpb.Value`, **not** a `Struct`. |
| `UpdateTask` RPC + authorization | [run_server.go:520](../../backend/src/apiserver/server/run_server.go#L520) | `PATCH /runs/{run_id}/tasks/{task_id}`; SAR `update` on the `Run` resource. |
| Selective, row-locked persistence | [task_store.go:1060-1265](../../backend/src/apiserver/storage/task_store.go#L1060-L1265) | `SELECT … FOR UPDATE`; `StatusMetadata` written as a whole-column JSON replace only when non-nil ([1166-1172](../../backend/src/apiserver/storage/task_store.go#L1166-L1172)). |
| Authenticated API client in launcher | [kfpapi/api.go:94-96](../../backend/src/v2/apiclient/kfpapi/api.go#L94-L96), [client_manager.go:13-16](../../backend/src/v2/client_manager/client_manager.go#L13-L16) | `l.clientManager.KFPAPIClient().UpdateTask(...)` already exists ([launcher_v2.go:354](../../backend/src/v2/component/launcher_v2.go#L354)). |
| Retrieval via `GetRun(view=FULL)` | [run_server.go:328-359](../../backend/src/apiserver/server/run_server.go#L328-L359), [run_store.go:471-533](../../backend/src/apiserver/storage/run_store.go#L471-L533), [api_converter.go:1895-1906](../../backend/src/apiserver/server/api_converter.go#L1895-L1906) | Stages come back with no server changes. |

## Corrections to the README design

Two of the README's Layer-1 (launcher) assumptions do not match the current code
(Corrections 1 and 2); a third is directionally right — the in-memory stage map
and the need to preserve stages on failure are both correct — but states the
wrong mechanism (Correction 3). Each is addressed in the plan below.

### Correction 1 — there is no direct `exec.Cmd`; execution is behind an interface using a blocking `Run()`

The README says to pass the write-end "via `exec.Cmd.ExtraFiles`." The launcher
never touches `exec.Cmd` directly. It calls
[`CommandExecutor.Run(ctx, cmd, args, stdin, stdout, stderr)`](../../backend/src/v2/component/interfaces.go#L50-L55),
whose production implementation uses the **blocking** `command.Run()` with **no
`ExtraFiles` and no `Start()`/`Wait()` split**
([interfaces.go:112-118](../../backend/src/v2/component/interfaces.go#L112-L118)).

Implications:

- Passing FD 3 requires threading an `extraFiles []*os.File` through the
  `CommandExecutor.Run` signature. That ripples to `RealCommandExecutor`,
  `MockCommandExecutor` ([mocks.go:187](../../backend/src/v2/component/mocks.go#L187)),
  and the ~8 test call sites that inject a mock via `WithCommandExecutor`
  (they pass `nil`).
- **EOF is subtler than the README states.** With `ExtraFiles`, the *parent*
  process also holds a copy of the write-end; the reader goroutine only sees EOF
  once *every* write-end is closed. Because `Run()` blocks and offers no
  "after Start" hook, the correct lifecycle is:
  1. Launcher creates `r, w := os.Pipe()`.
  2. Launcher starts the reader goroutine on `r`.
  3. Launcher passes `w` as the single `ExtraFiles` entry (→ FD 3 in the child).
  4. `cmdExecutor.Run(...)` blocks until the child exits.
  5. **After `Run()` returns, the launcher closes the parent's `w`**, which lets
     the goroutine drain any buffered lines and then observe EOF.
  6. Launcher joins the goroutine, then closes `r`.

  (An alternative is to split `RealCommandExecutor` into `Start()`/`Wait()` for
  true EOF-at-child-exit; it is cleaner semantically but a larger refactor.
  See Decision 1.)

### Correction 2 — the concurrent goroutine must write directly, not through the `BatchUpdater`

Normal launcher task writes are accumulated by a **`BatchUpdater`** and flushed
at the end of `executeV2()` and in `finalizeExecution()`
([batch_updater.go](../../backend/src/v2/component/batch_updater.go)). It is
**not thread-safe** and is flushed on the main goroutine, so it is the wrong tool
for the concurrent stage goroutine.

The stage goroutine should call `UpdateTask` **directly** (the pattern already
exists at [launcher_v2.go:354](../../backend/src/v2/component/launcher_v2.go#L354)),
sending a **minimal** `PipelineTask{TaskId, RunId, StatusMetadata{custom_properties}}`
so the store's selective update touches only the `StatusMetadata` column and
leaves `State`, parameters, pods, etc. untouched.

The README's claim that the goroutine is the "sole writer for the duration of
the component" does hold in practice: during `cmdExecutor.Run(...)` the main
goroutine is blocked, and no other code path writes this task's
`StatusMetadata` in that window.

### Correction 3 — Risk #3's `finalizeExecution` change is unnecessary

The design holds the full `StatusMetadata` in memory: the goroutine accumulates
stages into `l.options.Task.StatusMetadata.CustomProperties["stages"]`, and that
in-memory object is what gets persisted. Risk #3 proposes changing
`finalizeExecution` to "merge rather than replace" so stages survive a failed
task — but that change is **not needed**, because the current code already
preserves them
([launcher_v2.go:293-304](../../backend/src/v2/component/launcher_v2.go#L293-L304)):

- On **success** (`executionErr == nil`) the `if` block (lines 294-300) is skipped
  entirely — `finalizeExecution` never touches `StatusMetadata`.
- On **failure** it only sets `.Message` *in place*, leaving `custom_properties`
  intact.

So once the goroutine has accumulated the stages into the in-memory
`StatusMetadata` (and is joined before finalize — see Correction 1), both the
end-of-`executeV2` flush and `finalizeExecution` persist exactly that in-memory
version. The store's whole-column write of `StatusMetadata`
([task_store.go:1166-1172](../../backend/src/apiserver/storage/task_store.go#L1166-L1172))
is precisely what we want: the finalized in-memory copy becomes the DB copy. No
store merge, no change to `finalizeExecution`.

Risk #3's instinct is sound — keep the stages in memory and make sure the
finalized version lands in the DB. Only its claim that `finalizeExecution`
"replaces `StatusMetadata` wholesale" is inaccurate, which makes the mitigation it
proposes moot.

## Design details (revised)

### Layer 1: Launcher (Go)

Files: [`backend/src/v2/component/`](../../backend/src/v2/component/)

- Extend `CommandExecutor.Run` with an `extraFiles []*os.File` parameter; set
  `command.ExtraFiles` in `RealCommandExecutor`; update `MockCommandExecutor`
  and the 5 test call sites that inject it via `WithCommandExecutor` (pass `nil`).
- In [`execute()`](../../backend/src/v2/component/launcher_v2.go#L802-L859):
  create the pipe, start the reader goroutine (tracked by a `sync.WaitGroup`),
  pass the write-end as the single `ExtraFiles` entry (first entry ⇒ FD 3 in
  the child), and after `cmdExecutor.Run` returns close the parent write-end
  and `wg.Wait()` the goroutine. The join is near-instant (the goroutine is
  just draining the last buffered lines after EOF) but provides a happens-before
  guarantee: the main goroutine must not read or flush `StatusMetadata` until
  after the reader goroutine has finished its final map write and RPC.
- New file `stage_reader.go`: the goroutine scans JSONL from the read-end,
  accumulates each stage (keyed by the `stage` field) into the in-memory
  `l.options.Task.StatusMetadata.CustomProperties["stages"]`. Because
  `CustomProperties` is `map[string]*structpb.Value`, the stages value must be
  wrapped: `CustomProperties["stages"] = structpb.NewStructValue(stagesStruct)`,
  where `stagesStruct` is a `*structpb.Struct` mapping stage names to their
  detail structs. The goroutine issues a direct `UpdateTask` per update for live
  visibility (Correction 2).
  Malformed lines are discarded. Because the stages live on the shared in-memory
  `StatusMetadata`, the normal end-of-`executeV2` flush and `finalizeExecution`
  persist them unchanged (Correction 3). Per-line updates match the README;
  batching is its own future optimization (README Risk #4).

### Layer 2: SDK (Python)

Files: [`sdk/python/kfp/`](../../sdk/python/kfp/)

- New module `kfp/task_stage.py`, **stdlib-only** (`os`, `json`, `sys`, `errno`).
  Runtime containers install `kfp` with `--no-deps` and set `_KFP_RUNTIME=true`,
  which disables most SDK imports, so this module must pull in no heavy
  dependencies.
- `set_task_stage(stage, message, state="RUNNING", progress=None, **custom)`:
  lazily `os.fdopen(3, "w")` inside `try/except OSError` (→ silent no-op in
  local/test mode where FD 3 is absent); write one JSON line and flush; catch
  and suppress `BrokenPipeError` (launcher already exited).
- Export it **unconditionally** in
  [`kfp/__init__.py`](../../sdk/python/kfp/__init__.py) — *outside* the
  `_KFP_RUNTIME` guard at lines 31-37 — following the runtime/compile-time split
  pattern in [`kfp/dsl/__init__.py`](../../sdk/python/kfp/dsl/__init__.py).
  Placing it inside the guard would make it unimportable at runtime, which is the
  only place it runs. The user calls it from within their component function
  (user code runs at [`executor.py:483`](../../sdk/python/kfp/dsl/executor.py#L483));
  no executor change is required.

### Wire format

Each line written to FD 3 is a JSON object: `stage` (string, required, used as
the map key), `message` (string, required), `state`
(`"RUNNING"|"COMPLETED"|"FAILED"|"SKIPPED"`), plus any additional keys passed
through as-is. The launcher nests all stages under a single
`custom_properties["stages"]` key whose value is a `Struct` mapping stage name →
`{message, state, …}`.

### Stage retrieval

No server changes are needed and no new endpoint is introduced. Stages live in
`status_metadata.custom_properties["stages"]` and are retrieved through existing
GET endpoints: `GetRun(view=FULL)` returns every task's `status_metadata`, and
`GetTask` (`GET /runs/{run_id}/tasks/{task_id}`) / `ListTasks` return it per task
([run_server.go:663-757](../../backend/src/apiserver/server/run_server.go#L663-L757)).
This resolves the README's "to be discussed" note: a dedicated per-task stages
endpoint is not required, because `GetTask` already serves the stages for a single
task.

## KFP Local considerations

Unchanged from the README: in local mode (`SubprocessRunner`/`DockerRunner`) FD 3
is not inherited — the subprocess handler does not set `pass_fds`
([subprocess_task_handler.py:108-136](../../sdk/python/kfp/local/subprocess_task_handler.py#L108-L136)),
so `set_task_stage()` detects the missing FD via `OSError` and no-ops.

## Open decisions

The only genuine open choice is **how to pass FD 3 to the child**, since command
execution is behind the `CommandExecutor` abstraction (Correction 1):

- **Recommended:** add an `extraFiles []*os.File` parameter to
  `CommandExecutor.Run`, set `command.ExtraFiles` in `RealCommandExecutor`, and
  close the parent write-end after `Run()` returns, then `wg.Wait()` the reader
  goroutine before proceeding.
- **Alternative:** split `RealCommandExecutor` into `Start()`/`Wait()` for
  EOF-at-child-exit semantics — cleaner, but a larger refactor.

Everything else is settled by the design and needs no decision: stages live in
`custom_properties["stages"]`, are held in the in-memory `StatusMetadata` and
persisted by the normal finalize path (no store or `finalizeExecution` change),
and are read back via the existing GET endpoints (no new endpoint).

## Delivery plan

**Phase 1 — Launcher plumbing (Go)**
1. Add `extraFiles []*os.File` to `CommandExecutor.Run`; implement in
   `RealCommandExecutor`; update `MockCommandExecutor` + 5 test call sites.
2. Wire the pipe + reader goroutine (`sync.WaitGroup`-tracked) into
   `execute()`, with the post-`Run()` write-end close and `wg.Wait()` join.
3. Add `stage_reader.go`: JSONL parse, accumulate into the in-memory
   `l.options.Task.StatusMetadata.CustomProperties["stages"]`, and issue direct
   `UpdateTask` calls for live visibility. The normal finalize path persists the
   in-memory copy — no `finalizeExecution` or store change.

**Phase 2 — SDK (Python)**
4. Add `kfp/task_stage.py` (stdlib-only) and export `set_task_stage`
   unconditionally from `kfp/__init__.py`.

**Phase 3 — Tests**
5. Go unit tests: valid JSONL, malformed lines, rapid writes, EOF/close, and the
   failure-path preservation case.
6. Python unit tests (unittest, modeled on
   [`kfp/init_test.py`](../../sdk/python/kfp/init_test.py)): real pipe FD, closed
   FD (`BrokenPipeError`), no FD 3 (no-op), and importability under
   `_KFP_RUNTIME=true`.
7. Integration test: a multi-stage component asserting `GetRun(view=FULL)`
   returns the expected `custom_properties.stages`.

No proto or database regeneration is required, so the heavy `make generate` step
is avoided entirely.
