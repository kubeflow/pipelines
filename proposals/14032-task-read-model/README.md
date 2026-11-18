# KEP-14032: Versioned task read model

- **Status:** Draft for independent discussion; not approved for implementation.
- **Author:** Jeff Spahr (@jeffspahr).
- **Draft date:** 2026-11-18.
- **Tracking:** [#14032](https://github.com/kubeflow/pipelines/issues/14032).
- **Target:** staged, optional work on `main`; release placement and activation
  depend on reviewed contracts and measured evidence, not approval of the API/UI
  separation proposal.
- **Source baseline:** the original design used commit
  [`8a79799a2`](https://github.com/kubeflow/pipelines/commit/8a79799a2f54a678b7535eb4697dfc6812a4bb66),
  with the boundary implementation rechecked on November 11 against
  [`b5e247523`](https://github.com/kubeflow/pipelines/commit/b5e247523201027c2c0b05c3260f53bd0c89008b).
  The November 18 review verified task read authorization, child loading and the current
  proto bindings on [`main` at `7780f12ac`](https://github.com/kubeflow/pipelines/commit/7780f12acfbde9f16169c7fcbfdfb6c3ef5432ea).

## Contents

- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [Risks and mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Existing APIs and compatibility](#existing-apis-and-compatibility)
  - [Summary and change APIs](#proposed-summary-and-change-apis)
  - [Storage and consistency](#storage-and-consistency)
- [Migration Strategy](#migration-strategy)
- [Frontend Considerations](#frontend-considerations)
- [Adjacent Read Models](#adjacent-read-models-with-separate-decisions)
- [KFP Local Considerations](#kfp-local-considerations)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Decisions Required Before Implementation](#decisions-required-before-implementation)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)

## Summary

Add bounded task summaries and a resumable change API so clients can build a
consistent run graph without repeatedly retrieving every task's full metadata.
Use versioned projections and a finite change journal to preserve a snapshot
across pages, detect expired history, and invalidate selected-task details when
the canonical record changes.

This proposal separates optional task read-model work from
[KEP-14668's API/UI separation review](https://github.com/kubeflow/pipelines/pull/15489),
tracked by [#14668](https://github.com/kubeflow/pipelines/issues/14668). Removing UI
storage credentials and Kubernetes access, including the first artifact-content
migration in [#14031](https://github.com/kubeflow/pipelines/issues/14031), does not
wait for snapshots, grouping, search, or lineage improvements. The operation
names, schemas, limits, and storage approach below are proposals for review.

## Motivation

Full task metadata includes information a run graph does not need. Repeated full
reads increase response size and backend/browser work as runs grow, while paging
over mutable records alone cannot guarantee that a graph represents one revision.
An incremental client also needs explicit deletion, retry, and detail-invalidation
semantics; a timestamp or a generic polling interval is insufficient.

### Goals

1. Bound summary responses and preserve membership and values across snapshot pages.
2. Publish replayable changes, including deletions and accepted run-attempt resets.
3. Keep full task I/O out of graph summaries while detecting stale task details.
4. Make history finite, authorization per-request, and recovery from expiration explicit.
5. Introduce storage and capture without blocking production reads or writes.
6. Measure gains and database costs before enabling frontend adoption.

### Non-Goals

- A prerequisite for UI/API separation or for all metadata correctness fixes.
- Historical full-task/I/O reads, an unbounded audit log, or indefinite cursor validity.
- A watch/SSE transport in the initial API.
- Making existing bulk task updates atomic.
- Approval of grouped artifacts, cross-run search, or a lineage-subgraph API.
  Those retain separate owners and decisions below.

## Proposal

Introduce an additive, capability-gated summary/change API alongside current task
reads. Maintain versioned summaries and a finite journal in the same transactions
as canonical mutations, bootstrap existing runs without blocking their writers,
then adopt the APIs in shared frontend queries after qualification. Keep accepted
attempt identity and standalone correctness fixes independently deliverable.

### Risks and mitigations

A busy run can serialize writers on its revision lock. Measure write latency and lock waits
during capture and bootstrap before enabling the capability. A missed writer or a restored
database can create a gap in history. Retire that capture epoch and rebuild the baseline
before serving new pages. A stale browser response can expose data after a scope change.
Partition caches by authorization scope and discard pending responses when that scope ends.

## Design Details

### Existing APIs and compatibility

The canonical `backend/api/v2/run.proto` provides:

| Existing operation | HTTP binding / purpose |
| --- | --- |
| `GetTask` | `GET /apis/v2/runs/{run_id}/tasks/{task_id}` |
| `ListTasks` | `GET /apis/v2/runs/{run_id}/tasks` |
| `UpdateTask` | `PATCH /apis/v2/runs/{run_id}/tasks/{task_id}` |
| `UpdateTasksBulk` | `POST /apis/v2/runs/{run_id}/tasks:batchUpdate` |
| `FindCachedTask` | `POST /apis/v2/tasks:findCached`; namespace/fingerprint cache lookup, not task search |

`backend/api/v2/artifact.proto` provides `ListArtifacts`, `GetArtifact`, and
`ListArtifactTasks` (`GET /apis/v2/artifact_tasks`). The relationship request
accepts one `IOType`; it has no direction predicate.

Keep these operations supported. Introduce additive protobuf fields and APIs,
regenerate Go/Python/TypeScript clients, and preserve the documented v2beta1
compatibility routes and generated aliases. New definitions belong only in
`backend/api/v2`; the frozen legacy descriptor remains a compatibility fixture.
Do not replace current task reads until server capability detection and
mixed-version tests exist.

### Proposed summary and change APIs

Start with two unary RPCs and JSON HTTP bindings. Defer watch/SSE transport until
resume semantics are proven.

| Proposed operation | HTTP binding | Response |
| --- | --- | --- |
| `ListTaskSummaries` | `GET /apis/v2/runs/{run_id}/task_summaries` | `tasks`, `run_summary`, `snapshot_revision`, `next_page_token`, final-page `changes_cursor` |
| `ListTaskChanges` | `GET /apis/v2/runs/{run_id}/task_changes?cursor=…` | ordered `changes`, `through_revision`, `next_page_token`, final-page `changes_cursor` |

A task summary contains task/run IDs, name/display name, parent/scope/iteration,
task type/state, timestamps, a bounded diagnostic summary, current executor
references, I/O-presence markers, `task_revision`, and execution-attempt identity.
It excludes parameter values, artifact metadata, pipeline specifications, and
full state history. At revision R, `run_summary` supplies the run ID, execution
state, accepted attempt ID when known, and task-finalization state. Changes to
these fields produce a run-summary record even when no task changes.

`GetTask` gains an output-only `task_revision` covering its full current
response. Today the [task store hydrates artifact relationships](https://github.com/kubeflow/pipelines/blob/c1ff1d2658d497b189509e67c32658bc2c3cfa89/backend/src/apiserver/storage/task_store.go#L926-L954)
and the [handler separately loads child tasks](https://github.com/kubeflow/pipelines/blob/c1ff1d2658d497b189509e67c32658bc2c3cfa89/backend/src/apiserver/server/run_server.go#L676-L696).
Read the task, hydrated I/O, child list and revision in one short consistent
transaction for the revisioned response. A selected detail can be newer than the
graph. Label it as current and refresh the graph through its change API.

#### Snapshot pagination

The first summary request captures committed revision R. Pages use deterministic
task-ID keyset order at R; mutations during pagination cannot change membership
or values in that snapshot. Initially support whole-run summaries without
mutable-state filters, avoiding ambiguous delta filter-exit semantics.

Proposed initial limits are 500 rows and 1 MiB per response, with 15-minute cursor
validity; benchmarks must validate defaults before approval. Individual summary
fields are bounded, and diagnostic truncation is explicit, so a single row cannot
cause a zero-progress page loop. Row/byte limits apply to both summary and change
pages. Bind opaque, integrity-protected tokens to the run, authenticated caller
and namespace scope, capture epoch, projection version, ordering, revision
bounds, expiry and last key. Every request, including continuations, uses the
existing `GetTask`/`ListTasks` run access check. In multi-user mode this requires
`pipelines.kubeflow.org` `runs.get` for the resolved run and namespace. Preserve
the existing single-user behavior. Check access before disclosing readiness,
token expiry or data. Send `Cache-Control: private, no-store` on caller-specific
HTTP pages and errors, including through compatibility adapters. Keep the
client's working view inside its authorized scope.

#### Change pagination and replay

The first change request captures a fixed upper watermark U and returns changes
after R through U. Later pages remain at U. Changes contain complete replacement
summaries or deletion tombstones, plus run-summary/attempt-transition records.
Clients stage pages in `(revision, ordinal)` order and ignore replayed records.
Publish the completed view and its durable cursor together after the final page.
A page break can split one transaction's records. Keep the last complete view
visible while collecting its replacement. An unchanged run
returns an empty page and fresh cursor. A lost response can be replayed without
losing or duplicating visible state. Work scales with changed data, subject to
pagination; arbitrarily large changes do not fit into one response.

### Storage and consistency

One feasible implementation is a bounded, transactionally maintained
summary-version table plus change journal. Every task mutation increments its
task record revision. Mutations that affect a summary update its projection and
journal in the same database transaction as the canonical state. These include
task creation, pod publication, relevant relationship changes, deletion and retry
reset. A change to hydrated I/O advances the affected task revision even if the
other summary fields stay equal. Inserting, removing, reparenting or renaming a
child can change a parent's returned `child_tasks`. Advance each affected
parent's revision and emit its replacement summary in the same transaction.
The client can then invalidate a selected parent's details.

Allocate a monotonically increasing per-run revision under the run row lock,
using one documented run-before-task lock order. Existing bulk updates remain
individually committed; this proposal does not silently make them atomic.

Summary versions have validity intervals, allowing reads at R without keeping a
database transaction open across HTTP requests. Retain the last version visible
at the compaction floor plus later versions; index snapshot lookup by
run/task/revision and journal reads by run/revision/ordinal. Each page validates
its retention floor and reads under one short consistent transaction. Run status
and summaries must come from the same revision boundary. Benchmark per-run
revision allocation contention and MySQL/PostgreSQL query plans before accepting
this design.

#### Retention, expiration, and deletion

History has finite age and byte/row quotas. Compaction can invalidate old cursors
explicitly; it must never silently skip changes. Return `FAILED_PRECONDITION`
with `google.rpc.ErrorInfo.reason=SNAPSHOT_EXPIRED` or `CHANGE_CURSOR_EXPIRED`,
requiring a fresh snapshot for the still-authorized caller. Invalid tokens return
`INVALID_ARGUMENT`; quota exhaustion returns `RESOURCE_EXHAUSTED`.
The final HTTP error mapping must preserve these machine-readable reasons.

Cursor expiry and retained-history bounds are distinct: a token is usable only
while both remain valid. Publish the supported replay window and whether storage
pressure may shorten it before approving the defaults. Tombstones persist through
that window, and compaction must advance the applicable retention floor before
removing records still needed by an accepted cursor. Deleting a run makes
subsequent snapshot/change requests follow normal authorized not-found behavior;
these unary APIs do not create a stream that needs termination.

#### Accepted attempts and finalization

Use the shared opaque `run_attempt_id` and `task_attempt_id` contract owned by
[#15516](https://github.com/kubeflow/pipelines/issues/15516), coordinated with
[#14322](https://github.com/kubeflow/pipelines/issues/14322). A manual run retry
establishes a new accepted execution epoch and a revisioned reset before accepting
its task writes. Automatic task retries create a new `task_attempt_id` within the
existing `run_attempt_id`; they do not reset the whole run snapshot. Earlier
run-epoch writers are fenced transactionally. An otherwise valid retained cursor
that crosses a run reset receives an attempt-reset record and restarts its
snapshot rather than merging generations; an expired cursor follows the normal
expiration path.

Existing `RetryGeneration` is a claim/fencing token: failed claims can increment
it without executing a new attempt. Do not expose it as a human attempt number.
Terminal task-write protection can ship separately under
[#15008](https://github.com/kubeflow/pipelines/pull/15008); snapshot storage must
later preserve the same invariant.

Before promising immutable terminal snapshots, distinguish execution terminality
from task finalization. Ordinary stale runtime writes remain rejected. Either
reconcile and commit task outcomes before sealing the final run snapshot, or
expose an explicit finalization-pending state until bounded reconciliation fenced
by current attempt and pod UID completes. The chosen contract must cover
[#14989](https://github.com/kubeflow/pipelines/issues/14989)'s missing-final-report
case and #15008's guard together; an immutable terminal snapshot cannot be
advertised while permitted reconciliation can still change it. A focused guard
or reconciliation fix can ship with its tested ordering/exception contract
before snapshots; this does not allow arbitrary post-terminal rewrites.

The shared identity work adds authoritative namespace and attempt fields to
`TaskPod`, preserving `name`, `uid`, and the existing `type` field
(`UNSPECIFIED`, `DRIVER`, `EXECUTOR`). It owns backend verification, early regular
executor publication and its failure/retry policy, and explicit unavailable
historical identities. Preserve the importer's existing early publication.
When snapshots are active, publication advances `task_revision` even if state
remains RUNNING. Do not duplicate identity ownership or require snapshots for
backend log-source discovery; imported records must not cause clients to guess
the UI namespace.

## Migration Strategy

Install transactional capture on every writer first. Bootstrap existing runs in
bounded, restartable batches with conditional projection upserts and journal
catch-up. Use only short watermark/finalization barriers, never a whole-run scan
while holding the run lock. Captured deletion/reset tombstones must defeat stale
bootstrap upserts. Retain capture history throughout bootstrap or explicitly
restart if compaction overtakes it. Publish a completed baseline watermark and
serve only revisions at or after it; prove that baseline gap-free before
advertising snapshot capability.

Capture has a persistent epoch identifying one uninterrupted history. It is
independent of the accepted execution attempt. Disabling public snapshot reads
can preserve the epoch if every writer continues capturing. Before admitting a
writer that can bypass capture, mark affected baselines unavailable and retire
that epoch. If continuity is uncertain after rollback or restore, start a fresh
epoch, verify all writers and rebuild the affected baselines before serving.
After recovery old tokens return the corresponding snapshot/change expiry reason
even if numeric revisions overlap. Define legacy-runtime attempt handling with
#15516/#14322 instead of assuming an omitted execution epoch is safe.
New history in [#14981](https://github.com/kubeflow/pipelines/issues/14981) is not a
blanket prerequisite for this contract or basic backend log access.

Use an independent, operator-controlled capability, provisionally
`task_snapshots.v1`. The optional [namespace capability registry](https://github.com/kubeflow/pipelines/issues/15519)
advertises support, version and limits for the caller and namespace. The first
authorized summary read determines readiness for that run. An incomplete baseline
returns `FAILED_PRECONDITION` with `ErrorInfo.reason=SNAPSHOT_NOT_READY` and a
bounded retry delay. #14032 owns this response and can ship independently of the
registry. Preserve version negotiation for older servers without the registry.
Fall back to bounded polling on the existing task API only for a known disabled
or unsupported feature, or explicit per-run unavailability. An ordinary run
lookup failure follows the existing error path. Authorization failures clear the
affected view. Review the mixed-version window before implementation.
Disabling the optional capability may restore current task reads; it must not
restore infrastructure credentials removed by API/UI separation.

Follow [nonblocking database migration guidance](../../docs/agents/database-migrations.md).
Use additive tables/fields and bounded, restartable backfills. Preserve stored
values and support old readers/writers until capture/epoch enforcement activates.
Verify explicit supported DDL algorithms, bounded lock waits, replica coordination,
and recovery on populated MySQL and PostgreSQL. No blocking table-copy or
whole-table startup backfill is an acceptable fallback. Qualify the actual
populated 2.18 upgrade baseline and publish any irreversible activation conditions;
this proposal does not require a 2.17 backport or promise unrestricted rollback.

## Frontend Considerations

| Work | Tracking | Delivery boundary |
| --- | --- | --- |
| Snapshot APIs, capture, bootstrap, retention | [#14032](https://github.com/kubeflow/pipelines/issues/14032) | Activation consumes reviewed identity/finalization and all-writer capture. Its terminal guard remains independently deliverable. |
| Shared frontend queries | [#14040](https://github.com/kubeflow/pipelines/issues/14040) | Can close against current APIs before snapshots exist. Sharing and cancellation are useful independently. |
| Snapshot adoption | [#14028](https://github.com/kubeflow/pipelines/issues/14028) | The snapshot slice consumes #14032 and shared queries. Other performance/hardening slices remain independently shippable. |
| Accepted attempts and executor identity | [#15516](https://github.com/kubeflow/pipelines/issues/15516) | Shared correctness contract; no dependency on summary storage. |
| Lifecycle/finalization | [#14322](https://github.com/kubeflow/pipelines/issues/14322), [#14989](https://github.com/kubeflow/pipelines/issues/14989), [#15008](https://github.com/kubeflow/pipelines/pull/15008) | Coordinate the relevant snapshot invariant; do not block all lifecycle fixes on the read model. |

Scope client caches to principal, namespace, run and capture epoch. Distinguish
snapshot revision, task revision and attempt. A same-scope cursor expiry or
transient refresh failure may keep the last authorized view visible with a
freshness indicator. Authentication or authorization failure, an identity or
namespace switch, or confirmed run deletion clears affected graph, detail and
cursor caches. Cancel outstanding requests and discard late responses from the
old scope. Preserve user selection during same-scope refresh, clear stale
errors after recovery and avoid duplicate effects on remount.

## Adjacent Read Models with Separate Decisions

The following designs retain their existing owners. They are not requirements
to close #14032 or complete UI/API separation.

### Relationship selectors and conditional lineage

[#15522](https://github.com/kubeflow/pipelines/issues/15522) owns extending
`ListArtifactTasksRequest` with `repeated IOType types` and optional
`PRODUCER|CONSUMER` direction. Preserve old `type`; because it is a non-optional
proto3 enum, only a non-UNSPECIFIED value counts as a selected legacy filter.
Reject combinations of active legacy `type`, nonempty `types`, and explicit
`direction`; omitted/default legacy enum values do not conflict with new
selectors. Retain artifact/task/run IDs and key filters, namespace authorization,
stable pagination, and explicit unknown-type behavior. This removes the
four-request producing-relationship lookup without requiring snapshot storage.

A purpose-built lineage-subgraph API remains separately designed. The existing
bounded neighborhood UI remains valid. Before closing #15522, explicitly accept,
defer, or transfer its retained lineage requirements; a selector-only delivery
must not silently erase them.

### Grouped artifacts and task discovery

[#14041](https://github.com/kubeflow/pipelines/issues/14041) owns proposed
`ListArtifactGroups` and group-member reads after choosing pipeline/workspace
equivalents and multi-producer/imported/orphan semantics.
[#14301](https://github.com/kubeflow/pipelines/issues/14301) owns namespace-scoped
`SearchTasks` and, separately, an authorized legacy-execution mapping lookup.
Only legacy-link resolution consumes
[#14029](https://github.com/kubeflow/pipelines/issues/14029)'s execution mapping
scoped to the source installation; native search does not wait for historical import.
Neither feature should scan runs/artifacts in the browser. Final routes,
selectors, authorization, bounds, and representations require their own API
review and retained tests; these names do not establish approved schemas.

## KFP Local Considerations

The summary contract describes domain tasks and accepted attempts, not mandatory
Kubernetes pods. A runtime without pods must represent that absence explicitly.
Any future local backend advertising the capability must satisfy the same
revision, pagination, and capture semantics, with storage-engine-specific tests.
This proposal does not require the current local experience to adopt new task
storage or block the separate
[KFP Local proposal](https://github.com/kubeflow/pipelines/tree/main/proposals/15214-kfp-local).

## Test Plan

- [x] The author understands component owners may require additional tests before
  implementation is accepted.

This document preserves proposed implementation evidence, not claims that the
tests have run or that the feature is delivered. Current aggregate coverage has
not been measured.

### Unit and contract tests

- Verify projection completeness, full-I/O-only revision invalidation, stable
  ordering, explicit diagnostic truncation, and progress at row/byte bounds.
- Encode/decode tokens; reject tampering, mismatched run/caller scope, unknown
  projection versions, invalid selectors, and unsupported version combinations.
  Preserve machine-readable expiration reasons through gRPC/HTTP/client adapters.
- Split a revision across pages, drop and replay the last response, then restart
  the client. Verify both the visible view and its durable cursor advance together.
- Verify frontend query sharing, cancellation, refetch recovery, preserved user
  state, keyboard/focus behavior, and meaningful unavailable states.

### Integration and migration tests

- Race snapshot/change pages with writes, full I/O updates, deletion, compaction,
  and retry. Cover summary expiry, change expiry, authorization revocation,
  retention pressure, run deletion, and executor publication during a component.
- Add, remove or move a child and change hydrated artifact I/O during a detail
  read. Check every affected parent or task receives a changed revision and each
  returned detail matches its revision without mixing old and new fields.
- Test manual/automatic retries independently, failed retry claims, stale-epoch
  writes, pod-name reuse with different UIDs, and explicit unavailable historical
  identity. Cover the chosen terminal sealing/reconciliation ordering.
- Race bootstrap with creation/deletion/reset, interrupt and resume batches,
  overtake capture retention, and prove the published baseline is gap-free.
  Disable reads with capture active, then introduce an uncaptured write during a
  rollback. Verify a rebuilt baseline includes it and rejects the old cursor
  even when revision numbers overlap.
- Exercise two tenants, authorized read-only and write-capable principals under
  the deployed RBAC, and unauthorized callers. Cursor possession and forged
  identity/namespace fields must not grant access. Revoke access or switch
  namespace between pages and during a detail fetch. Verify the old view clears
  and a delayed success cannot repopulate the cache.
- Run populated MySQL/PostgreSQL upgrades with concurrent reads/writes, competing
  replicas, bounded lock contention, interruption and rollback. Capture actual
  generated DDL and latency during migration; inspect query plans and verify
  stored-value preservation. SQLite/empty-schema results are insufficient.
- Test capability detection, per-run bootstrap readiness, old/new UI/API/client
  combinations, bounded-polling fallback, and disabling optional adoption.

### End-to-end and scale tests

Use production UI and SDK clients against the same authorized fixtures. Measure
initial loads and unchanged/changing refreshes at 100/1,000/2,000/10,000 tasks and
2/5/10-run comparisons: request counts, response bytes, refresh latency, rows
scanned, per-run lock contention, browser render time, cancellation, and bounded
memory. Set numerical release budgets from these results before graduating the
capability. Preserve existing full-task details and supported comparison behavior.

Implementation PRs retain test names and CI evidence. Regeneration follows the
[canonical v2 API guide](../../docs/agents/generation.md), including
`make -C backend/api API_VERSION=v2 generate`, SDK-owned Python client generation,
and `frontend`'s `npm run apis:all`. New HTTP representations need matching route
and error-contract tests.

## Graduation Criteria

1. **Draft to implementable:** the repository review pools named below approve
   the relevant schemas, attempt/finalization invariants, authorization, bounds,
   retention, compatibility, and a feasible nonblocking capture/bootstrap design.
2. **Backend capability enabled:** consistency, isolation, migration, recovery,
   and measured scale tests pass; published readiness and limits match reality.
3. **Frontend adoption complete:** #14028's snapshot slice meets agreed budgets
   and recovery behavior after shared-query adoption, without losing current
   task-detail/comparison behavior.

Adjacent relationships/grouping/search graduate through their own tickets. None
of these criteria redefine completion of the mandatory #14668 separation work.

## Decisions Required Before Implementation

Review routing follows the versioned repository `OWNERS` files below, with their
approver handles recorded for this draft. These are review pools, not newly
assigned implementers or a requirement that every listed person approve. The
applicable `OWNERS` files govern approval when implementation PRs are opened.

| Scope | Repository owner path and approvers |
| --- | --- |
| Backend API, runtime and database | [backend/OWNERS](../../backend/OWNERS): @chensun, @hbelmiro, @mprahl, @nsingla, @alyssacgoins. This covers canonical `backend/api/v2`; root `api/OWNERS` covers a different directory. |
| Frontend queries, cache and adoption | [frontend/OWNERS](../../frontend/OWNERS): @chensun, @HumairAK, @jeffspahr, @manaswinidas, @alyssacgoins. |
| Python SDK/client compatibility | [sdk/OWNERS](../../sdk/OWNERS): @chensun, @mprahl, @alyssacgoins. |
| Deployment, RBAC and capability configuration | [manifests/OWNERS](../../manifests/OWNERS): @chensun, @HumairAK, @juliusvonkohout. |
| Proposal approval and release coordination | [OWNERS](../../OWNERS): @chensun, @droctothorpe, @HumairAK, @jeffspahr, @mprahl, @zazulam. |

| Decision | Proposed default | Review routing |
| --- | --- | --- |
| Routes, RPCs, field types and clients | Canonical v2 additive operations above; reserve field numbers and approve HTTP schemas/error mappings and compatibility fixtures. | [backend/OWNERS](../../backend/OWNERS), with [sdk/OWNERS](../../sdk/OWNERS) and [frontend/OWNERS](../../frontend/OWNERS) for generated clients. |
| Summary/detail merge behavior | Current `GetTask` plus output-only revision; approve stale/newer detail presentation without implying historical I/O. | [backend/OWNERS](../../backend/OWNERS), [frontend/OWNERS](../../frontend/OWNERS). |
| Accepted attempt identity and legacy writers | Consume #15516; coordinate retry reset, legacy epoch handling, and legal terminal/finalization writes. | [backend/OWNERS](../../backend/OWNERS). |
| Storage, write amplification and capture | Version table/journal with per-run revision allocation; validate contention, lock order and every mutation path. | [backend/OWNERS](../../backend/OWNERS). |
| Cursor, page and retention bounds | Provisional 500 rows, 1 MiB and 15 minutes; approve lifetime versus compaction/quota guarantees and retry behavior from tests. | [backend/OWNERS](../../backend/OWNERS), [frontend/OWNERS](../../frontend/OWNERS). |
| Bootstrap and activation | Qualify conditional backfill, capture-epoch invalidation and the per-run `SNAPSHOT_NOT_READY` response with old/new clients and writers. | [backend/OWNERS](../../backend/OWNERS), [frontend/OWNERS](../../frontend/OWNERS), [manifests/OWNERS](../../manifests/OWNERS) for activation controls. |
| Authorization and scope | Recheck existing domain read authorization on every page; approve cursor scope, revocation behavior and deployed RBAC coverage. | [backend/OWNERS](../../backend/OWNERS), [manifests/OWNERS](../../manifests/OWNERS). |
| Compatibility and rollback | Set the supported mixed-version window and activation conditions; disable optional reads without restoring retired UI infrastructure access. | [OWNERS](../../OWNERS), [backend/OWNERS](../../backend/OWNERS), [manifests/OWNERS](../../manifests/OWNERS), with SDK/frontend compatibility review above. |
| Separate relationship/discovery designs | #15522/#14041/#14301 retain their own schemas and acceptance; adoption of this proposal does not approve those APIs. | [backend/OWNERS](../../backend/OWNERS), [frontend/OWNERS](../../frontend/OWNERS), through each separate issue/design review. |

Acceptance as a draft does not declare the feature implementable or relax any
existing issue's compatibility, authorization, retention, or migration criteria.

## Drawbacks

Versioned summaries and capture add storage, cleanup, write amplification, and
coordination cost. A per-run revision lock may contend on large, active runs;
retain the design only if measured costs are acceptable. Cursor recovery and
mixed-version activation add client and operational complexity.

## Alternatives

Current task APIs with bounded polling and independently improved query sharing
remain a valid baseline. A current-only materialized summary could reduce bytes
but would not alone provide stable multi-page snapshots or replay. A watch/SSE
API might reduce polling, but adds transport/lifecycle questions and must still
solve finite retention and resumption; it is deferred. Holding a database
transaction across client pages or retaining an unbounded journal is not the
proposed solution.

## Implementation History

- 2026-11-10: Initial metadata API design drafted within KEP-14668.
- 2026-11-11: Reviewed capture, retry, dependency ordering and terminal-finalization
  semantics alongside implementation ownership.
- 2026-11-18: Split into this independently reviewed proposal under existing
  #14032; preserved optional relationship/discovery ownership and implementation
  test/decision requirements. No implementation or release graduation is claimed.
