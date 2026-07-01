# Page Token Sort Fields

- Location: `backend/src/apiserver/list/`
- Scope: how list/pagination page tokens encode sort fields, including
  validation, legacy migration, and prefix/filter-key handling in
  `token.unmarshal()`.

For version history (A → B → C) and the defects that motivated the current
design, see [PR #13547](https://github.com/kubeflow/pipelines/pull/13547).

## How sorting works

Two kinds of sort exist:

- **Regular column** (e.g. `created_at`): the column lives on the model's
  own table. Sort directly: `ORDER BY CreatedAtInSec`.
- **Metric** (e.g. `metric:accuracy`): metric values live in the
  `run_metrics` long table — one row per (run, metric). To sort by a metric
  you must first **pivot** the long table into one value-per-run column,
  then order by the pivot alias:

  ```sql
  SELECT ...,
         MAX(CASE WHEN rm.Name = ? THEN rm.NumberValue END) AS sort_metric_value
         --                       ^ bind param = the metric name ("accuracy")
  FROM ... LEFT JOIN run_metrics rm ON ...
  GROUP BY run.UUID
  ORDER BY sort_metric_value          -- the pivoted alias
  ```

After the pivot, the metric column behaves exactly like a regular column —
the `ORDER BY` clause takes one column name, always from `SortByFieldName`.
The metric *name* is a separate concern: it selects which metric row to
pivot, and only flows as a **bind parameter**, never as a SQL identifier.

| Concern | Token field | Needed when |
|---|---|---|
| "order by which column?" | `SortByFieldName` | Both kinds of sort |
| "pivot which metric from the long table?" | `SortByMetricName` | Metric sort only |

Metric sort is a superset of regular sort: same `ORDER BY`, plus one extra
"pivot the long table" step.

## Constants and validators

| Category | Question it answers | Implementation | Applies to |
|---|---|---|---|
| **Constant** | Fixed pivot alias for `ORDER BY` | `model.MetricSortSQLAlias = "sort_metric_value"` — hardcoded, never user input | Metric sort SQL alias |
| **Field validator** | "Is this a real column on this model?" | `IsRegularField(name)` — checks against `APIToModelFieldMap()` values via `isRegularFieldIn()`. For table names, `GetModelName()` is used directly | `SortByFieldName` (non-alias), `KeyFieldName`, filter key segments |
| **Metric validator** | "Does this listable support metric sorting for this name?" | `GetField("metric:" + name)` — returns `ok` with the fixed alias only for models that support metric sort (currently only `Run`) | Legacy token migration (confirms before promoting) |
| **Bind value validator** | "Is this value safe as a bind parameter?" | `validateBindValue(name)` — denylist: length bound (`maxBindValueLength = 256`) + reject non-printable/control chars. Permissive by design since parameterization prevents injection regardless of character shape | `SortByMetricName` |

**Rule for any new field**: identify which question it answers, then pick
the matching validator. Never cross them — `IsRegularField` returning
`false` for a bind value is expected, not a security signal.

## Unmarshal and legacy token migration

`token.unmarshal(listable Listable, pageToken string)` decodes a
base64+JSON token, detects its version via `detectTokenVersion()`, and
migrates older layouts into the current structure in memory (nothing is
re-persisted). Called by `NewOptionsFromToken` (page 2+).

### Version detection

| Marker | Detected version |
|---|---|
| `SortByMetricName != ""` | Current — no migration needed |
| `SortBySQLColumn == MetricSortSQLAlias` and `SortByFieldName != ""` | Legacy B |
| Neither marker present | Legacy A |

### Migration actions

**Current tokens** — no action.

**Legacy B → current (metric sort)**:

1. Confirm the listable supports metric sorting: `GetField("metric:" + SortByFieldName)` must return `ok`.
2. `SortByFieldName` (held the raw metric name) → `SortByMetricName`.
3. `SortBySQLColumn` (held the alias) → `SortByFieldName`.
4. If the metric validator rejects → `InvalidInputError`.

**Legacy A → current**:

1. `IsRegularField(SortByFieldName)` — if `true`, it's a real column,
   no migration needed (already a valid regular sort).
2. If `false` → `GetField("metric:" + SortByFieldName)` confirms metric
   support before promoting (same field shift as legacy B).
3. Both reject → `InvalidInputError`.

`SortBySQLColumn` is consumed and cleared in all paths so it never
propagates past `unmarshal()`.

### Post-migration validation

After migration, `unmarshal()` validates all fields against the listable:

- `KeyFieldName` — `IsRegularField` check.
- `SortByFieldName` — `IsRegularField` check (unless it equals `MetricSortSQLAlias`).
- `SortByMetricName` — `validateBindValue` hygiene check.
- `ModelName` — must equal `GetModelName()`.

## Prefix and filter key handling

- **`KeyFieldPrefix` / `SortByFieldPrefix`** are table aliases (e.g.
  `"runs."`). `unmarshal()` ignores whatever the token claims and
  recomputes both from the listable (`GetKeyFieldPrefix()` /
  `GetSortByFieldPrefix(SortByFieldName)`). The token's claimed value
  never reaches SQL.
- **Filter keys** (e.g. `"pipelines.Name"`) are validated per-segment via
  `Filter.ValidateKeys(func(segment string, isLast bool) error)` — the
  final segment (column) is checked with `IsRegularField`, earlier segments
  (table qualifier) against `GetModelName()`.
