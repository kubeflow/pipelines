# MLMD Database Migration Design

## Overview

This document describes the design for migrating historical pipeline metadata from ML Metadata (MLMD) to native KFP database tables as part of the MLMD removal effort.

## Motivation

When upgrading KFP to a version without MLMD, existing deployments have historical data (executions, artifacts, events) stored in MLMD that must be preserved. This migration design addresses the challenge of moving this data to the new KFP native schema with the following goals:
- Minimizing operational disruption (zero-downtime migration)
- Providing flexibility for different operational constraints (manual vs automatic execution)
- Ensuring data integrity and safety (idempotent, resumable on failure)

## Goals

1. **Idempotent Migration**: Can run multiple times, migrating only new data since last run
2. **Zero-Downtime**: Migration runs while API server serves traffic
3. **Dual Execution Modes**: Support both standalone CLI tool and API server integration
4. **Safe for Multi-Replica**: Database locks prevent concurrent migration
5. **Resumable**: Can resume after interruption (pod crash, SIGTERM)

## Design Details

### Architecture

The migration system consists of:

1. **Migration Engine** (`backend/src/apiserver/migration/`)
   - Core migration logic used by both execution modes   

2. **Standalone CLI Tool** (`backend/cmd/mlmd-migrate/`)
   - Compiled Go binary for manual execution
   - Wrapper that parses CLI flags, calls migration engine, exits

3. **API Server Integration** (`backend/src/apiserver/main.go`)
   - Wrapper for automatic execution
   - Checks env vars, spawns background goroutine

**Note:** Both modes use the **exact same migration logic**. Only the entry point differs.

### File Structure

```
backend/src/apiserver/
├── migration/                          # NEW: Shared migration engine
│   ├── manager.go                      # Migration orchestration & state machine
│   ├── transformers.go                 # MLMD entity → KFP model transformations
│   ├── checkpoint.go                   # Idempotency tracking & resume logic
│   ├── validation.go                   # Pre/post migration validation
│   └── models.go                       # Migration tracking database models
│
├── model/
│   ├── migration_status.go             # NEW: Migration status tracking
│   ├── migration_checkpoint.go         # NEW: Resume checkpoint tracking
│   └── mlmd_id_map.go                  # NEW: MLMD ID → KFP UUID mapping
│
└── main.go                             # MODIFIED: Add migration check on startup

backend/cmd/
└── mlmd-migrate/                       # NEW: Standalone CLI tool
    └── main.go                         # CLI wrapper around migration engine
```

### Migration Data Flow

```
MLMD Database                 KFP Database
━━━━━━━━━━━━━                 ━━━━━━━━━━━━━━━━━━━━
Contexts (Runs)      ──►  Validate against run_details (no insert)

Executions           ──►  tasks table
                          - Generate deterministic UUID
                          - Map state, timestamps, parameters
                          - Detect TaskType (ROOT, RUNTIME, LOOP, etc.)

Artifacts            ──►  artifacts table
                          - Split metrics into separate rows
                          - Map to ArtifactType enum
                          - Store metrics with NumberValue field

Events               ──►  artifact_tasks table  (LAST — has FKs to both tasks and artifacts)
                          - Link artifacts to tasks
                          - Preserve IOType and producer info
```

> **Dependency order is critical.** `artifact_tasks` has foreign keys to both `tasks` and `artifacts`, so it must be migrated last. Migrating it before its dependencies would cause FK constraint violations.

### Database Schema Changes

#### New Tables for Migration Tracking

**`migration_status` table:**
```sql
CREATE TABLE migration_status (
    ID               int AUTO_INCREMENT PRIMARY KEY,
    MigrationName    varchar(128) NOT NULL UNIQUE,
    Status           varchar(32) NOT NULL,  -- NOT_STARTED, IN_PROGRESS, COMPLETED, FAILED
    StartedAt        bigint,
    CompletedAt      bigint,
    LastCheckpoint   text,                   -- JSON checkpoint data
    ErrorMessage     text,
    TotalEntities    bigint DEFAULT 0,
    MigratedEntities bigint DEFAULT 0,
    CurrentStage     varchar(64)
);
```

**Purpose:** Track migration progress and status across restarts.

