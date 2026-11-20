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
| `CreateArtifact` with `operation_id` | Replay-safe | Same operation and same payload returns the stored artifact and link. The same operation with a different payload fails with `INVALID_ARGUMENT`. A different `operation_id` is a distinct write. After a run retry has reset the task, a replay fails with `FAILED_PRECONDITION` (see below). `operation_id` is limited to 191 bytes. |
| `CreateArtifact` without `operation_id` | Unsafe | Each delivery allocates a new artifact ID, so a replay after a lost response creates a second artifact and link. |
| `CreateArtifact` with `reuse_if_exists` (importer) | Conditionally safe | Creates the artifact only if none in the same namespace matches the stable reuse identity (type, URI, name, description and metadata); concurrent callers share one row and each still gets its own link. This cannot be combined with `operation_id`; the server rejects the combination. |
| `CreateArtifactsBulk` with `operation_id` on every artifact | Replay-safe | The batch is all-or-nothing. Each artifact is deduplicated by its own operation identity, and any item that would fail as a single `CreateArtifact` fails the whole batch with the same status. `reuse_if_exists` is rejected in bulk requests. |
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

## Replay after a run retry

Retrying a run resets every task that did not succeed, was not cached and was
not skipped. The reset deletes the task's artifact-task links and reuses the
task ID, but keeps the artifact and its write identity. A delayed delivery of
an `operation_id` written by the reset attempt therefore cannot return its
original result without attaching old-attempt output to the new attempt.

The server resolves such a replay as follows:

| Situation | Result |
| --- | --- |
| Same `operation_id`, different payload | `INVALID_ARGUMENT`, checked first, also after a retry |
| Same `operation_id` and payload, link still present (including tasks a retry preserved) | The original artifact |
| Same `operation_id` and payload, link removed by a run retry | `FAILED_PRECONDITION`; no link is created and no row changes |
| Same `operation_id` and payload, link missing on a run that was never retried | `INTERNAL` (inconsistent state) |

`FAILED_PRECONDITION` is permanent for that write: the caller must discard it
and must not retry it. A new attempt uses new `operation_id` values. The server
tells a retry reset apart from an inconsistency by the run's `RetryGeneration`,
read under a row lock; the generation is evidence only and is not part of the
write identity.

Replay resolution locks the run row before the artifact and link rows, and
retry cleanup locks the run row before deleting links. Keeping this order
prevents index-order deadlocks between replays and cleanup on MySQL.

`TestCreateArtifactWithTask_ReplayAfterRetryCleanupFailsPrecondition`,
`TestRetryRun_ReplayedArtifactWriteAfterCleanupIsStaleAttempt`,
`TestArtifactServer_ReplayAfterRetryCleanupFailsPrecondition` and the
MySQL/PostgreSQL tests
`TestCreateArtifactWithTask_ReplayAfterRetryCleanupProductionDatabases` and
`TestCreateArtifactWithTask_ReplayRacingRetryCleanupProductionDatabases` pin
this behavior.

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

- **Schema and upgrade:** the table is new and is created by AutoMigrate at API
  server startup. No existing table is altered and nothing is backfilled.
- **Rollback:** an older API server ignores the table. Writes it accepts are
  not deduplicated, which is the behavior before this table existed.
- **Retention:** the server never deletes identity rows, including when a run
  retry resets a task or when a run is deleted. A retry must keep them, because
  deleting an identity would let a delayed replay create output for the new
  attempt. The API server's garbage collector does not currently remove
  artifact rows either (see `development.md`). A retention policy for identity
  rows is a separate lifecycle decision.
