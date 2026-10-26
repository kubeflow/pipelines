# MLMD database migration implementation plan

This plan groups the seven implementation steps in
[the original migration design](./database-migration-design.md#implementation-plan)
into three PRs. The design defines the migration behavior. A change to that
behavior needs an explicit design decision; a fixture or implementation detail
does not silently amend it.

## Design contract

- Migrate an MLMD-backed KFP installation into its native **MySQL** schema.
  The source is the existing installation's one KFP database and its paired
  MLMD store; this is not a merge of independent MLMD installations.
  Fresh and already-native PostgreSQL installations remain supported; a legacy
  PostgreSQL schema is outside this migration until separately qualified.
- Run the shared engine from a standalone CLI or API-server startup. The CLI
  can copy history ahead of the upgrade; startup performs the final true-up.
  Hold the same MySQL advisory lock in both modes.
- Enter maintenance mode, block submissions and retries across all API replicas,
  and pause recurring triggers while existing runs finish and report. Require zero
  in-flight KFP runs and Argo Workflows before either migrator enters Phase 2/3;
  then terminate every old API replica and wait for its requests and database
  connections to close. The MySQL advisory lock does not fence old replicas.
- Create new artifact tables with manual SQL and migrate the existing
  `tasks` table through the design's copy-and-rename procedure. Compare
  manual SQL with GORM's new-install schema in CI. Back up the KFP database
  before upgrade; restore it with the old server image for rollback.
- Map MLMD IDs to deterministic UUID v5 identities. Insert in dependency
  order, resume by discovering missing target identities, and resolve parent
  task IDs in a second pass. The design specifies binary search and
  `INSERT IGNORE`; the correctness gates below must cover their edge cases
  before either mechanism is enabled.
- Apply the design's mappings: `CANCELED` becomes `FAILED`; split scalar
  metrics into native rows and link source events to the primary metric row.
  For joined tasks, the KFP task row owns nonempty pod, state, and lifecycle
  timestamps; MLMD owns execution type, properties, and relationships. Use
  MLMD values when the corresponding KFP field is absent or unspecified.
  Preserve the KFP run identity and state history. Copy an MLMD cache
  fingerprint only for a `COMPLETE` or `CACHED` execution, as the design
  specifies. Do not populate a fingerprint for other execution states.
- Warn and skip individual transformation failures and missing optional
  properties as described in the design. Missing MLMD configuration, exhausted
  connection retries, disk exhaustion, and schema mismatch fail migration.
  Record skipped source IDs and reasons so completion is auditable.

## PR 1 — Source evidence and migration contract

Capture the released source schema, matching KFP run/task rows, MLMD
executions/artifacts/events, submitted Workflow, pipeline source, and runtime
image provenance from one representative pipeline execution. Keep small
direct-writer fixtures for sparse IDs, long values, metrics, and unusual
states. Hand-author the expected native/API result independently of the
generator and assert the source-to-target joins, parents, types, events, cache
lineage, and loop attributes. Assert KFP-row precedence for its pod, state,
and lifecycle fields, and the design's fingerprint state gate.

Reproduce the current startup failure on a populated legacy MySQL database.
In CI, check the released Workflow's MLMD/API writer routes and source write
timestamps against terminal status. Document the maintenance, run-drain,
old-replica fence, and migration sequence, including observable gate conditions
for both CLI and API migration. PR 3 qualifies the handoff.
The `2.18.0` release fixture supplies
pipeline-executed evidence; the `2.17.3` direct-writer fixtures cover edge
cases.

**Exit gate:** fixtures and commands are reproducible in PR CI; mapping and
source-field preservation are reviewable; the design's upgrade protocol is
specified without enabling migration.

## PR 2 — Shared engine, schema, and CLI

Implement the design's schema preparation, shared MLMD reader and
transforms, deterministic identities, batched writes, parent fixup,
completion checks, MySQL lock, and standalone CLI. Keep source history
untouched. The CLI reports skipped entities, failure, and completion
unambiguously. A completed installation must start later without MLMD
configuration. Copy eligible source fingerprints according to the design's
execution-state gate. Backfill native cache keys for eligible tasks in the
90 days before a fixed migration cutoff only when all native key inputs can be
reconstructed; retain the source fingerprint and report skipped keys and reasons.

Exercise MySQL upgrade from a populated released schema, repeated runs,
interruption and resume, sparse IDs, metric splits, missing parents, source
errors, contention, and rollback from backup. Compare manual schema SQL with
the native model's new-install schema. A legacy PostgreSQL database must fail
before changes; fresh/native PostgreSQL behavior must remain intact.

**Exit gate:** a built CLI migrates and validates a stable disposable source,
recovers from interruption, verifies the bounded native-key backfill, and
refuses unqualified schemas. API startup migration remains disabled; production
CLI use awaits the PR 3 writer fence.

## PR 3 — API startup, deployment, and upgrade qualification

Add startup classification and automatic migration using the same engine.
Expose maintenance responses to users while retaining internal runtime
reporting until all existing runs finish and terminal reports are stored.
Block submissions, retries, and recurring triggers across all API routes. Add
upgrade orchestration that scales the old API Deployment to zero, waits for
old pods and in-flight writes to finish, and requires zero active KFP runs and
Argo Workflows before either migrator enters Phase 2/3. Keep normal traffic
blocked until the completion checklist passes.
Package the CLI and document the maintenance, backup, failure, and restore
procedures.

Run upgrade CI from a pinned MLMD-era release through both CLI-ahead and
API-only paths. Cover concurrent entry points, replica restarts, missing MLMD
settings, skipped records, user-facing maintenance behavior, runtime
reporting, refusal to migrate with an in-flight run, proof that old replicas
cannot restart or write after the fence, cache hits from safely backfilled
native keys, and an MLMD-free restart.

**Exit gate:** the disposable end-to-end upgrade and rollback pass, the final
completion predicate is durable, and no old writer can race the cutover.

## Design feasibility checks before implementation

These are questions raised by review and current code, not replacements for
the original design:

1. **Resume:** binary search for the first missing UUID can miss later holes;
   a metric artifact also expands into multiple native rows, and MLMD events
   lack their own IDs. Prove the design's resume and completion predicates
   under these cases or amend the design before implementing them.
2. **Schema:** the proposed table copy and rename must be assessed against
   [the repository's nonblocking database policy](../../docs/agents/database-migrations.md).
   Do not enable a blocking production migration without resolving that
   conflict.
3. **Drain:** prove submissions and retries are blocked across old replicas,
   recurring triggers are paused, and both migrators refuse Phase 2/3 until no
   KFP run or Argo Workflow remains active and final reports are stored. Do not
   use MLMD execution state alone: the 2.18 fixture has `RUNNING` DAG executions after
   their runs succeed.
4. **Cache:** the design copies `COMPLETE`/`CACHED` source fingerprints.
   Current native key inputs and lookup scope differ. Before the PR 2 backfill,
   amend the design to specify separate storage and lookup for legacy and native
   keys; never treat a legacy hash as a native key or invent missing inputs.
5. **Metrics and skipped records:** verify primary-only event links and
   warning-and-skip behavior preserve the API history promised by the design.
   Record any accepted loss and its recovery path before removing MLMD.
6. **Joined task fields:** assert KFP precedence for nonempty pod, state, and
   lifecycle fields, MLMD ownership of execution properties and links, and
   MLMD fallback when the KFP value is absent or unspecified.
7. **Skipped-state evidence:** the released SQL retains the failed conditional
   run's Argo `Skipped` node, KFP `SKIPPED` row, and MLMD `CANCELED` execution
   in one run. CI checks their names and run context. The KFP row has no
   execution ID, so it cannot override the design's `CANCELED` → `FAILED`
   mapping by the specified join; PR 2 must test that mapping in the engine.

The `2.18.0` pipeline-executed source is captured in
`backend/src/apiserver/migration/mlmd/testdata/release_218/`. The direct-writer
fixtures remain edge cases. The writer-handoff evidence and the remaining
field-mapping decisions above still gate PR 1.
