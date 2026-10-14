# Database changes without blocking upgrades

Read this guide before changing persistent model tags, schema reconciliation,
indexes, startup SQL, data backfills or database migration instructions. It applies
to fresh-install and upgrade paths, including changes generated indirectly by an ORM.

## Required availability boundary

**Never introduce or execute a production migration that requires stopping reads
or writes, or silently falls back to a table-copying or write-blocking DDL
algorithm.** Automatic execution, `AutoMigrate`, a background worker and a rolling
Kubernetes Deployment do not make database work nonblocking.

Online operations can still acquire brief metadata locks and consume I/O, disk
and CPU. Bound lock acquisition waits, measure concurrent request latency and
state those limits explicitly. Do not promise zero locking or zero downtime.
If no verified online path exists, stop that implementation and report the
constraint with a staged alternative; do not ship a blocking fallback or hide it
behind operator instructions to run the same SQL manually.

## Inspect the actual upgrade path

- Compare the deployed release's models and database catalog with the target
  branch. Inventory column types, widths, defaults, nullability, constraints and
  index definitions; do not assume the repository model describes every existing
  installation.
- Trace model tags through all `AutoMigrate` calls, relationship-driven migrations,
  explicit SQL and startup ordering. A legacy length-preflight list is not a
  complete inventory of ORM-generated changes. Removing a type tag can itself
  cause a conversion to the ORM's inferred type.
- Preserve existing stored types and values when adding an index. Never narrow a
  column merely to make it indexable. For example, shrinking MySQL
  `VARCHAR(255)` to `VARCHAR(191)` can require a table copy. Where appropriate, a
  nonunique prefix index can preserve the column, but queries must retain
  full-value comparison and tests must cover equal prefixes with different suffixes.
- Review readiness, startup probe budgets and mixed-version behavior. A migration
  completion marker or schedule handoff coordinates only the work it actually
  protects; it is not a database-wide schema lock.

## Design online changes and recovery

- Select and verify explicit algorithms for each supported engine/version. For
  MySQL secondary-index creation, require `ALGORITHM=INPLACE, LOCK=NONE` when
  supported; fail rather than retry without those constraints. `INPLACE` alone
  does not establish that writes continue or that no table rebuild occurs. For
  PostgreSQL, use `CREATE INDEX CONCURRENTLY` outside a transaction when applicable.
  These examples are not blanket permission to apply arbitrary DDL online.
- Reuse compatible indexes after checking their actual definition and validity.
  Do not drop or replace a same-named index merely because it exists. A failed
  concurrent PostgreSQL build can leave an invalid index: define a safe recovery
  path and test it. Never drop another process's active build as stale.
- Coordinate replicas around migration work using database-scoped ownership.
  Session locks, session timeouts and DDL must use the same physical connection.
  Release locks and restore settings on success, errors and cancellation; discard
  a connection if cleanup cannot safely return it to the pool.
- Bound lock waits, retries and total attempts. Do not leave DDL queued indefinitely:
  a waiting metadata lock can block later application writes. Avoid retry storms.
  Emit actionable progress/failure diagnostics without sensitive data; describe
  whether failure delays readiness, aborts startup or defers optional work.
- Use additive, staged schema changes when a direct conversion is unsafe. Keep old
  and new application versions compatible through rollout and rollback; defer
  destructive cleanup until old readers/writers have retired. Moving work out of
  startup is insufficient if that work still blocks the live database.
- Make large backfills bounded, restartable and idempotent. Use short transactions,
  stable batch traversal, durable progress and explicit handling of concurrent
  application writes. Do not add an unbounded whole-table update or long transaction
  to startup. Test interruption and resume without losing or overwriting newer data.

## Evidence required before declaring the upgrade safe

Use disposable instances of the supported production database engines. SQLite
and empty-schema tests cannot establish online upgrade behavior.

1. Seed representative prior-release schemas and populated tables, including long
   values and supported historical schema variants. Exercise the actual migration
   entrypoint, not just a hand-written approximation of its SQL.
2. Capture generated DDL and verify the required algorithm. Where avoiding a table
   rebuild is part of the claim, check engine evidence (for example, InnoDB table
   identity before/after) as well as successful execution.
3. Run concurrent reads and writes throughout the operation. Prove requests
   **complete while the migration is still active**, not only after it releases
   locks. Record latency, errors, row counts and value preservation; a zero-error
   result alone does not prove requests were not blocked.
4. Exercise long-running transactions/metadata-lock contention, timeouts,
   cancellation, interrupted builds, repeated startup and competing replicas.
   Verify bounded failure, cleanup and recovery, including invalid indexes.
5. Verify query correctness and useful index selection, mixed-version compatibility
   and rollback behavior where applicable. Add durable regression coverage to the
   existing database CI lanes so engine-specific tests do not silently skip there.

In the PR and upgrade guidance, identify the tested engines, source/target schema,
data volume, measured impact and remaining operational limits. Distinguish schema
DDL, reference/data backfills, scheduling adoption and optional maintenance work.
Evidence for one operation must never be presented as proof that the entire
release upgrade is nonblocking. If a required check cannot run, record that gap;
do not label the migration safe or ready based on inference alone.
