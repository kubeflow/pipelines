# Recover run creation during a KFP 2.18.0 MySQL upgrade

KFP 2.18.0 runs a pipeline reference backfill before API readiness. On a large
InnoDB database its transaction can block run creation and status reports from
the previous replicas. A longer startup probe allows the transaction to hold
its locks longer. Rolling the deployment back can restore healthy pods while the
database still has an update in flight.

## Restore writes

Stop the 2.18.0 rollout and prevent those replicas from restarting the backfill
while you recover. In the KFP MySQL database, inspect active transactions with
an account that can see the KFP sessions.

```sql
SELECT t.trx_mysql_thread_id, p.USER, p.DB, t.trx_started, t.trx_state,
       t.trx_rows_locked, t.trx_query
FROM information_schema.innodb_trx AS t
JOIN information_schema.processlist AS p ON p.ID = t.trx_mysql_thread_id
WHERE p.DB = DATABASE();
```

Match the user and query to the KFP backfill. Look for a long-running update
of `run_details` or `jobs` setting `PipelineId` or `PipelineVersionId` from
a subquery. After confirming the owning session, run
`KILL CONNECTION <trx_mysql_thread_id>` with the displayed numeric ID.
Terminating the connection rolls back the entire transaction. Monitor
`innodb_trx` until rollback finishes and confirm that run creation and status
reporting recover. A replacement API image with the background backfill can then
resume the pending migration. The manual preparation below is for operators who
need to deploy the original 2.18.0 image.

## Prepare a large database for the original 2.18.0 upgrade

In a maintenance window, pause writers and keep 2.18.0 API servers stopped.
Use the KFP database after its run and job reference columns exist. The six
plain reads below list the primary keys and values that the migration would
repair. Apply each result set before running the next query for that table.
A newly filled version ID can supply a previously missing pipeline ID.

```sql
SELECT r.UUID, rr.ReferenceUUID AS repair_value
FROM resource_references rr JOIN run_details r ON r.UUID = rr.ResourceUUID
WHERE rr.ResourceType = 'Run' AND rr.ReferenceType = 'pipeline'
  AND rr.ReferenceUUID <> '' AND (r.PipelineId IS NULL OR r.PipelineId = '');

SELECT r.UUID, rr.ReferenceUUID AS repair_value
FROM resource_references rr JOIN run_details r ON r.UUID = rr.ResourceUUID
WHERE rr.ResourceType = 'Run' AND rr.ReferenceType = 'PipelineVersion'
  AND rr.ReferenceUUID <> '' AND (r.PipelineVersionId IS NULL OR r.PipelineVersionId = '');

SELECT r.UUID, pv.PipelineId AS repair_value
FROM run_details r JOIN pipeline_versions pv ON pv.UUID = r.PipelineVersionId
WHERE r.PipelineVersionId <> '' AND pv.PipelineId <> ''
  AND (r.PipelineId IS NULL OR r.PipelineId = '');

SELECT j.UUID, rr.ReferenceUUID AS repair_value
FROM resource_references rr JOIN jobs j ON j.UUID = rr.ResourceUUID
WHERE rr.ResourceType = 'Job' AND rr.ReferenceType = 'pipeline'
  AND rr.ReferenceUUID <> '' AND (j.PipelineId IS NULL OR j.PipelineId = '');

SELECT j.UUID, rr.ReferenceUUID AS repair_value
FROM resource_references rr JOIN jobs j ON j.UUID = rr.ResourceUUID
WHERE rr.ResourceType = 'Job' AND rr.ReferenceType = 'PipelineVersion'
  AND rr.ReferenceUUID <> '' AND (j.PipelineVersionId IS NULL OR j.PipelineVersionId = '');

SELECT j.UUID, pv.PipelineId AS repair_value
FROM jobs j JOIN pipeline_versions pv ON pv.UUID = j.PipelineVersionId
WHERE j.PipelineVersionId <> '' AND pv.PipelineId <> ''
  AND (j.PipelineId IS NULL OR j.PipelineId = '');
```

Use parameterized, primary-key updates in autocommit mode. For example,
bind the first query's repair value and UUID in this order.

```sql
UPDATE run_details SET PipelineId = ?
WHERE UUID = ? AND (PipelineId IS NULL OR PipelineId = '');
```

For the other queries, substitute the corresponding table and destination
column in both the assignment and the empty-value check. Repeat all six
reads. Once they all return zero rows and writers are still paused, record
completion. A database upgrading from an earlier release may need the marker
table created first.

```sql
CREATE TABLE IF NOT EXISTS migration_statuses (
  Name varchar(64) NOT NULL PRIMARY KEY,
  AppliedAtInSec bigint NOT NULL,
  ClaimToken varchar(64) NOT NULL
) ENGINE=InnoDB;

INSERT INTO migration_statuses (Name, AppliedAtInSec, ClaimToken)
SELECT 'backfill_pipeline_refs', UNIX_TIMESTAMP(), UUID()
WHERE NOT EXISTS (
  SELECT 1 FROM migration_statuses WHERE Name = 'backfill_pipeline_refs'
);

SELECT Name, AppliedAtInSec FROM migration_statuses
WHERE Name = 'backfill_pipeline_refs';
```

Resume the rollout after verifying the marker. KFP 2.18.0 checks its name at
startup and skips the completed migration.