**`mlmd_id_map` table:**
```sql
CREATE TABLE mlmd_id_map (
    mlmd_entity_type varchar(20) NOT NULL,  -- "execution" or "artifact"
    mlmd_id          bigint NOT NULL,
    kfp_uuid         varchar(191) NOT NULL,
    migrated_at      bigint NOT NULL,
    PRIMARY KEY (mlmd_entity_type, mlmd_id)
);
```

**Purpose:** Map MLMD integer IDs to KFP UUIDs for:
- Resolving parent task references
- Rollback support
- Audit trail

**Lifecycle:** Temporary table, can be dropped after 30-90 day retention period.

### Idempotency Strategy

Migration can be run multiple times safely without creating duplicate records. This is achieved through three mechanisms:

#### 1. Deterministic UUID Generation

**Problem:** MLMD uses integer IDs (e.g., `execution_id = 12345`), but KFP uses UUIDs (e.g., `task_uuid = "abc-def-123"`). We need to convert IDs consistently.

**Solution:** Use UUID v5 (SHA1-based) to generate the same UUID every time for the same MLMD ID.

**How it works:**
```
Input:  MLMD Execution ID = 12345
Step 1: Create string = "execution:12345"
Step 2: Hash with SHA1 using DNS namespace = a1b2c3d4...
Step 3: Format as UUID v5 = "a1b2c3d4-e5f6-5789-abcd-ef0123456789"

Key property: Same input (12345) ALWAYS produces same UUID
```

**Example:**
```
First run:  Execution 12345 → UUID "a1b2c3d4-e5f6-5789-abcd-ef0123456789"
Second run: Execution 12345 → UUID "a1b2c3d4-e5f6-5789-abcd-ef0123456789" (identical!)
```

**Why this matters:** If migration crashes and reruns, the same MLMD execution generates the same UUID, so `INSERT IGNORE` will skip it (no duplicate).

**In-flight data safety:** The new API server writes new runs using random UUID v4, while migrated records use deterministic UUID v5. These UUID spaces never collide. Combined with `INSERT IGNORE`, this means the migration can never overwrite a record created by the live API server,even if both are writing to the `tasks` table simultaneously.

#### 2. Safe Insert Pattern

Use MySQL's `INSERT IGNORE` to skip records that already exist:

```sql
INSERT IGNORE INTO tasks (UUID, Name, State, ...) VALUES (...);
```

**Behavior:**
- If UUID already exists: Skip insert (RowsAffected = 0), no error
- If UUID is new: Insert record (RowsAffected = 1)

**Example scenario:**
```
Run 1: Migrates executions 1-1000 (inserts 1000 tasks)
       Crashes at execution 500
Run 2: Tries to migrate executions 1-1000 again
       - Executions 1-500: Same UUIDs, INSERT IGNORE skips them
       - Executions 501-1000: New inserts succeed
Result: No duplicates, migration completes!
```

#### 3. Checkpoint-Based Resume

Track migration progress to avoid re-processing large batches:

**Checkpoint structure:**
```json
{
  "last_execution_id": 1000,
  "last_artifact_id": 5000,
  "last_migration_timestamp": 1709000000
}
```

**Resume logic:**
```sql
-- Query MLMD for only unprocessed entities
SELECT * FROM Execution
WHERE id > 1000  -- Resume from checkpoint
ORDER BY id ASC
```

**Combined with deterministic UUIDs:**
- Checkpoint reduces redundant work (skip querying already-migrated entities)
- Deterministic UUIDs ensure safety if checkpoint is lost (INSERT IGNORE prevents duplicates)

### MLMD Entity Transformations

#### Executions → Tasks

**Task Type Detection:**

| MLMD Execution Type | Detection Logic | KFP TaskType |
|---------------------|----------------|--------------|
| `system.DAGExecution` (root) | No `parent_dag_id` | ROOT (0) |
| `system.ContainerExecution` | Type check | RUNTIME (1) |
| `system.DAGExecution` (condition-branches-*) | Name prefix | CONDITION_BRANCH (2) |
| `system.DAGExecution` (condition-*) | Name prefix, child of CONDITION_BRANCH | CONDITION (3) |
| `system.DAGExecution` + `iteration_count` | Custom property | LOOP (4) |
| `system.DAGExecution` (exit-handler-*) | Name prefix + fallbacks (see below) | EXIT_HANDLER (5) |
| `system.ImporterExecution` | Type check | IMPORTER (6) |
| `system.DAGExecution` (with parent) | Has `parent_dag_id` | DAG (7) |

