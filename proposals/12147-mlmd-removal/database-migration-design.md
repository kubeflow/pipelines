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

1. **Data preservation** — All executions, artifacts, and events recorded in MLMD are migrated into the KFP native schema with full fidelity.
2. **Safe upgrade** — On startup, always migrate when the schema is out of date or a migration is in progress. During migration the API runs in maintenance mode. Normal traffic resumes only after migration completes successfully. Startup fails when migration is required but cannot run (for example, missing MLMD connection info).
3. **Idempotency** — The migration can be run multiple times. Repeated runs produce no duplicates and leave the database in a consistent state.
4. **Resumability** — A migration interrupted by a pod crash, SIGTERM, or node failure can resume by discovering the first missing migrated ID and continuing. Already-migrated rows are not replayed as new work.
5. **Dual execution modes** — Operators can run migration automatically on API server startup or manually via a standalone CLI tool.

---

## Proposal

### User Stories

**Story 1: Admin upgrading an existing KFP deployment**

> As a KFP platform admin, I want to upgrade from a MLMD-backed version of KFP to a MLMD-free version without losing any historical pipeline run data or artifact lineage, so that my team can continue to audit and debug past runs after the upgrade.

**Story 2: Admin preferring manual control**

> As a KFP admin who manages migration windows carefully, I want to run the migration as a standalone job on my own schedule, before or after the API server upgrade, so I can control when the extra database load occurs and verify results before proceeding.

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
│   resume-by-binary-search, validation)│
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

- Verify MLMD gRPC connectivity.
- Verify MySQL connectivity and sufficient disk space (estimated ~2× MLMD database size).
- Confirm all MLMD contexts (runs) have a corresponding row in `run_details`. Orphaned MLMD records (contexts without a `run_details` counterpart) are logged as warnings and skipped rather than failing the entire migration. This handles common partial-failure states from past runs.

**Phase 2 — Schema migration**

Schema changes are handled differently depending on whether a table is new or pre-existing, to avoid GORM `AutoMigrate` silently reverting or corrupting manual modifications to existing tables:

