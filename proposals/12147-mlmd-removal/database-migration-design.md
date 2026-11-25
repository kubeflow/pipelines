# MLMD to KFP Native Database Migration

- **Related Proposal:** [MLMD removal proposal](./README.md)

---

## Table of Contents

1. [Summary](#summary)
2. [Motivation](#motivation)
3. [Goals](#goals)
4. [Proposal](#proposal)
   - [User Stories](#user-stories)
   - [Design Details](#design-details)
     - [Architecture](#architecture)
     - [Migration Phases](#migration-phases)
     - [New Database Tables](#new-database-tables)
     - [Idempotency](#idempotency)
     - [Migration completion checklist](#migration-completion-checklist)
     - [MLMD connection configuration](#mlmd-connection-configuration)
     - [MLMD Entity Transformations](#mlmd-entity-transformations)
     - [Concurrent CLI and API migration](#concurrent-cli-and-api-migration)
     - [Execution Modes](#execution-modes)
     - [Maintenance Mode](#maintenance-mode)
     - [Handling In-Flight Runs](#handling-in-flight-runs)
     - [Backup and Recovery](#backup-and-recovery)
     - [Error Handling](#error-handling)
5. [Risks and Mitigations](#risks-and-mitigations)
6. [Test Plan](#test-plan)
7. [Implementation Plan](#implementation-plan)

---

## Summary

This KEP proposes a design for database migration to transfer historical pipeline metadata — executions, artifacts, and events — from ML Metadata (MLMD) into KFP's native MySQL-backed schema. This migration is a prerequisite for the broader MLMD removal effort ([#12147](https://github.com/kubeflow/pipelines/issues/12147)). Without it, upgrading from a MLMD-backed KFP version to a future MLMD-free version would permanently lose run history, task records, and artifact lineage.

---

## Motivation

MLMD is being replaced with native storage in the KFP API server, as part of the broader [MLMD removal initiative](./README.md) ([#12147](https://github.com/kubeflow/pipelines/issues/12147)). Why MLMD is going away and what the new runtime path looks like is covered in the [parent proposal](./README.md) and [design details](./design-details.md). This KEP is concerned specifically with what happens to the historical data already sitting in MLMD.

Every MLMD-backed KFP installation has built up a history of task executions, artifacts, lineage events, and cache fingerprints inside MLMD's database. Once MLMD is gone, that history needs to be moved into KFP's own MySQL schema before the removal can happen.

This KEP specifies the migration subsystem that copies historic MLMD records into KFP's `tasks`, `artifacts`, and `artifact_tasks` tables, so upgrades don't come at the cost of run history, artifact lineage, or cache hits from past pipelines. Concretely, the engine needs to:

- Migrate all relevant historical metadata from MLMD into KFP's native tables.
- Run migration under a planned maintenance window so the API does not serve normal traffic during the migration.
- Be idempotent, so it's safe to rerun without creating duplicates or corrupting live data.

---

## Goals

1. **Data preservation** — All executions, artifacts, and events recorded in MLMD are migrated into the KFP native schema with full fidelity, as well as supported KFP execution, artifact, and lineage metadata. Account for unsupported records using the [skip policy](#error-handling).
2. **Safe upgrade** — On startup, always migrate when the schema is out of date or a migration is in progress. During migration the API runs in maintenance mode. Normal traffic resumes only after migration completes successfully. Startup fails when migration is required but cannot run (for example, missing MLMD connection info).
3. **Idempotency** — The migration can be run multiple times. Repeated runs produce no duplicates and leave the database in a consistent state.
4. **Resumability** — A migration interrupted by a pod crash, SIGTERM, or node failure can resume by  filling gaps in bounded batches. Already-migrated rows are not replayed as new work.
5. **Dual execution modes** — Operators can run migration automatically on API server startup or manually via a standalone CLI tool.

---

## Proposal

### User Stories

**Story 1: Admin upgrading an existing KFP deployment**

> As a KFP platform admin, I want to upgrade from a MLMD-backed version of KFP to a MLMD-free version without losing any historical pipeline run data or artifact lineage, so that my team can continue to audit and debug past runs after the upgrade.

**Story 2: Admin runs the migration utility before the upgraded API starts**

> As a KFP admin, I want to run the standalone migration utility after existing runs drain and old writers stop, so the upgraded API can verify completion at startup.

**Story 3: Recovery from a failed migration**

> As a KFP admin, if the migration fails partway through (e.g., due to a pod restart), I want to be able to re-trigger it and have it resume where it left off, rather than restarting from scratch.

---

### Design Details

#### Architecture

The migration system has three parts that share one migration engine:

```
┌──────────────────────────────────────┐
│          Migration Engine            │
│  (orchestration, transforms,         │
│   target-based resume, validation)   │
└──────────────────┬───────────────────┘
                   │
     ┌─────────────┴─────────────┐
     │                           │
┌────▼────────────┐    ┌─────────▼──────────┐
│  Standalone CLI  │    │  API Server         │
│  (manual run)    │    │  (migrate at startup│
│                  │    │   under maintenance │
│                  │    │   mode)             │
└─────────────────┘    └────────────────────┘
```

Both entry points use the same migration logic. Only how migration is triggered and configured differs.

#### Migration Phases

**Phase 1 — Pre-migration validation**

- Classify the KFP database before migration writes or startup schema reconciliation. Legacy MLMD migration supports MySQL only. Reject legacy PostgreSQL with an actionable error before changing its schema or data. Fresh and already-native PostgreSQL follow normal startup without MLMD.
- Verify MLMD gRPC connectivity.
- Use the existing installation's MLMD store for the initial migration and every resume.
- Verify KFP database connectivity and sufficient disk space (estimated ~2× MLMD database size).
- Confirm all MLMD contexts (runs) have a corresponding row in `run_details`. Orphaned MLMD records (contexts without a `run_details` counterpart) are logged as warnings and skipped rather than failing the entire migration. This handles common partial-failure states from past runs.

**Phase 2 — Schema migration**

Schema changes are handled differently depending on whether a table is new or pre-existing, to avoid GORM `AutoMigrate` silently reverting or corrupting manual modifications to existing tables:

- **New tables** (`artifacts`, `artifact_tasks`): created via manually written SQL (see [schema_changes.sql](./schema_changes.sql)), verified against GORM `AutoMigrate` in CI. No pre-existing data or schema to protect.
- **Modified existing table** (`tasks` only): migrated via a table-copy-and-rename workflow. The legacy table includes cache entries and Workflow node reports (see [schema_changes.sql](./schema_changes.sql)):
  1. Create a temporary copy of the table (`tasks_migration_tmp`) with the target schema, using manually written SQL (not `AutoMigrate`) as the source of truth for the DDL.
  2. Identify legacy cache rows for Phase 3. Skip legacy Workflow node reports with a blank `MLMDExecutionID`, regardless of their reported state. Preserve each run's existing `run_details.WorkflowRuntimeManifest` as-is. The [run reporting design](design-details.md#run-reporting) retires the persistence agent's per-node task writes.
  3. Within a single transaction: rename the existing table to a backup name (`tasks_pre_migration`), rename the temporary table to `tasks`.
  4. After migration completes successfully, drop `tasks_pre_migration` (see [Backup and Recovery](#backup-and-recovery)).

**Schema verification (CI safeguard):** To confirm the manual SQL scripts and `AutoMigrate` never diverge for the *new*-table schemas, CI generates a schema dump after running the manual migration SQL, generates a second dump after running `GORM.AutoMigrate()` against a clean database, and diffs the two. A non-empty diff fails CI. `AutoMigrate` is used here as a schema check only, not to migrate production databases. For `tasks`: **new installs** still rely on `AutoMigrate` to create the post-MLMD `tasks` table from the GORM model, the table-copy-and-rename path above applies only when upgrading an existing pre-migration `tasks` table.

Blocks normal API traffic until schema migration finishes. While migration is in progress the server stays in maintenance mode (see [Maintenance Mode](#maintenance-mode) and [Handling In-Flight Runs](#handling-in-flight-runs)).

**Phase 3 — Data migration**

Use the MLMD execution ID to derive each migrated task UUID. Resolve each legacy cache row with a verified same-run execution to the task created from that execution.

Entities are migrated in dependency order to satisfy foreign key constraints:

```
MLMD Executions  ──►  tasks table
                       (deterministic UUID v5, state mapping, TaskType detection)

MLMD Artifacts   ──►  artifacts table
                       (metrics split into separate rows; UUIDs via UUID v5)

MLMD Events      ──►  artifact_tasks table       ← LAST: FK deps on both above
                       (link tasks to artifacts via UUID v5, preserve IOType)

Second pass      ──►  Resolve parent_task_uuid via uuidv5("execution:"+parent_id)
```

Process each entity type in ascending MLMD ID order using bounded transactions. Committed batches remain available for [target-based resume](#idempotency).

In API Server mode, Phase 3 runs while the server remains in maintenance mode. Normal API traffic resumes only after the [completion checklist](#migration-completion-checklist) passes.

#### New Database Tables

Permanent post-migration tables (`artifacts`, `artifact_tasks`) and the altered `tasks` schema are defined in [schema_changes.sql](./schema_changes.sql).

#### Idempotency

Three overlapping mechanisms ensure the migration is safe to run multiple times:

**1. Deterministic UUID generation (UUID v5)**

MLMD integer IDs are mapped to UUIDs deterministically using UUID v5 (SHA-1 based):

```
Input:  MLMD Execution ID = 12345
Step 1: Namespace string = "execution:12345"
Step 2: SHA-1 hash with DNS namespace
Step 3: Format as UUID v5 = "a1b2c3d4-e5f6-5789-abcd-ef0123456789"
```

The same MLMD ID always produces the same UUID. Live API server writes use random UUID v4; these two namespaces never collide.

**2. Safe insert semantics**

```sql
INSERT IGNORE INTO tasks (UUID, ...) VALUES (...);
```

Duplicate rows (same UUID) are silently skipped. A crashed-and-resumed run encounters `RowsAffected = 0` for already-migrated rows and `RowsAffected = 1` for new ones — no errors, no duplicates.

**3. Resume by binary search**

Commit each insert stream (`tasks`, `artifacts`, `artifact_tasks`) in atomic batches ordered by MLMD ID. This leaves a complete prefix. On restart, binary-search for the first incomplete record by checking its expected rows and fields. Target rows track progress without cursors or phase status.

Before completion and cleanup, validate all expected rows and fields in bounded batches. This catches gaps that binary search can miss. Fail on missing rows or conflicting fields and report the source ID and the mismatch.

#### Migration completion checklist

On API server startup, decide whether to migrate from **schema / progress state**:

1. **Migration required (always try):** the KFP schema is out of date relative to the post-MLMD-removal target **or** a migration is already in progress (for example leftover `tasks_pre_migration`, or the checklist below is only partially satisfied). Enter maintenance mode and migrate/resume.
2. **MLMD connection required when migrating:** if migration is required and MLMD gRPC connection config is missing, **fail startup**. The operator/installer must supply connection info for upgrade boots (see [MLMD connection configuration](#mlmd-connection-configuration)).
3. **Classify before writes.** Fresh and confirmed native installations follow normal startup. Legacy or interrupted MySQL upgrades enter maintenance mode. Fail with a diagnostic if the schema cannot be classified safely.
4. **Already current (skip migration):** schema is already at the target shape **and** the completion predicates below are satisfied (including no leftover temp tables). Skip migrate and serve normally. MLMD connection config is not required in this case.
5. **Migration complete (leave maintenance):** all of:
   - All expected rows pass validation.
   - All eligible same-run parent links are present.
   - `tasks_pre_migration` is absent.

#### MLMD connection configuration

Both entry points read from the MLMD store of the KFP installation being upgraded. Operators must configure that store's endpoint and reuse it on resume. The migrator keeps the source read-only and validates run ownership against `run_details`.

Migration needs the following MLMD gRPC settings:

| Setting | Purpose |
|---|---|
| MLMD gRPC address (host + port) | Connect to `metadata-grpc-service` (or equivalent) to read executions, artifacts, and events |
| Optional TLS | Only if the deployment’s MLMD endpoint requires TLS |
| Optional request headers | Extra gRPC metadata to send with each MLMD call (e.g. `authorization: Bearer …`), so admins can pass whatever auth the endpoint needs |

- **API server mode:** the installer/operator injects these settings into the API server for upgrade boots where migration may run. Detection is schema/progress-based as above.
- **CLI mode:** the same connection info is passed via flags (for example `--mlmd-address`, `--mlmd-header=authorization:Bearer <token>`).
- After cutover, once migration is complete and MLMD is removed, those settings can be omitted; already-current clusters do not need them.

#### MLMD Entity Transformations

**Executions → Tasks (TaskType detection)**

| MLMD Execution Type | Detection Logic | KFP TaskType |
|---|---|---|
| `system.DAGExecution` (no parent) | No `parent_dag_id` | ROOT (1) |
| `system.ContainerExecution` | Type check | RUNTIME (2) |
| `system.DAGExecution` (condition-branches-*) | Name prefix | CONDITION_BRANCH (3) |
| `system.DAGExecution` (condition-*) | Child of CONDITION_BRANCH, or `condition-<number>` directly under ROOT in 2.18 | CONDITION (4) |
| `system.DAGExecution` + `iteration_count` | Custom property | LOOP (5) |
| `system.DAGExecution` (exit-handler-*) | Name prefix; fallbacks: on-exit-*, onexit-*, or custom prop | EXIT_HANDLER (6) |
| `system.ImporterExecution` | Type check | IMPORTER (7) |
| `system.DAGExecution` (with parent) | Has `parent_dag_id` | DAG (8) |

Detection order is significant: specific patterns (exit-handler, condition, loop) are evaluated before falling back to generic DAG.

**Parent/child task resolution (second pass)**

MLMD stores parent links as integer execution IDs (`parent_dag_id` on the child). KFP stores them as `ParentTaskUUID` with a self-FK on `tasks.UUID`. Because the parent must already exist as a KFP UUID before the FK can be set, parent/child wiring is done in two passes:

1. **Pass 1 — Insert all executions as tasks.** For each MLMD execution: generate a deterministic UUID v5, detect `TaskType` (including whether `parent_dag_id` is present), and `INSERT IGNORE` into `tasks` with `ParentTaskUUID` unset/`NULL`.
2. **Pass 2 — Idempotent parent fixup.** Set `ParentTaskUUID` to `uuidv5("execution:"+parent_dag_id)` for migrated parents in the same run. For cross-run parents, keep the child with a NULL parent and warn with both execution IDs.

Example:

```
MLMD:  exec 10 (DAG, no parent)     → ROOT
       exec 20 (Container, parent=10) → RUNTIME
       exec 30 (Container, parent=10) → RUNTIME

Pass 1: tasks rows uuid-10, uuid-20, uuid-30 (ParentTaskUUID NULL)

Pass 2: UPDATE children SET ParentTaskUUID = uuidv5(execution:10) WHERE still NULL
```

The same fixup applies to nested trees. Pass 1 may use `parent_dag_id` for TaskType classification, Pass 2 materializes the FK using UUID v5.

**MLMD State → KFP TaskState**

| MLMD State | KFP TaskState | Notes |
|---|---|---|
| `NEW` | RUNTIME_STATE_UNSPECIFIED (0) | |
| `RUNNING` | RUNNING (1) | |
| `COMPLETE` | SUCCEEDED (2) | |
| `FAILED` | FAILED (4) | |
| `CACHED` | CACHED (5) | |
| `CANCELED` | FAILED (4) | No CANCELED in KFP TaskState; maps to FAILED |

##### Cache fingerprints

Store MLMD fingerprints for `COMPLETE` and `CACHED` executions in
`StatusMetadata.customProperties.legacy_cache_fingerprint`. Reserve
`tasks.Fingerprint` for native lookup keys.

- **Eligibility.** Backfill only original successful producers whose legacy
  cache entries were created in the 90 days before migration start. Cache hits
  never extend that window. Under the migration lock, record the start time in
  `mlmd_migration_state` before the first data batch. Both entry points reuse it
  on retry.
- **Backfill.** Recompute the 2.18 fingerprint from recovered inputs and require
  a match. Generate the native key with the native driver algorithm, which
  includes pipeline identity for automatic keys. Write missing keys in bounded,
  resumable batches once native task and artifact reads can return the outputs.
- **Skips.** Keep task history and report missing or ambiguous inputs, hash
  mismatches, incomplete outputs, and custom keys. Native lookup cannot preserve
  legacy pipeline scope for custom keys. These skips allow migration to complete.

**Native field mappings**

The [runtime design](design-details.md) and [proposed schema](schema_changes.sql) provide the following destinations. The fixture follows the current v2 field and enum definitions.

| Legacy value | Native destination |
|---|---|
| Linked cache row lifecycle times | Use populated `CreatedTimestamp`, `StartedTimestamp`, and `FinishedTimestamp` for the task's corresponding seconds fields. Without a legacy creation time, use the MLMD creation time in seconds. |
| Execution `pod_name` and `pod_uid` | Store in `Pods`. Prefer a populated KFP pod name on a verified linked row. Carry the MLMD UID when it describes the same pod. Use the unspecified pod type when the source has no role. |
| `iteration_index` and `iteration_count` | Store in `TypeAttrs`. Preserve a present zero index. |
| `cached_execution_id` | Store the resolved task UUID in nullable `tasks.CachedTaskUUID` and expose it as optional string `PipelineTask.cached_task_id`. Native cache hits populate the same field with the source task UUID. |
| Artifact fields | Convert creation and update times to `CreatedAtInSec` and `LastUpdateInSec`. For non-metrics artifacts, use the display name or fall back to the source name. Typed scalar metadata and split numeric metrics use `Metadata` and `NumberValue`. |

**Historical field mappings**

- **Timestamps.** Store native times in seconds. Exact timestamps, execution last-update times, and event times stay in MLMD or its backup.
- **Parameters.** Keep typed values. Set I/O types and producer references when the saved spec and captured values establish them. Otherwise leave them unspecified.
- **Lineage.** Link `INPUT` and `OUTPUT` events with migrated endpoints, a known native I/O type, and one nonempty named step. Store the name in `artifact_tasks.ArtifactKey`. Report skipped events. Set producer iteration only for verified indices.
- **Other metadata.** Unmapped fields stay in MLMD or its backup. Use KFP run configuration for the configured pipeline root and launcher configuration for storage credentials.

**Artifacts → Artifacts (metrics splitting)**

MLMD stores scalar metrics as a single artifact with multiple key-value custom properties. KFP stores each metric as a separate row with a `NumberValue` field. A single MLMD metrics artifact with N metrics produces N KFP artifact rows. UUIDs are deterministic: the primary (lineage) row uses `uuidv5("artifact:"+mlmd_id)`, additional metric rows use `uuidv5("artifact:"+mlmd_id+":metric:"+name)` (or a stable metric index). Events that referenced the original MLMD artifact link via `artifact_tasks` to the primary UUID.

#### Concurrent CLI and API migration

MySQL advisory locks prevent the standalone CLI and API server startup migration from running at the same time:

```sql
SELECT GET_LOCK('kfp_mlmd_migration_lock', 10);
```

- CLI and API server acquire the same named lock before Phase 2/3. Whichever acquires it first proceeds, the other exits/skips with a clear "migration already in progress" outcome.
- Locks are session-scoped and auto-release on connection loss or crash — no manual cleanup required.
- The lock does not fence old API replicas; both entry points must complete the maintenance-mode writer fence below before Phase 2/3.

#### Execution Modes

**Mode 1 — Standalone CLI (manual)**

```bash
mlmd-migrate \
  --mlmd-address=metadata-grpc-service:8080 \
  --db-uri=mysql:3306
```

- Runs in foreground, blocks until complete.
- Exits 0 on success, non-zero on failure.
- Refuses Phase 2/3 unless the no-in-flight-run and old-replica gates are satisfied.
- Always idempotent on re-run: discovers the resume point via binary search.
- Configuration via CLI flags.

**Mode 2 — API Server integration (automatic)**

```yaml
env:
  - name: METADATA_GRPC_SERVICE_HOST
    value: metadata-grpc-service
  - name: METADATA_GRPC_SERVICE_PORT
    value: "8080"
```

- At startup, if the schema is out of date or a migration is in progress, enter maintenance mode and run/resume Phase 2–3 only after the no-in-flight-run and old-replica gates pass (see [completion checklist](#migration-completion-checklist)).
- If migration is required and MLMD connection config is missing, refuse to start.
- Normal API traffic is not served until the [completion checklist](#migration-completion-checklist) passes.
- On later restarts, if the schema is current and the checklist already passes, migration work is skipped and the API serves normally (MLMD env may still be present or already removed).
- After a failure the process exits non-zero under maintenance, a restart re-runs the idempotent migrator (binary search + parent fixup).
- When the CLI completed the fenced migration, startup verifies completion. If CLI work was interrupted, startup may resume it after the same writer checks pass.

#### Maintenance Mode

While migration is in progress, the API server enters maintenance mode: user facing APIs (creating/listing/starting runs, and similar control-plane operations) return a clear error such as "Kubeflow Pipelines is down for maintenance. Try again later." so users do not operate against half-migrated data.

Internal runtime traffic from the persistence agent, driver, and launcher must still succeed while existing runs finish and report their final status.

Health/readiness: the pod may be up for migration and that runtime traffic, but readiness should still signal that the service is not ready for normal user traffic.

#### Handling In-Flight Runs

Do not start migration while any pipeline run or Argo Workflow is in flight. Let existing runs finish on the old runtime before the schema and metadata paths change.

**Limitation:** Migration requires a maintenance window and may wait for long-running pipelines. Do not cancel or retry runs as part of the upgrade; if they cannot drain, postpone migration.

Rolling upgrades across the MLMD migration boundary are unsupported. The administrator must complete steps 1–3 before starting either migration path.

Recommended upgrade flow:

1. Block new submissions and retries across all API replicas, and pause recurring-run triggers; keep internal runtime reporting available.
2. Wait for all KFP runs and Argo Workflows to reach terminal state and confirm their final reports are stored. If any remain active, do not migrate.
   Immediately before scaling down the old API, run the read-only CLI command `validate` to check KFP database and MLMD gRPC connectivity and schema support, and list active KFP runs and Argo Workflows. It exits nonzero if any remain active or a check fails. This is a point-in-time check. The administrator remains responsible for keeping new work blocked.
3. Drain and terminate every old API replica; verify none can still serve requests or write to the database, then recheck that no run or Workflow became active. Keep the upgraded API in maintenance mode.
   To terminate the old replicas, scale the old API Deployment to zero and wait for all its Pods to terminate before starting the upgraded API.
4. Run schema and data migration (Phases 2–3). Refuse startup only if migration cannot proceed.
   The upgraded API can run migration automatically at startup once the prerequisites are met.
5. Exit maintenance mode and resume submissions and recurring-run triggers after the [completion checklist](#migration-completion-checklist) passes.

The administrator manages deployment resources and must keep old API replicas and recurring-run triggers stopped throughout migration, including preventing deployment controllers from restoring them. The migrator validates observable prerequisites and refuses migration with actionable errors when they are unmet.

#### Backup and Recovery

- **Before upgrade:** The admin must back up the KFP database.
- **To reverse after a failed or undesirable upgrade:** Restore that backup and revert to the previous API server version.
- **After successful migration:** Drop temporary objects (`tasks_pre_migration`).

#### Error Handling

Transient MLMD gRPC or MySQL connection errors mid-migration are retried with exponential backoff. If retries are exhausted, the process fails fatally as below.

| Error Type | Behavior |
|---|---|
| Individual entity transformation failure | Log warning, skip entity, continue |
| Missing optional MLMD properties | Log warning, use default, continue |
| Migration required but MLMD connection info missing / unreachable at startup | Fatal: refuse to start (cannot migrate) |
| MLMD gRPC connection loss mid-migration | Retry with backoff, if exhausted → fatal: remain in maintenance mode, exit non-zero, surface error in logs |
| MySQL connection loss mid-migration | Retry with backoff, if exhausted → fatal: exit non-zero, surface error in logs |
| Disk space exhaustion | Fatal: exit non-zero, surface error in logs |
| Schema mismatch | Fatal: exit non-zero, surface error in logs |

Recovery: restart the API server or re-run the CLI. Migration is always idempotent (binary search + parent fixup).
Each skip warning includes the entity type, source ID, and reason. Both entry points report migrated and skipped counts by entity type and break down skips by reason. Skipped records remain in MLMD for recovery. Preserve access to the source or its backup until the skip report has been reviewed. After resolving a fatal error, restart the API server or rerun the CLI.

For skipped executions, log the MLMD execution ID as the source ID. Log the final counts of copied and skipped records. Resume and completion checks run independently of these logs, with no persistent skip records.

---

## Risks and Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| Long maintenance window while data migration runs | Medium | Operators may pre-run the CLI to shorten downtime; API stays in maintenance until the [completion checklist](#migration-completion-checklist) passes |
| A run remains active when migration starts | High | Require zero in-flight KFP runs and Argo Workflows with final reports stored before Phase 2/3 |
| A new run starts during the drain | High | Block submissions and retries across old replicas, pause recurring triggers, and recheck after draining replicas |
| Opening traffic before migration is fully done | High | Leave maintenance mode only after the [completion checklist](#migration-completion-checklist) passes (inserts, parent fixup, temp-table cleanup) |

---

## Test Plan

### Unit Tests

- Entity transformation functions for all TaskType detection paths, including edge cases (missing custom properties, unknown execution types, exit-handler fallback chain).
- State mapping for all MLMD → KFP TaskState transitions including `CANCELED`.
- Deterministic UUID v5 generation: same input always produces same UUID; execution and artifact namespaces never collide; no collision with UUID v4 space.
- Metrics artifact splitting: N key-value pairs produce exactly N KFP artifact rows with deterministic primary vs per-metric UUID v5 names.
- Binary-search resume: first missing UUID discovery for executions, artifacts, and events.
- Parent fixup idempotency: re-running converges to zero NULL parent links for tasks that need parents.
- `INSERT IGNORE` idempotency: running the same entity through the engine twice produces exactly one row.
- Cache fingerprint gating: preserve legacy hashes as provenance for `COMPLETE` and `CACHED` executions. Write a native key only for an eligible successful producer after legacy hash verification.

### Integration Tests

- **Full migration:** Cover all task types. Assert counts, UUIDs, states, relationships, skip diagnostics, fingerprint provenance, and native cache hits.
- **Idempotent re-run:** Run migration to completion, then run it again. Assert no duplicate rows and no errors (`INSERT IGNORE` + deterministic UUIDs).
- **Resume after crash:** Inject a failure mid-way through Phase 3. Restart migration. Assert binary search resumes at the first missing ID and the final state matches a clean run.
- **Inconsistent resume state:** Verify an interior gap or conflicting field blocks completion and identifies the source record.
- **Parent fixup resume:** Crash after execution inserts but before/during parent resolution. Restart. Assert unresolved parents are fixed via UUID v5.
- **Completion checklist:** Interrupt before validation is recorded and again before cleanup. Verify an unvalidated upgrade requires MLMD. Verify a validated upgrade finishes cleanup and restarts without MLMD.
- **PostgreSQL boundary:** Verify CLI and startup reject a legacy PostgreSQL schema before writes. Verify fresh and already-native PostgreSQL start without MLMD.
- **Drain gate:** Assert CLI and API migration refuse to start while a KFP run or Argo Workflow is active, and proceed only after terminal reports are stored and old API replicas are gone.
- **Large metrics artifacts:** Seed MLMD with multi-metric artifacts. Assert correct splitting and deterministic UUID v5 assignment. Assert artifact counts stay within an expected ratio vs MLMD when metrics are split.
- **Orphaned MLMD contexts:** Mix supported records with unsupported records and MLMD contexts without matching `run_details` rows. Verify skip IDs, reasons, counts, source preservation, and fidelity for supported records. Inject a schema or infrastructure failure and verify that completion is never recorded, and remaining data is migrated correctly.

### API-level migration validation

Automate an end-to-end CI job or integration suite that proves migrated history is still correct when read the same way operators and UIs do, through the KFP API, not only via direct SQL asserts.

- Bring up a disposable MySQL environment with MLMD and pre- and post-migration KFP API server images. Exercise the standalone CLI and API-server startup upgrade paths.
- Use checked-in **test data** (or a small generator) for a known graph: multiple runs covering each TaskType, nested DAGs (parent links), artifacts including multi-metric splits, and events linking tasks↔artifacts. Load that data into both MLMD and any required KFP run/`run_details` rows so Phase 1 orphan rules do not skip it.
- Reuse one populated upgrade fixture across the API reads. Define request and response assertions for the run, task, artifact, and lineage behaviors the [runtime design](design-details.md#frontend-changes) relies on. Cover both list and get when they exercise different filters, projections, or relationships. Keep field transforms and edge cases in focused tests.
- Capture a golden snapshot before migration and verify the upgraded run, task, artifact, and lineage API reads. Check task IDs, states, parent links, and counts against eligible MLMD executions and their mapped IDs. Check artifact lineage against MLMD events. Verify that saved Workflow manifests are unchanged and that skipped reports add no native task rows. Use the existing Go/Python API clients for API checks.
- Run this job on PRs that touch migration code. Keep the golden snapshot in-repo (or generate it in-job) so failures show a concrete API diff in CI logs.

### Scale and duration tests

Run migration against large synthetic histories, including CLI-first upgrades after the writer fence, and record how long schema and data migration take so operators can plan maintenance windows.

### Backup and Recovery Tests

- Assert parent-resolution (Pass 2) completes and `ParentTaskUUID` links are correct (UUID v5 from MLMD parent ids).
- Assert that after the [completion checklist](#migration-completion-checklist) passes, `tasks_pre_migration` is gone.
- Recovery path is restore from a pre-migration DB backup plus the previous API server image.

---

## Implementation Plan

1. **Migration engine** — Build shared orchestration, transforms, resume-by-binary-search, parent fixup, completion checklist, and validation used by both entry points.
2. **Standalone CLI** — Thin wrapper that runs the engine manually.
3. **API Server integration** — On startup, detect schema-out-of-date or migration-in-progress and always migrate under maintenance mode, require MLMD connection config when migrating (fail otherwise), resume normal serving after the completion checklist passes. Document operator-injected MLMD gRPC settings.
4. **Schema** — Create new tables (`artifacts`, `artifact_tasks`) via manual SQL, keep GORM models in sync, add a CI job that diffs manual-SQL and `AutoMigrate` schema dumps for those new-table schemas. Implement the `tasks` table-copy-and-rename path via manual SQL for upgrades from the pre-migration `tasks` shape, new installs continue to create `tasks` via `AutoMigrate` from the GORM model.
5. **Unit tests** — Cover all transformation paths, UUID generation, and idempotency.
6. **Integration tests** — Full migration, idempotent re-run, resume-after-crash via binary search, parent fixup resume, completion checklist / maintenance gating, post-success cleanup of temp tables.
7. **Documentation** — Operator guide covering pre-upgrade DB backup, injecting MLMD connection settings for upgrade boots, maintenance mode, draining all runs before migration, both execution modes, and restore-from-backup recovery.