> **Detection order matters.** Check specific patterns (exit-handler, condition, loop) *before* falling back to generic DAG. Exit handler detection uses name prefix `exit-handler-*` as the primary signal, with fallbacks: `on-exit-*` or `onexit-*` prefixes, and `exit_handler` custom property.

**MLMD State → KFP TaskState Mapping:**

| MLMD Execution State | KFP TaskState | Notes |
|---------------------|---------------|-------|
| `NEW` | RUNTIME_STATE_UNSPECIFIED (0) | Execution created but not started |
| `RUNNING` | RUNNING (1) | |
| `COMPLETE` | SUCCEEDED (2) | |
| `FAILED` | FAILED (4) | |
| `CACHED` | CACHED (5) | |
| `CANCELED` | FAILED (4) | No CANCELED in KFP TaskState; maps to FAILED |

> **Note:** `SKIPPED (3)` is not mapped from MLMD — it is only used by the new API server for conditional branches that were not taken.

**Namespace Resolution:**

MLMD sub-executions don't always carry the namespace. The migration resolves it in priority order:
1. Execution's own `namespace` custom property (if present)
2. Fallback: parent `run_details.Namespace` (always available)

This ensures `Namespace NOT NULL` constraints are satisfied for all migrated tasks and artifacts.

**Cache Fingerprint Handling:**
- Only set `Fingerprint` field for executions in `COMPLETE` or `CACHED` state
- Leave fingerprint empty for `FAILED` or `RUNNING` executions
- Per upstream requirement: prevent caching from incomplete executions

**Example Transformation:**
```
MLMD Execution:
  id: 12345
  type: "system.ContainerExecution"
  name: "preprocess-data"
  state: COMPLETE
  cache_fingerprint: "abc123"

→ KFP Task:
  UUID: "deterministic-uuid-for-12345"
  Type: RUNTIME (1)
  Name: "preprocess-data"
  State: SUCCEEDED (2)
  Fingerprint: "abc123"  ← Only set because state is COMPLETE
```

#### Artifacts → Artifacts

**Metrics Handling:**
- MLMD stores metrics as single artifact with multiple key-value pairs
- KFP stores each metric as **separate artifact row** with `Type=Metric` and `NumberValue`

**Example:**
```
MLMD Artifact (id: 100):
  type: "system.Metrics"
  custom_properties: {
    "accuracy": 0.95,
    "loss": 0.05
  }

→ KFP Artifacts:
  [
    {
      UUID: "deterministic-uuid-100-accuracy",
      Type: Metric (6),
      NumberValue: 0.95
    },
    {
      UUID: "deterministic-uuid-100-loss",
      Type: Metric (6),
      NumberValue: 0.05
    }
  ]
```

**Why separate rows?** Enables efficient querying and indexing of individual metrics.

**ID Mapping for split metrics:** When a single MLMD artifact produces multiple KFP rows, only the *first* metric's UUID is registered in `mlmd_id_map`. This is the UUID that `artifact_tasks` will reference when linking Events to the original MLMD artifact.

### Execution Modes

#### Mode 1: Standalone CLI Tool (Manual)

**Usage:**
```bash
mlmd-migrate \
  --mlmd-address=metadata-grpc-service:8080 \
  --mysql-host=mysql:3306 \
  --batch-size=5000 \
  --workers=8
```

**Behavior:**
- Runs in foreground (blocks until complete)
- Exits with status code (0=success, non-zero=failure)
- Admin has full control over timing and resources

**Configuration:** CLI flags

#### Mode 2: API Server Integration (Automatic)

**Configuration:**
```yaml
env:
  - name: MLMD_MIGRATE
    value: "true"
  - name: RETRY_FAILED_MIGRATION
    value: "false"            # Set to "true" to auto-retry after failure
  - name: MIGRATION_BATCH_SIZE
    value: "1000"
  - name: MIGRATION_WORKERS
    value: "2"
```

**Behavior:**
- API server checks migration status on startup
- If `MLMD_MIGRATE=true` and status is `NOT_STARTED`:
  - Schema migration (Phase 2) runs synchronously (~5-10s)
  - Data migration (Phase 3) spawns as background goroutine
  - API server starts serving traffic immediately after Phase 2
- If status is `COMPLETED`: skips migration entirely
- If status is `FAILED` and `RETRY_FAILED_MIGRATION=true`: retries in background
- Historical data appears progressively as background migration completes