- **New tables** (`artifacts`, `artifact_tasks`, `mlmd_id_map`): created via manually written SQL (see [schema_changes.sql](./schema_changes.sql)), verified against GORM `AutoMigrate` in CI. No pre-existing data or schema to protect.
- **Modified existing table** (`tasks` only): migrated via a table-copy-and-rename workflow rather than in-place `AutoMigrate`. The current `tasks` table stores cache fingerprints only; the post-migration schema is incompatible (see [schema_changes.sql](./schema_changes.sql)):
  1. Create a temporary copy of the table (`tasks_migration_tmp`) with the target schema, using manually written SQL (not `AutoMigrate`) as the source of truth for the DDL.
  2. Backfill `tasks_migration_tmp` from the existing `tasks` table (cache-fingerprint rows), mapping old columns into the new schema.
  3. Within a single transaction: rename the existing table to a backup name (`tasks_pre_migration`), rename the temporary table to `tasks`.
  4. After migration completes successfully, drop `tasks_pre_migration` (see [Backup and Recovery](#backup-and-recovery)).

**Schema verification (CI safeguard):** To confirm the manual SQL scripts and `AutoMigrate` never diverge for the *new*-table schemas, CI generates a schema dump after running the manual migration SQL, generates a second dump after running `GORM.AutoMigrate()` against a clean database, and diffs the two. A non-empty diff fails CI. `AutoMigrate` is used here as a schema check only, not to migrate production databases. This does not apply to `tasks`, which never goes through `AutoMigrate` directly.

Blocks normal API traffic until schema migration finishes. While migration is in progress the server stays in maintenance mode (see [Maintenance Mode](#maintenance-mode) and [Handling In-Flight Runs](#handling-in-flight-runs)).

**Phase 3 — Data migration**

Entities are migrated in dependency order to satisfy foreign key constraints:

```
MLMD Executions  ──►  tasks table
                       (deterministic UUID, state mapping, TaskType detection)

MLMD Artifacts   ──►  artifacts table
                       (metrics split into separate rows, register in mlmd_id_map)

MLMD Events      ──►  artifact_tasks table       ← LAST: FK deps on both above
                       (link tasks to artifacts, preserve IOType)

Second pass      ──►  Resolve parent_task_uuid references via mlmd_id_map
Cleanup         ──►  Drop mlmd_id_map
```

Each entity type is processed in serial batches of a fixed size (define a default batch size) in ascending MLMD ID order. Each batch runs in its own transaction using `INSERT IGNORE`. After a crash, already-committed batches stay in the database; the next run binary-searches for the first MLMD ID whose deterministic UUID is missing and continues from there.

In API Server mode, Phase 3 runs while the server remains in maintenance mode. Normal API traffic resumes only after the [completion checklist](#migration-completion-checklist) passes.

#### New Database Tables


**`mlmd_id_map`** — temporary map of MLMD integer IDs to KFP UUIDs for parent-task resolution during migration:

```sql
CREATE TABLE mlmd_id_map (
    mlmd_entity_type VARCHAR(20)  NOT NULL,   -- "execution" or "artifact"
    mlmd_id          BIGINT       NOT NULL,
    kfp_uuid         VARCHAR(191) NOT NULL,
    migrated_at      BIGINT       NOT NULL,
    PRIMARY KEY (mlmd_entity_type, mlmd_id)
);
```

`mlmd_id_map` is a temporary table and drop it only after the parent-resolution pass finishes.

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

After a crash, for each insert stream (`tasks`, `artifacts`, `artifact_tasks`) the engine binary searches over ascending MLMD IDs and checks the KFP DB for the deterministic UUID. The first missing ID is the resume point, migration continues with per batch transactions from there.

After execution inserts, run an **idempotent parent fixup** until no unresolved rows remain: for each child that had `parent_dag_id` in MLMD and still has `ParentTaskUUID IS NULL`, set `ParentTaskUUID = uuidv5("execution:"+parent_dag_id)` (map lookup optional). When that set is empty, the parent stage is done.

#### Migration completion checklist

On API server startup, decide whether to migrate from **schema / progress state**:

1. **Migration required (always try):** the KFP schema is out of date relative to the post-MLMD-removal target **or** a migration is already in progress (for example leftover `tasks_pre_migration` / `mlmd_id_map`, or the checklist below is only partially satisfied). Enter maintenance mode and migrate/resume.
2. **MLMD connection required when migrating:** if migration is required and MLMD gRPC connection config is missing, **fail startup**. The operator/installer must supply connection info for upgrade boots (see [MLMD connection configuration](#mlmd-connection-configuration)).
3. **Already current (skip migration):** schema is already at the target shape **and** the completion predicates below are satisfied (including no leftover temp tables). Skip migrate and serve normally. MLMD connection config is not required in this case.
4. **Migration complete (leave maintenance):** all of:
   - Insert stages done: for executions, artifacts, and events, the first missing deterministic key is past the max migratable MLMD ID (and/or counts match migratable MLMD entities after Phase 1 skip rules).
   - Parent fixup done: zero tasks that need a parent still have `ParentTaskUUID IS NULL`.
   - Cleanup done: `mlmd_id_map` and `tasks_pre_migration` are absent.

#### MLMD connection configuration

Migration needs MLMD gRPC connection settings on the migrator entry points:

| Setting | Purpose |
|---|---|
| MLMD gRPC address (host + port) | Connect to `metadata-grpc-service` (or equivalent) to read executions, artifacts, and events |
| Optional TLS / credentials | Only if the deployment’s MLMD endpoint requires them |

- **API server mode:** the installer/operator injects these settings into the API server for upgrade boots where migration may run. Detection is schema/progress-based as above.
- **CLI mode:** the same connection info is passed via flags (for example `--mlmd-address`).
- After cutover, once migration is complete and MLMD is removed, those settings can be omitted; already-current clusters do not need them.

#### MLMD Entity Transformations

**Executions → Tasks (TaskType detection)**

| MLMD Execution Type | Detection Logic | KFP TaskType |
|---|---|---|
| `system.DAGExecution` (no parent) | No `parent_dag_id` | ROOT (0) |
| `system.ContainerExecution` | Type check | RUNTIME (1) |
| `system.DAGExecution` (condition-branches-*) | Name prefix | CONDITION_BRANCH (2) |
| `system.DAGExecution` (condition-*) | Name prefix, child of CONDITION_BRANCH | CONDITION (3) |
| `system.DAGExecution` + `iteration_count` | Custom property | LOOP (4) |
| `system.DAGExecution` (exit-handler-*) | Name prefix; fallbacks: on-exit-*, onexit-*, or custom prop | EXIT_HANDLER (5) |
| `system.ImporterExecution` | Type check | IMPORTER (6) |
| `system.DAGExecution` (with parent) | Has `parent_dag_id` | DAG (7) |

Detection order is significant: specific patterns (exit-handler, condition, loop) are evaluated before falling back to generic DAG.

**Parent/child task resolution (second pass)**

MLMD stores parent links as integer execution IDs (`parent_dag_id` on the child). KFP stores them as `ParentTaskUUID` with a self-FK on `tasks.UUID`. Because the parent must already exist as a KFP UUID before the FK can be set, parent/child wiring is done in two passes:

1. **Pass 1 — Insert all executions as tasks.** For each MLMD execution: generate a deterministic UUID v5, detect `TaskType` (including whether `parent_dag_id` is present), `INSERT IGNORE` into `tasks` with `ParentTaskUUID` unset/`NULL`, and optionally register `(execution, mlmd_id) → kfp_uuid` in `mlmd_id_map`.
2. **Pass 2 — Idempotent parent fixup.** For each migrated task that had a `parent_dag_id` in MLMD and still has `ParentTaskUUID IS NULL`, set `ParentTaskUUID` to `uuidv5("execution:"+parent_dag_id)` (or via `mlmd_id_map`). Re-run until zero unresolved rows remain.
3. **Drop `mlmd_id_map`.** Only after Pass 2 has fully committed. Parent links then live only on `tasks.ParentTaskUUID`.

Example:

```
MLMD:  exec 10 (DAG, no parent)     → ROOT
       exec 20 (Container, parent=10) → RUNTIME
       exec 30 (Container, parent=10) → RUNTIME

Pass 1: tasks rows uuid-10, uuid-20, uuid-30 (ParentTaskUUID NULL)

Pass 2: UPDATE children SET ParentTaskUUID = uuidv5(execution:10) WHERE still NULL

Then: DROP TABLE mlmd_id_map
```

The same fixup applies to nested trees. Pass 1 may use `parent_dag_id` for TaskType classification, Pass 2 materializes the FK. Dropping the map before Pass 2 is finished would leave `ParentTaskUUID` unset only if Pass 2 still depended on the map, prefer computing parent UUIDs with UUID v5 so Pass 2 does not require the map to remain.

**MLMD State → KFP TaskState**

| MLMD State | KFP TaskState | Notes |
|---|---|---|
| `NEW` | RUNTIME_STATE_UNSPECIFIED (0) | |
| `RUNNING` | RUNNING (1) | |
| `COMPLETE` | SUCCEEDED (2) | |
| `FAILED` | FAILED (4) | |
| `CACHED` | CACHED (5) | |
| `CANCELED` | FAILED (4) | No CANCELED in KFP TaskState; maps to FAILED |

Cache fingerprints are copied only for `COMPLETE` or `CACHED` executions to prevent incomplete executions from poisoning the cache.

**Artifacts → Artifacts (metrics splitting)**

MLMD stores scalar metrics as a single artifact with multiple key-value custom properties. KFP stores each metric as a separate row with a `NumberValue` field. A single MLMD metrics artifact with N metrics produces N KFP artifact rows. Only the first metric's UUID is registered in `mlmd_id_map` — it is the UUID that `artifact_tasks` uses when linking events back to the original MLMD artifact. (See Open Questions #1.)

#### Concurrent CLI and API migration

MySQL advisory locks prevent the standalone CLI and API server startup migration from running at the same time:

```sql
SELECT GET_LOCK('kfp_mlmd_migration_lock', 10);
```

- CLI and API server acquire the same named lock before Phase 2/3. Whichever acquires it first proceeds, the other exits/skips with a clear "migration already in progress" outcome.
- Locks are session-scoped and auto-release on connection loss or crash — no manual cleanup required.

#### Execution Modes

**Mode 1 — Standalone CLI (manual)**

```bash
mlmd-migrate \
  --mlmd-address=metadata-grpc-service:8080 \
  --mysql-host=mysql:3306
```

- Runs in foreground, blocks until complete.
- Exits 0 on success, non-zero on failure.
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

- At startup, if the schema is out of date or a migration is in progress, always enter maintenance mode and run/resume Phase 2–3 (see [completion checklist](#migration-completion-checklist)).
- If migration is required and MLMD connection config is missing, refuse to start.
- Normal API traffic is not served until the [completion checklist](#migration-completion-checklist) passes.
- On later restarts, if the schema is current and the checklist already passes, migration work is skipped and the API serves normally (MLMD env may still be present or already removed).
- After a failure the process exits non-zero under maintenance, a restart re-runs the idempotent migrator (binary search + parent fixup).

#### Maintenance Mode

While migration is in progress, the API server enters maintenance mode: user facing APIs (creating/listing/starting runs, and similar control-plane operations) return a clear error such as "Kubeflow Pipelines is down for maintenance. Try again later." so users do not operate against half-migrated data.

Internal runtime traffic from the persistence agent, driver, and launcher must still succeed so suspended in-flight steps can finish and report status before cancel and migrate.

Health/readiness: the pod may be up for migration and that runtime traffic, but readiness should still signal that the service is not ready for normal user traffic.

#### Handling In-Flight Runs

Pipelines that are already running before the upgrade cannot safely keep going once the system switches to the new metadata path. Prefer finishing the current step, then canceling and retrying after migration.

**Limitation:** Migration requires a maintenance window. Clients get a predictable maintenance response instead of partial failures, but new runs cannot be submitted until migration completes. In-flight runs are not resumed mid-pipeline, they are canceled after the current step finishes and retried after the upgrade.

Recommended upgrade flow:

1. Enter maintenance mode (block new user submissions, keep persistence agent / driver / launcher working so the current step can still finish and report).
2. Suspend in-flight Argo Workflow objects so the **current** step can finish.
3. Once workflows are suspended, cancel those pipeline runs. Cancel + later retry is required so Argo Workflows are recompiled against the new driver and launcher images.
4. Run schema and data migration (Phases 2–3). Refuse startup only if migration cannot proceed.
5. Exit maintenance mode after the [completion checklist](#migration-completion-checklist) passes.
6. Retry the canceled pipeline runs after the upgrade.

Wait until migration finishes successfully before retrying canceled runs.

#### Backup and Recovery

- **Before upgrade:** The admin must back up the KFP database.
- **To reverse after a failed or undesirable upgrade:** Restore that backup and revert to the previous API server version.
- **After successful migration:** Drop temporary objects (`tasks_pre_migration`, `mlmd_id_map`).

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

---

## Risks and Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| Long maintenance window while data migration runs | Medium | Operators may pre-run the CLI to shorten downtime; API stays in maintenance until the [completion checklist](#migration-completion-checklist) passes |
| Retrying canceled runs before migration completes | High | Document and gate retries on the [completion checklist](#migration-completion-checklist) |
| Canceling mid-step loses in-progress work | Medium | Suspend Argo Workflows first so the current step can finish, then cancel |
| Opening traffic before migration is fully done | High | Leave maintenance mode only after the [completion checklist](#migration-completion-checklist) passes (inserts, parent fixup, temp-table cleanup) |

---

## Test Plan

### Unit Tests

- Entity transformation functions for all TaskType detection paths, including edge cases (missing custom properties, unknown execution types, exit-handler fallback chain).
- State mapping for all MLMD → KFP TaskState transitions including `CANCELED`.
- Deterministic UUID v5 generation: same input always produces same UUID; execution and artifact namespaces never collide; no collision with UUID v4 space.
- Metrics artifact splitting: N key-value pairs produce exactly N KFP artifact rows; only first UUID registered in `mlmd_id_map`.
- Binary-search resume: first missing UUID discovery for executions, artifacts, and events.
- Parent fixup idempotency: re-running converges to zero NULL parent links for tasks that need parents.
- `INSERT IGNORE` idempotency: running the same entity through the engine twice produces exactly one row.
- Cache fingerprint gating: fingerprint present only for `COMPLETE`/`CACHED` states.

### Integration Tests

- **Full migration:** Seed a test MLMD database with executions, artifacts, and events across all TaskTypes. Run migration. Assert KFP tables contain expected records with correct state, UUID, type, and relationships. Assert entity counts (executions ≈ tasks; events = artifact_tasks), TaskType distribution, foreign-key integrity, and that only `SUCCEEDED`/`CACHED` tasks carry cache fingerprints.
- **Idempotent re-run:** Run migration to completion, then run it again. Assert no duplicate rows and no errors (`INSERT IGNORE` + deterministic UUIDs).
- **Resume after crash:** Inject a failure mid-way through Phase 3. Restart migration. Assert binary search resumes at the first missing ID and the final state matches a clean run.
- **Parent fixup resume:** Crash after execution inserts but before/during parent resolution. Restart. Assert unresolved parents are fixed and then `mlmd_id_map` is dropped.
- **Completion checklist:** Assert maintenance mode remains until inserts, parent fixup, and temp-table cleanup all pass, assert migrate is attempted when schema is out of date or migration is in progress, assert startup fails when migration is required but MLMD connection config is missing, assert skip when schema is already current and checklist passes.
- **Large metrics artifacts:** Seed MLMD with multi-metric artifacts. Assert correct splitting and `mlmd_id_map` registration. Assert artifact counts stay within an expected ratio vs MLMD when metrics are split.
- **Orphaned MLMD contexts:** Include MLMD contexts without matching `run_details` rows. Assert they are skipped with a warning and remaining data is migrated correctly.

### Backup and Recovery Tests

- Assert parent-resolution (Pass 2) completes and `ParentTaskUUID` links are correct **before** `mlmd_id_map` is dropped.
- Assert that after the [completion checklist](#migration-completion-checklist) passes, `tasks_pre_migration` and `mlmd_id_map` are gone.
- Recovery path is restore from a pre-migration DB backup plus the previous API server image.

---

## Implementation Plan

1. **Migration engine** — Build shared orchestration, transforms, resume-by-binary-search, parent fixup, completion checklist, and validation used by both entry points.
2. **Standalone CLI** — Thin wrapper that runs the engine manually.
3. **API Server integration** — On startup, detect schema-out-of-date or migration-in-progress and always migrate under maintenance mode, require MLMD connection config when migrating (fail otherwise), resume normal serving after the completion checklist passes. Document operator-injected MLMD gRPC settings.
4. **Schema** — Create new tables (`artifacts`, `artifact_tasks`, `mlmd_id_map`) via manual SQL, keep GORM models in sync, add a CI job that diffs manual-SQL and `AutoMigrate` schema dumps for those new-table schemas. Implement the `tasks` table-copy-and-rename path via manual SQL only (`tasks` never uses `AutoMigrate`).
5. **Unit tests** — Cover all transformation paths, UUID generation, and idempotency.
6. **Integration tests** — Full migration, idempotent re-run, resume-after-crash via binary search, parent fixup resume, completion checklist / maintenance gating, post-success cleanup of temp tables.
7. **Documentation** — Operator guide covering pre-upgrade DB backup, injecting MLMD connection settings for upgrade boots, maintenance mode, suspend → cancel → migrate → retry for in-flight runs, both execution modes, and restore-from-backup recovery.
