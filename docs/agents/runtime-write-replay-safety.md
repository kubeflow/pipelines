# Runtime write replay safety

The native runtime (driver and launcher) writes to the API server over gRPC.
A write can commit on the server while its response is lost, so a client that
sends the same write again must not corrupt state. This page classifies every
runtime write by what a repeated delivery does. It is the reference for deciding
whether a write may ever be retried automatically.

Reads are always safe to repeat and are not listed.

## Classification

- **Replay-safe**: a repeated delivery of the same operation converges on the
  original result and creates no duplicate.
- **Conditionally safe**: replay is safe only when a stated prerequisite holds,
  or only for an identical payload.
- **Unsafe**: a repeated delivery can create a duplicate or overwrite newer state.

| Write | Class | What a repeated delivery does |
| --- | --- | --- |
| `CreateArtifact` with `operation_id` | Replay-safe | Same operation and same payload returns the stored artifact and link. The same operation with a different payload is an explicit conflict. A different `operation_id` is a distinct write. |
| `CreateArtifact` without `operation_id` | Unsafe | Each delivery allocates a new artifact ID, so a replay after a lost response creates a second artifact and link. |
| `CreateArtifact` with `reuse_if_exists` (importer) | Conditionally safe | Creates the artifact only if none in the same namespace matches the stable reuse identity (type, URI, name, description and metadata); concurrent callers share one row and each still gets its own link. This cannot be combined with `operation_id`; the server rejects the combination. |
| `CreateArtifactsBulk` with `operation_id` on every artifact | Replay-safe | The batch is all-or-nothing. Each artifact is deduplicated by its own operation identity. `reuse_if_exists` is rejected in bulk requests. |
| `CreateArtifactsBulk` without `operation_id` | Unsafe | Same as `CreateArtifact` without an operation ID. |
| `CreateArtifactTask`, `CreateArtifactTasks` | Replay-safe | A repeated delivery of the same logical link (artifact, task, type, iteration, artifact key) is treated as success and creates no second link. |
| `CreateTask` | Conditionally safe | Deduplicated by logical identity: run, parent task, scope path, name, type and iteration index. The existing task is returned unchanged; a different payload under the same identity is not reported as a conflict. When `scope_path`, `name` or `type` is missing there is no logical identity and every delivery creates a new task. |
| `UpdateTask` | Conditionally safe | Fields merge by key and state history grows only when the state changes, so replaying an identical update is idempotent. There is no revision fencing: replaying a stale update after a newer one can overwrite newer fields, including the state. Task revisions and terminal-state immutability are tracked in #14032 and attempt fencing in #14322. The server also rejects task updates for runs that no longer allow runtime writes. |
| `UpdateTasksBulk` | Conditionally safe, not atomic | See below. |

## `UpdateTasksBulk` partial completion

`UpdateTasksBulk` validates the whole request first (run scope, task ownership,
authorization). It then applies the updates one task at a time in sorted task-ID
order, each in its own transaction. Therefore:

- A request that fails validation changes nothing.
- A failure while applying task *N* leaves tasks 1 to *N-1* committed and returns
  an error with no response body. Callers must not treat an error as "nothing
  changed".
- Because each member update is idempotent for an identical payload, sending the
  corrected request again converges: updates that already committed are accepted
  again and the remaining updates are applied. A client that retries a whole
  batch must retry with the same payload for the members that already succeeded.

`TestUpdateTasksBulk_FailureAfterFirstUpdateIsPartialAndReplaySafe` and
`TestUpdateTasksBulk_ValidationFailureLeavesEveryTaskUnchanged` pin this
behavior.

## Eligibility for automatic retries

Only operations classified as replay-safe, with the prerequisite satisfied, may
be retried automatically by the runtime client. Conditionally safe writes must
not be retried until their prerequisite or fencing is guaranteed by the caller,
and unsafe writes are never retried. The runtime client's retry policy
(#14422) retries reads only and gives every write a single, time-bounded
attempt.

## Identity storage and retention

Artifact write identities are stored in the `artifact_write_identities` table
together with a hash of the canonical payload, so conflict detection survives a
server restart. They are scoped by namespace, run, task and operation ID and are
never shared across those boundaries. The API server's garbage collector does
not remove artifact rows today (see `development.md`); the retention of identity
rows follows the same policy and is a follow-up decision rather than part of the
replay contract.
