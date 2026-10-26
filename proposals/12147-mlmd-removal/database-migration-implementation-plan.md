# MLMD database migration implementation plan

This plan covers upgrades from KFP 2.18 to KFP 3.0. The
[migration design](database-migration-design.md#implementation-plan) describes
how 2.18 metadata moves into the [MLMD-free runtime](design-details.md).

A KFP administrator initiates the upgrade during a planned maintenance window
by running the migration CLI or deploying the upgraded API server.

## PR 1 — Capture legacy data and define the expected result

PR 1 captures the KFP 2.18.0 database, reproduces the startup failure, and
defines initial migration expectations and cutover checks.

- **Fixture contract**
  - **Capture.** Use [KFP 2.18.0](../../backend/src/apiserver/migration/mlmd/testdata/release_218/README.md)
    to capture `run_details`, legacy tasks, the MLMD graph, and submitted
    Workflows. Include sparse IDs, long values, and representative states and
    pipeline sources.
  - **Assertions.** Check identities, joined fields, hierarchy, state, metrics,
    cache lineage, and event links.
  - **Legacy rows.** The 179 Workflow reports add no task rows. Saved manifests
    preserve Workflow history at snapshot granularity. The 12 cache rows resolve
    to existing execution identities.
  - **Expected counts.** 34 tasks, 26 artifacts after metric splits, and 26 links.
- **API expectations.** Use legacy `GetRun` responses and the captured MLMD graph
  to define expected reads for `ListRuns`, `GetRun`, `ListTasks`, `GetTask`,
  `ListArtifacts`, `GetArtifact`, and `ListArtifactTasks`. Reuse one fixture and
  cover distinct filters, projections, and relationship hydration. PRs 2 and 3
  exercise the endpoints on migrated data.
- **Startup reproducer.** Replay the populated 2.18.0 database, reproduce the old
  `tasks` schema’s `AutoMigrate` failure, and verify the source rows stay intact.
- **Administrator handoff.** Document writer ownership and shutdown evidence. Match
  Workflows to the installation by run ID and namespace. Ordinary container cache
  hits publish `CACHED` to MLMD. Some PVC driver operations write cache rows.
  [Shutdown procedure](database-migration-design.md#handling-in-flight-runs) ·
  [Drain checker](../../backend/src/apiserver/migration/mlmd/check_workflow_drain.py)

**Exit gate.** Source inventory, contract, and startup reproducer pass in CI.
Initial API expectations and the shutdown runbook are ready. Migration stays
disabled.

## PR 2 — Build the migration engine and CLI

- **Engine and CLI.** Read from the installation's MLMD store. Validate run
  ownership against `run_details` and reject legacy PostgreSQL before writes.
  Use one worker and follow the
  [resume and validation rules](database-migration-design.md#idempotency).
- **Fixture expectations.** Reuse the PR 1 capture. Update expected rows and
  API assertions for the [final mappings](database-migration-design.md#mlmd-entity-transformations).
- **Skip reporting.** Warn with the entity type, source ID, and reason for each
  unsupported record. Report aggregate counts and leave skipped records in
  MLMD. Fail on schema errors or unrecoverable infrastructure errors.
  Use the MLMD execution ID for skipped executions and log final copied and
  skipped counts. Keep resume and completion independent of logs and persistent
  skip records.
- **Typed cache provenance.** Add optional `PipelineTask.cached_task_id` and
  nullable `tasks.CachedTaskUUID`. Populate it during migration and native cache
  hits. Update API conversions, generated clients, and fixture expectations.
  Test both writers.
- **Cache backfill.** Implement the [fingerprint rules](database-migration-design.md#cache-fingerprints).
- **Disposable upgrade tests.** Replay the fixture and verify native API reads.
  Compare clean and resumed results with sparse IDs, metric splits, and missing
  parents. Check interior gaps, conflicting fields, skip reports, fatal errors,
  lock contention, legacy PostgreSQL rejection before writes, and
  `CANCELED` → `FAILED`. Verify cache cutoff reuse, native key parity, and
  backfill skips.

**Exit gate.** The CLI migrates and resumes a disposable upgrade, schema checks
pass, and migrated fingerprints follow the approved lookup rules. Production
CLI and startup migration wait for PR 3.

## PR 3 — Enable production migration through the CLI and API startup

- **Startup.** Classify the database before schema reconciliation. Reject legacy
  PostgreSQL before writes. Verify fresh and already-native PostgreSQL start
  normally without MLMD. Verify incomplete MySQL upgrades still require MLMD.
- **Enforcement.** Apply the writer checks to both entry points before
  Phase 2/3. Keep normal traffic blocked until completion.
- **Preflight CLI.** Add the read-only `validate` command for administrators to
  run before scaling down the old API. Reuse the PR 2 connectivity and schema
  checks. Test active-run and Workflow output and nonzero exits.
- **Upgrade CI.** Test CLI-first and startup-only upgrades from the same 2.18.0
  capture. For CLI-first, restore both schemas from that dump and run the built
  CLI against the restored KFP schema and live 2.18.0 MLMD endpoint. In both
  paths, verify run, task, artifact, and lineage reads. Test contention,
  restarts, skips, final reports, active-run refusal, writer shutdown, and cache.
- **Rollback test.** After upgrade, restore the pre-upgrade KFP backup. Start the
  2.18.0 API server with the preserved MLMD source and verify run and task reads.
- **Administrator guide.** Package the CLI and document maintenance, backup,
  and recovery. Reuse the installation's MLMD endpoint on retry and review
  skip diagnostics before removing MLMD.

**Exit gate.** Both paths pass upgrade and API checks, rollback works, and a
completed installation restarts without MLMD.

## Design feasibility checks before implementation

| Check | Required proof |
|---|---|
| Resume and idempotency | Verify which event IDs the MLMD read path exposes and how equivalent links replay. Prove ordered commits leave a complete prefix for binary search, including atomic metric splits. |
| Schema | Qualify copy and rename under the [nonblocking database policy](../../docs/agents/database-migrations.md) or agree on a nonblocking design before production use. |
| Drain | Prove both entry points reject active KFP runs, matching Workflows, missing final reports, and reachable old writers. The capture includes `RUNNING` MLMD DAG executions in completed runs. |
| Cache | Reproduce the 2.18 hash from captured inputs and verify the backfilled key matches native lookup. |
