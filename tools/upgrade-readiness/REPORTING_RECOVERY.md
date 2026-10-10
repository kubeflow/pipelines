# Persistence reporting recovery acceptance

This is a **mutating disposable Kind test**, never a production readiness command.
It requires the existing `kind-kfp-readiness` owned fixture and token, source
2.17.2, and a candidate containing the persistence retry backport (#14715) and
legacy schedule adoption (#14741). The parent upgrade workflow must install the
same candidate in the API server and persistence agent.

Before upgrade, run:

```bash
python3 tools/upgrade-readiness/live_reporting_recovery.py prepare \
  --fixture-state "$STATE/state.json" --state-dir "$STATE/reporting" \
  --endpoint http://127.0.0.1:8888 --token-file "$STATE/token"
```

This creates raw-Argo V1 immediate and recurring runs using the real source API.
The outage and deletion scenarios each receive an immediate/recurring pair.
All four Workflows are suspended; each job is disabled after its first persisted run. Source ownership collection
must observe stored identity and experiment namespace for all four runs.
No database write or synthetic report seeds the source state. Keep the Workflows
suspended through deployment. Reopen forwarding and refresh the token after the
upgrade, then run the same command with `recover` instead of `prepare`.

The probe removes only Workflow `get` from the API server's copied fixture Role
(`fixture-ml-pipeline-infrastructure` in `kfp-readiness-test`) and
verifies effective denial while the persistence agent retains get/list/watch.
After resuming the original Workflows, it requires successful Kubernetes
execution, nonterminal API state, and a transient persistence-worker failure for
each Workflow. It restores permissions and requires those exact run IDs to reach
`SUCCEEDED` without restarting the persistence worker. UID, experiment and
recurring ownership must stay unchanged. Each wait is bounded.

The normal exception path restores the rules in `finally`. The enclosing CI job
must also restore `rules` from `reporting-rbac-restore.json` in its unconditional
cleanup step, because cancellation or process termination can bypass Python
cleanup. Invoke the helper's `restore` phase with `--fixture-state` and
`--state-dir` (no endpoint or token required). It refuses to overwrite unexpected
concurrent RBAC edits. That file is private operational state, not an evidence artifact.
Upload only `reporting-source.json`, `reporting-blocked.json` and
`reporting-recovered.json`, `reporting-deleted.json` and
`reporting-ownership.json`, and optional `reporting-proxy-diagnostics.json`. The
last file records only rollout status, allowlisted failure reasons and probe
status codes; it is diagnostic evidence, not acceptance. These files contain synthetic resource identifiers and no
raw Workflow specifications, tokens or logs.

After outage recovery, deploy the test-only report proxy configured with exact
`deletion_runs` targets from `reporting-source.json`. Route the persistence
agent's API connection through it and wait for that deployment to become ready.
The proxy forwards authentication unchanged. Run the `delete` phase with the
same arguments plus `--proxy-endpoint http://127.0.0.1:PROXY_PORT`. This resumes
the second source pair. The proxy captures each actual worker terminal report,
deletes only its configured immutable Workflow UID, waits for NotFound, then
forwards the unmodified report. Acceptance requires both proxy deletion evidence
and successful API terminal state with original ownership, with no worker
restart during this phase. A NotFound response after final-state persistence is
allowed only when the API confirms `SUCCEEDED`.

## What this proves, and what it does not

A successful live run proves eventual catch-up after a reversible API-server
lookup authorization failure through the actual persistence worker. It does
not prove every network failure mode or arbitrary deletion recovery.

Deletion has two distinct cases:

- The worker already holds a terminal Workflow snapshot when the API server's
  live lookup returns NotFound: the server can validate the persisted immutable
  identity and commit that terminal snapshot. The live deletion phase covers the source immediate and recurring pair.
  Resource/server regression tests must additionally cover immediate, recurring and legacy identities, including forged,
  cross-namespace and replacement-UID rejection. A replay of a saved report is
  API regression evidence, not evidence of worker retry behavior.
- The Workflow disappears before the worker captures its terminal snapshot (or
  before a retry can read it): the worker has no terminal event to reconstruct.
  Its missing-object path is permanent. This fixture deliberately does not
  claim successful recovery in that case. Retain Workflows until final-state
  persistence is confirmed; detection/recovery guidance must describe stranded
  historical runs separately.

A live result must be linked from release acceptance; passing the local Python
unit tests alone does not establish cluster recovery.
