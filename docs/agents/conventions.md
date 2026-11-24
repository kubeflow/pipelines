# Conventions

## Reuse and design

- Search for and reuse existing helpers before adding code. Refactor an existing implementation if that avoids duplication.
- Use descriptive names; avoid unexplained abbreviations and single-letter names.
- Keep `ResourceManager` focused on run/job persistence and lifecycle coordination.
- Keep execution-engine behavior behind compiler or execution abstractions. Shared code must remain engine-neutral; do not downcast to `*util.Workflow` in shared layers.
- Put reusable interfaces in neutral packages and use natural domain types. Preserve documented ownership and field-wise override behavior.

## Tests

- Add a unit test for non-trivial functions, methods, and exported APIs. Add coverage when changed behavior needs it.
- Run the relevant tests before submitting. Document unrelated pre-existing failures in the PR description.

## Comments and documentation

- Comment only non-obvious constraints, invariants, workarounds, or surprising behavior.
- Keep comments short. Do not narrate implementation, current-PR history, or obvious test setup.
- Error messages must state the problem and corrective action.
- Write concise GoDoc for exported Go APIs. Python SDK public docstrings are user-facing Sphinx documentation.

## Database query guidelines (backend storage layer)

When writing or modifying database queries in the backend storage layer (`backend/src/apiserver/storage/`):

### Preferred approach: Use GORM for new features

- **GORM handles SQL dialects automatically**: GORM correctly quotes identifiers and generates dialect-specific SQL for both MySQL and PostgreSQL.
- **Use GORM for**: Simple CRUD operations, basic queries, and whenever GORM supports the required functionality.
- **Example**:
  ```go
  db.Model(&model.Experiment{}).
      Where("uuid = ?", uuid).
      Update("storage_state", model.StorageStateArchived.ToString())
  ```
  `Where("uuid = ?", uuid)` works even though `uuid` is lowercase because GORM matches raw condition strings against the model's schema fields and resolves it to the real column `"UUID"`. Squirrel below does not do this automatic resolution — every identifier must be quoted explicitly.

### Fallback: Squirrel + DBDialect for complex queries

When GORM cannot express the query (e.g., complex JOINs, subqueries, CTEs, UPSERT operations):

- **Use Squirrel query builder** with the `dialect.DBDialect` helper (stores expose this as the `dbDialect` field).
- **CRITICAL**: All table and column names MUST be quoted using `dialect.QuoteIdentifier()`.
- **Why quoting is required**: KFP uses lowercase table names (e.g. `experiments`) and CamelCase column names (e.g. `ExperimentUUID`) — a legacy design choice. Without quoting:
  - MySQL treats `ExperimentUUID` as `experimentuuid` (case-insensitive)
  - PostgreSQL treats `ExperimentUUID` as `experimentuuid` (lowercased), breaking queries
- **Example** (adapted from the real `ExperimentStore.UnarchiveExperiment` in `backend/src/apiserver/storage/experiment_store.go`):

  ```go
  func (s *ExperimentStore) ArchiveExperiment(id string) error {
      q := s.dbDialect.QuoteIdentifier
      qb := s.dbDialect.QueryBuilder()

      sql, args, err := qb.
          Update(q("experiments")).
          SetMap(sq.Eq{
              q("StorageState"): model.StorageStateArchived.ToString(),
          }).
          Where(sq.Eq{q("UUID"): id}).
          ToSql()
      if err != nil {
          return err
      }

      _, err = s.db.Exec(sql, args...)
      return err
  }
  ```

  `SetMap` is used instead of `Set` to match this codebase's established convention — every real UPDATE in `backend/src/apiserver/storage/` uses `SetMap`, even for single-column updates.

## Online data migrations

- Before putting work on the API startup path, check whether its cost grows
  with retained history. When requests can serve correctly with legacy rows,
  run the repair in a cancellable background worker. Document any dependency
  that requires the repair to complete before readiness.
- Find candidates with nonlocking reads that select only keys and required
  values. Update by indexed keys in bounded transactions, recheck the repair
  predicate at write time, and commit before reading the next batch. Inspect the
  write plan on the target engine. A zero affected-row count can still hide a
  locking table scan. For tables with large manifests, measure candidate
  discovery on wide rows and verify the write bound with production logging
  settings. MySQL FULL row images can log both copies of a manifest when only
  a key changes.
- Coordinate replicas with the database's existing migration locking pattern.
  Make committed batches safe to repeat, record completion after the last
  batch, and bound lock waits. On shutdown, interrupt the active database
  statement and release session-scoped locks before reusing a connection.
  Expose progress and document recovery for an interrupted rollout.
- For migrations of a live table, exercise the deployed database engine and
  isolation level in CI. Hold a batch open and require representative writes
  from a separate connection to finish within a deadline before releasing it.
  Cover empty and partially migrated tables, interruption, and replica
  contention. For startup changes, verify new-pod readiness and old-replica
  writes during a mixed-version upgrade. Record failing baseline and passing
  candidate results for a locking regression.
- In the PR, link the CI test that exercises the changed operation and identify
  its overlapping writer and timeout. Reviewers should check that the test runs
  on the PR's target branch with the database service enabled. A green package
  can include a skipped integration test.

## Commits

- Sign commits with `git commit -s`.
- Do not add AI agents as commit co-authors.
- Follow [`CONTRIBUTING.md`](../../CONTRIBUTING.md) for DCO and PR conventions.
