# Runtime write replay safety

The native runtime (driver and launcher) writes to the API server over gRPC.
A write can commit on the server while its response is lost, so a client that
sends the same write again must not corrupt state. This page classifies the
runtime write operations listed below by what a repeated delivery does. It
provides guidance for deciding whether a write may be retried automatically.

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
| `CreateArtifactTask`, `CreateArtifactTasksBulk` | Unsafe | Each delivery generates new artifact-task relationship IDs. Bulk creation is transactional, but replaying a committed request creates new relationship rows. |
| `CreateTask` | Conditionally safe | Uses a logical identity consisting of run UUID, normalized parent task UUID, scope path, name, type and optional iteration index. If a matching task exists, it is returned unchanged. If a concurrent insert fails and a subsequent logical-key lookup finds the winning task, that task is returned. A different payload under the same logical identity is not reported as a conflict. When required identity fields are missing, no logical identity is available and each delivery can create a new task. |
| `UpdateTask` | Conditionally safe | Updates are performed in a transaction with a row-level lock. Input and output parameters merge by key against the locked current state; provided scalar and metadata fields overwrite their corresponding values. State history is appended only when the state changes. There is no revision fencing, so a stale update can overwrite newer fields supplied in the request. The server also rejects task updates for runs that no longer allow runtime writes. |
| `UpdateTasksBulk` | Conditionally safe, not atomic | See below. |

## `UpdateTasksBulk` partial completion

`UpdateTasksBulk` performs request-wide validation of task existence,
run scope, and authorization before applying updates. It then processes
tasks one at a time in sorted task-ID order, with each task update in
its own transaction.

Some validation, including parent-task ownership validation, occurs
inside the apply loop rather than during request-wide validation.
Therefore:

- A failure during request-wide validation changes nothing.
- A failure during the apply loop can leave earlier task updates
  committed while later updates remain unapplied. The RPC returns
  an error without a response body.
- Replaying the same update payload for tasks that already committed
  does not append duplicate state history when the state is unchanged.
  A corrected request can then apply the remaining updates.
- This does not make arbitrary stale or changed task updates replay-safe.
  `UpdateTask` has no revision fencing, so an older update can overwrite
  newer fields.

`TestUpdateTasksBulk_FailureAfterFirstUpdateCanBeReplayed` and
`TestUpdateTasksBulk_ValidationFailureLeavesEveryTaskUnchanged`
pin these specific behaviors.

## Eligibility for automatic retries

Only operations classified as replay-safe, with the prerequisite satisfied, may
be retried automatically by the runtime client. Conditionally safe writes must
not be retried until their prerequisite or fencing is guaranteed by the caller,
and unsafe writes are never retried. The runtime client's retry policy
(#14422) retries reads only and gives every write a single, time-bounded
attempt.

## Identity storage and retention

Artifact write identities are stored in the `artifact_write_identities` table
together with a hash of the canonical payload, so conflict detection survives
a server restart. They are scoped by namespace, run, task and operation ID.
The API server's garbage collector does not currently remove artifact rows
(see `development.md`). The retention policy for artifact-write identity rows
is not established as part of the replay contract and remains a separate
lifecycle decision.
