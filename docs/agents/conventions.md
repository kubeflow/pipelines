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
  db.Model(&models.Experiment{}).
      Where("uuid = ?", uuid).
      Update("storage_state", models.ExperimentStorageStateARCHIVED)
  ```

### Fallback: Squirrel + DBDialect for complex queries

When GORM cannot express the query (e.g., complex JOINs, subqueries, CTEs, UPSERT operations):

- **Use Squirrel query builder** with the `dialect.DBDialect` helper.
- **CRITICAL**: All table and column names MUST be quoted using `dialect.QuoteIdentifier()`.
- **Why quoting is required**: KFP uses CamelCase schema names (legacy design). Without quoting:
  - MySQL treats `ExperimentUUID` as `experimentuuid` (case-insensitive)
  - PostgreSQL treats `ExperimentUUID` as `experimentuuid` (lowercased), breaking queries
- **Example**:

  ```go
  func (s *ExperimentStore) ArchiveExperiment(id string) error {
      quotedTable := s.dialect.QuoteIdentifier("Experiments")
      quotedUUID := s.dialect.QuoteIdentifier("UUID")
      quotedState := s.dialect.QuoteIdentifier("StorageState")

      sql, args, err := s.dialect.QueryBuilder().
          Update(quotedTable).
          Set(quotedState, models.ExperimentStorageStateARCHIVED).
          Where(sq.Eq{quotedUUID: id}).
          ToSql()

      _, err = s.db.Exec(sql, args...)
      return err
  }
  ```

## Commits

- Sign commits with `git commit -s`.
- Do not add AI agents as commit co-authors.
- Follow [`CONTRIBUTING.md`](../../CONTRIBUTING.md) for DCO and PR conventions.