**Configuration:** Environment variables

### Multi-Replica Safety

**Problem:** In multi-replica deployments, multiple API servers might attempt migration simultaneously.

**Solution:** MySQL advisory locks

```sql
SELECT GET_LOCK('kfp_mlmd_migration_lock', 10);
-- Returns 1 if lock acquired
-- Returns 0 if another replica holds lock
-- Lock is session-scoped: auto-released on crash
```

**Behavior:**
- Replica 1: Acquires lock → runs migration
- Replica 2-N: Fail to acquire → log and skip migration
- All replicas serve traffic normally

### Migration Phases

**Phase 1: Pre-Migration Validation**
1. Check MLMD connectivity
2. Verify MySQL connectivity and disk space
3. Validate all MLMD contexts have corresponding `run_details` rows

**Phase 2: Schema Migration (Synchronous, ~5-10 seconds)**
- GORM `AutoMigrate` creates/alters `tasks`, `artifacts`, `artifact_tasks` tables
- **Blocks API server startup** — the server does not serve traffic until this completes
- This is intentional: without the schema, the API server cannot write to the new tables

**Phase 3: Data Migration (Background, non-blocking)**
1. Migrate Executions → Tasks (deterministic UUIDs, `INSERT IGNORE`)
2. Migrate Artifacts → Artifacts (split metrics, register in `mlmd_id_map`)
3. Migrate Events → ArtifactTasks (last — depends on both tasks and artifacts)
4. Resolve parent task references (second pass using `mlmd_id_map`)

**Phase 4: Post-Migration Validation**
- Entity count comparison
- TaskType distribution check
- Foreign key integrity validation
- Cache fingerprint verification (only COMPLETE/CACHED)

### Validation

**Pre-Migration:**
- ✅ MLMD service reachable
- ✅ All MLMD contexts exist in `run_details`
- ✅ Sufficient disk space (~2x MLMD database size)

**Post-Migration:**
- ✅ MLMD Executions count ≈ KFP Tasks count
- ✅ MLMD Artifacts count ≤ KFP Artifacts count (metrics split increases count)
- ✅ MLMD Events count = KFP ArtifactTasks count
- ✅ TaskType distribution reasonable (has ROOT, RUNTIME, etc.)
- ✅ All tasks reference valid runs (FK integrity)
- ✅ Cache fingerprints only on SUCCEEDED/CACHED tasks

### Rollback Support

**Strategy:** Keep MLMD running for 30-90 days after migration completes.

**Rollback Procedure:**
1. **Backup database** (before any destructive operations)   
2. Stop API server
3. Export new runs created post-migration (if any)
4. Delete migrated data:
   ```sql
   -- artifact_tasks and artifacts are NEW tables (all data is migrated), so truncate entirely
   DELETE FROM artifact_tasks;
   DELETE FROM artifacts;

   -- tasks is a MODIFIED existing table — only delete migrated rows, preserve any
   -- rows that existed before migration (identified via mlmd_id_map)
   DELETE FROM tasks WHERE UUID IN (
     SELECT kfp_uuid FROM mlmd_id_map WHERE mlmd_entity_type = 'execution'
   );
   ```
4. Revert to previous API server version
5. Restart with MLMD enabled

### Error Handling

**Non-Fatal Errors** (log warning, continue):
- Individual execution transformation failure
- Missing optional properties

**Fatal Errors** (fail migration, set status to FAILED):
- Database connection loss
- MLMD gRPC connection failure
- Disk space exhaustion
- Schema mismatch

**Recovery:**
- Set `RETRY_FAILED_MIGRATION=true` and restart API server
- Or run standalone CLI tool with `--resume` flag

## Implementation Plan

- Implement the shared migration engine (`backend/src/apiserver/migration/`) — manager, transformers, checkpoint, validation
- Implement the standalone CLI tool (`backend/cmd/mlmd-migrate/main.go`)
- Integrate migration into API server startup (`backend/src/apiserver/main.go`) with env var configuration and background goroutine
- Unit tests for entity transformations, checkpoint logic, and idempotency
- Integration tests against a test MLMD database (full migration, delta migration, resume after crash)
- Multi-replica coordination tests (verify only one replica migrates at a time)

## References

- [Main MLMD Removal Proposal](./README.md)
- [Design Details](./design-details.md)
- [Schema Changes](./schema_changes.sql)
- [Test Plan](./test_plan.md)
