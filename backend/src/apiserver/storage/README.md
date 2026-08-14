# Kubeflow Pipelines Storage Layer

This directory contains the storage layer for the Kubeflow Pipelines API Server, providing database access for pipelines, experiments, runs, jobs, and related entities.

## Database Support

KFP supports multiple database backends:

- **MySQL 8.x** - Primary production database (default)
- **PostgreSQL 14+** - Alternative production database
- **SQLite** - Used for unit testing only

## Architecture Overview

### Key Components

- **Store interfaces and implementations**: Each entity type (experiments, runs, pipelines, etc.) has a dedicated store with CRUD and query operations.
- **DBDialect**: Handles database-specific SQL generation (identifier quoting, placeholders, data types).
- **GORM models**: Database schema definitions in `backend/src/apiserver/model/`.
- **Query builders**: Mix of GORM (ORM) and Squirrel (SQL builder) for different query complexities.

### Database Schema Naming Convention

**Important**: KFP uses **CamelCase** table and column names (e.g., `Experiments`, `ExperimentUUID`) as a legacy design choice from the original MySQL implementation.

**Why this matters**: Different databases handle unquoted identifiers differently:

- **MySQL**: Case-insensitive on most platforms (macOS, Windows). `ExperimentUUID` and `experimentuuid` are treated the same.
- **PostgreSQL**: Case-sensitive and auto-lowercases unquoted identifiers. `ExperimentUUID` becomes `experimentuuid` when unquoted, causing query failures.

**Solution**: All identifiers must be quoted with dialect-specific quotes:
- MySQL: `` `ExperimentUUID` ``
- PostgreSQL: `"ExperimentUUID"`

## Writing Database Queries

### Approach 1: GORM (Preferred for new features)

**Use GORM when**:
- Performing simple CRUD operations
- Querying with basic WHERE conditions
- GORM supports the required functionality

**Advantages**:
- Automatically handles identifier quoting for all databases
- Type-safe query building
- Automatic placeholder handling (`?` for MySQL, `$1, $2` for PostgreSQL)
- Less boilerplate code

**Example**:

```go
// GORM handles quoting automatically
err := s.db.Model(&models.Experiment{}).
    Where("uuid = ?", experimentID).
    Update("storage_state", models.ExperimentStorageStateARCHIVED).
    Error

// GORM also handles complex conditions
var experiments []models.Experiment
err := s.db.
    Where("namespace = ?", namespace).
    Where("storage_state = ?", models.ExperimentStorageStateAVAILABLE).
    Order("created_at DESC").
    Limit(10).
    Find(&experiments).
    Error
```

### Approach 2: Squirrel + DBDialect (For complex queries)

**Use Squirrel + DBDialect when**:
- GORM cannot express the query (complex JOINs, subqueries, CTEs)
- Need database-specific features (UPSERT, window functions)
- Require fine-grained control over SQL generation

**CRITICAL**: You MUST manually quote all table and column names using the `DBDialect` helper.

**Example**:

```go
func (s *ExperimentStore) ArchiveExperimentWithSubquery(experimentID string) error {
    // Quote all identifiers
    experimentsTable := s.dialect.QuoteIdentifier("Experiments")
    runsTable := s.dialect.QuoteIdentifier("Runs")
    uuidCol := s.dialect.QuoteIdentifier("UUID")
    experimentUUIDCol := s.dialect.QuoteIdentifier("ExperimentUUID")
    storageStateCol := s.dialect.QuoteIdentifier("StorageState")

    // Build subquery to find runs
    subquery := squirrel.Select(experimentUUIDCol).
        From(runsTable).
        Where(sq.Eq{uuidCol: runID}).
        PlaceholderFormat(s.dialect.PlaceholderFormat())

    subquerySQL, subqueryArgs, err := subquery.ToSql()
    if err != nil {
        return fmt.Errorf("failed to build subquery: %w", err)
    }

    // Build main update query
    mainQuery := squirrel.Update(experimentsTable).
        Set(storageStateCol, models.ExperimentStorageStateARCHIVED).
        Where(sq.Expr(fmt.Sprintf("%s = (%s)", uuidCol, subquerySQL), subqueryArgs...)).
        PlaceholderFormat(s.dialect.PlaceholderFormat())

    sql, args, err := mainQuery.ToSql()
    if err != nil {
        return fmt.Errorf("failed to build main query: %w", err)
    }

    // Execute
    _, err = s.db.Exec(sql, args...)
    return err
}
```

