# Workflow reporting: ownership and recovery

Run the read-only scan against the existing 2.17 installation before upgrading:

```bash
python3 tools/upgrade-readiness/readiness.py \
  --context my-cluster --system-namespace kubeflow --namespace team-a \
  --source-version 2.17.2 --kfp-endpoint https://pipelines.example \
  --kfp-token-file /secure/path/token --include-ownership --format json > readiness.json
```

Use the existing transport's HTTPS CA option if needed. The token needs permission
only to list runs/schedules and read the corresponding runs/experiments in selected
namespaces. No writes, database access, or Kubernetes Workflow reads are performed.
The V1 GetRun endpoint is required to read persisted runtime identity, including for
V2 runs; V2 Run responses do not expose that evidence. Runtime manifests are read
in memory but never emitted. Protect the report: it includes resource IDs.

## Interpret the results

The scan includes nonterminal/unknown-state runs and all schedules (including
currently disabled schedules). It reports namespace-reference, experiment-reference,
experiment-namespace, and stored Workflow identity findings separately. Namespace
references are API projections of persisted records, not direct database-column
verification. An absent namespace reference may be recoverable from the persisted
experiment; it is a review item, not proof of failure. Single-user legacy namespace
handling also differs from multi-user isolation.

`observed` means evidence was returned, **not** that a report will be accepted.
`review_required` identifies absent or inconsistent exposed evidence. `unknown`
includes denied/missing API responses, malformed records, unavailable endpoints,
and collection budgets. A 404 is not proof of physical absence: authorization or
source API behavior may hide a record. Schedule stored Workflow identity is not
exposed by this API and remains unknown. The scan never trusts a requested namespace,
live Workflow label, or incoming report as proof of ownership.

Collection is bounded and non-atomic; check `source.ownership_collection` for completed
scopes and repeat after correcting collection errors. All reports remain incomplete
(exit 2). Records with missing ownership may be excluded by namespace-scoped API
listing itself; an empty scan cannot prove there are no orphaned records. Compare
with your existing run/schedule inventory and escalate unexplained omissions. Do not
grant broad access simply to eliminate an unknown finding.

## Before and after the upgrade

1. Preserve original Workflows and persistence-agent evidence until runs have a
   terminal state recorded in KFP. Avoid TTL/GC/manual deletion during validation.
   Back up KFP metadata using your established procedure before upgrading.
2. Check API-server and persistence-agent logs for the affected run ID. Distinguish
   temporary live-lookup failures (`Unavailable` / “will retry”) from namespace,
   experiment, or stored-identity validation failures. Restore API connectivity and
   the intended scoped Kubernetes read permissions, then verify reporting catches
   up and terminal status is persisted. Successful health probes alone are insufficient.
3. A deleted Workflow is recoverable only when a trustworthy terminal snapshot is
   still available to the reporter and matches the persisted identity. Deletion
   before the worker captures that snapshot can lose the terminal event; retries
   cannot reconstruct it. Stored UID alone does not reconstruct terminal results.
4. For missing/inconsistent persisted ownership, retain evidence and seek maintainer
   diagnosis. Do not rewrite namespace, experiment, UID, or Workflow labels to make
   a report pass, and do not relax report authentication. There is no general
   supported in-place ownership repair command.
5. If recovery cannot be established, record the old run as unresolved outside KFP
   before any cleanup. Recreate a schedule or submit a replacement through the
   authorized API in the intended experiment only after assessing side effects,
   stopping duplicate submissions, and reconciling any still-running workload.
   Replacement creates new IDs and does not repair the original run's history.
   Use normal authorized lifecycle operations when they succeed; if ownership
   prevents them, escalate instead of bypassing checks or editing the database.

The release acceptance test must separately prove that valid existing runs recover
through a temporary lookup outage and that captured terminal reports survive the
supported deletion race. This scanner supplies pre-upgrade evidence, not that proof.