### DBDialect API Reference

The `DBDialect` interface (in `dialect/`) provides database-specific helpers:

```go
type DBDialect interface {
    // Quote a single identifier (table or column name)
    QuoteIdentifier(name string) string
    // Example: MySQL returns `name`, PostgreSQL returns "name"

    // Quote multiple identifiers and join them
    QuoteMultipleIdentifiers(names []string, separator string) string
    // Example: ["col1", "col2"] with ", " → `col1`, `col2` (MySQL)
    //                                      → "col1", "col2" (PostgreSQL)

    // Placeholder format for Squirrel
    PlaceholderFormat() squirrel.PlaceholderFormat
    // MySQL: squirrel.Question (?)
    // PostgreSQL: squirrel.Dollar ($1, $2, $3)

    // Get dialect name
    Name() string
    // Returns "mysql" or "pgx"
}
```

**Common patterns**:

```go
// Pattern 1: Simple table/column quoting
quotedTable := dialect.QuoteIdentifier("Experiments")
quotedCol := dialect.QuoteIdentifier("UUID")

// Pattern 2: Dynamic column selection
cols := []string{"UUID", "Name", "Description"}
quotedCols := dialect.QuoteMultipleIdentifiers(cols, ", ")
query := fmt.Sprintf("SELECT %s FROM %s", quotedCols, quotedTable)

// Pattern 3: Building WHERE conditions with Squirrel
builder := squirrel.Select("*").
    From(dialect.QuoteIdentifier("Runs")).
    Where(sq.Eq{dialect.QuoteIdentifier("ExperimentUUID"): experimentID}).
    PlaceholderFormat(dialect.PlaceholderFormat())
```

## Testing

### Unit Tests

**Location**: `*_test.go` files in this directory

**Database**: Uses SQLite in-memory database via `NewFakeDB()` or `NewFakeDBOrFatal()`

**Purpose**: Fast, isolated tests for business logic and behavior

**Example**:

```go
func TestArchiveExperiment(t *testing.T) {
    db, dialect := NewFakeDBOrFatal()
    defer db.Close()

    store := NewExperimentStore(db, dialect, time.Now, uuid.New)

    // Test behavior, not SQL strings
    err := store.ArchiveExperiment(experimentID)
    assert.Nil(t, err)

    exp, _ := store.GetExperiment(experimentID)
    assert.Equal(t, models.ExperimentStorageStateARCHIVED, exp.StorageState)
}
```

### SQL Generation Tests

**When to add**: For complex queries that use Squirrel + DBDialect

**Purpose**: Verify dialect-specific SQL correctness

**Example pattern** (see `list_filters_test.go`):

```go
func TestArchiveExperimentSQL(t *testing.T) {
    dialects := []struct {
        name string
        d    dialect.DBDialect
    }{
        {name: "mysql", d: dialect.NewDBDialect("mysql")},
        {name: "pgx", d: dialect.NewDBDialect("pgx")},
    }

    for _, dl := range dialects {
        t.Run(dl.name, func(t *testing.T) {
            // Build query using the dialect
            query := buildArchiveQuery(dl.d, experimentID)
            sql, args, _ := query.ToSql()

            // Verify key SQL characteristics
            assert.Contains(t, normalizeSQL(sql), "update")
            assert.Contains(t, normalizeSQL(sql), "experiments")
            assert.Len(t, args, 2) // Expected number of placeholders

            // Optionally verify full SQL
            expectedSQL := map[string]string{
                "mysql": "UPDATE `Experiments` SET `StorageState` = ? WHERE `UUID` = ?",
                "pgx":   `UPDATE "Experiments" SET "StorageState" = $1 WHERE "UUID" = $2`,
            }
            assert.Equal(t, normalizeSQL(expectedSQL[dl.name]), normalizeSQL(sql))
        })
    }
}

// Helper to normalize SQL for comparison
func normalizeSQL(sql string) string {
    sql = strings.ReplaceAll(sql, "`", "")
    sql = strings.ReplaceAll(sql, `"`, "")
    sql = strings.ToLower(sql)
    return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(sql, " "))
}
```

### Integration Tests

**Location**: `backend/test/integration/` and `backend/test/v2/integration/`

**Database**: Real MySQL or PostgreSQL instances

**Execution**: Requires flags:
- `-runIntegrationTests=true`: Enable integration tests
- `-runPostgreSQLTests=true`: Run PostgreSQL-specific tests

**CI Coverage**: GitHub Actions workflows automatically run integration tests against both MySQL and PostgreSQL:
- `.github/workflows/api-server-tests.yml`: API tests with both MySQL and PostgreSQL (`db_type` matrix)
- `.github/workflows/legacy-v2-api-integration-tests-postgres.yml`: PostgreSQL integration tests
- `.github/workflows/integration-tests-v1-postgres.yml`: V1 API with PostgreSQL

## Common Pitfalls and Solutions

### Problem: Query works in MySQL but fails in PostgreSQL

**Symptom**: `ERROR: column "experimentuuid" does not exist`

**Cause**: Forgot to quote identifiers, PostgreSQL lowercased `ExperimentUUID` to `experimentuuid`

**Solution**: Always use `dialect.QuoteIdentifier()`:

```go
// ❌ WRONG - will break on PostgreSQL
query := squirrel.Select("*").From("Experiments")

// ✅ CORRECT - works on all databases
query := squirrel.Select("*").From(dialect.QuoteIdentifier("Experiments"))
```

### Problem: Placeholder mismatch error

**Symptom**: `ERROR: bind message supplies 2 parameters, but prepared statement "" requires 0`

**Cause**: Forgot to set placeholder format for Squirrel

**Solution**: Always use `PlaceholderFormat()`:

```go
// ❌ WRONG - defaults to ? placeholders (MySQL only)
query := squirrel.Update("Experiments").Set("Name", name)

// ✅ CORRECT - adapts to database dialect
query := squirrel.Update(quotedTable).
    Set(quotedCol, name).
    PlaceholderFormat(dialect.PlaceholderFormat())
```

### Problem: CamelCase vs snake_case confusion

**Current state**: Database schema uses CamelCase (e.g., `ExperimentUUID`)

**Long-term**: Migration to snake_case (e.g., `experiment_uuid`) would be cleaner but requires a major version change

**What to do now**: Continue using CamelCase with proper quoting. See `AGENTS.md` for the project's approach to this issue.

## Future Improvements

### Potential enhancements

1. **Schema migration to snake_case**: Would eliminate the need for identifier quoting (major breaking change)
2. **Linter or pre-commit hook**: Detect unquoted identifiers in raw SQL strings
3. **Increase GORM usage**: Migrate more complex queries from Squirrel to GORM where possible

### Migration considerations

Any schema naming change would require:
- Major version bump
- Migration scripts for existing databases
- Backward compatibility layer or deprecation period
- Updates to all store implementations

For now, the DBDialect + quoting approach provides a robust solution for multi-database support without breaking changes.

## Additional Resources

- **Project-wide guidelines**: See `AGENTS.md` in the repository root
- **DBDialect implementation**: `dialect/` subdirectory
- **GORM models**: `backend/src/apiserver/model/`
- **Test examples**: `list_filters_test.go`, `sql_dialect_util_test.go`
- **Integration tests**: `backend/test/integration/db_test.go`

## Questions or Issues

- Check existing tests for patterns and examples
- Refer to `AGENTS.md` for high-level guidance
- CI failures related to PostgreSQL usually indicate missing identifier quoting
- For new features, prefer GORM; fall back to Squirrel + DBDialect only when necessary
