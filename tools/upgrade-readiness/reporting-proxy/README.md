# Disposable reporting race proxy

This test-only binary belongs only in the isolated `kind-kfp-readiness` cluster.
It is not a production proxy or an authenticated public service. Deploy it in the
same namespace as `ml-pipeline`, and route only the fixture persistence agent to
its Service. Restore the agent's original upstream and remove the proxy after the
experiment, including on failure.

Build using the repository Go toolchain:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o reporting-proxy ./tools/upgrade-readiness/reporting-proxy
```

Required environment:

- `FIXTURE_NAMESPACE=kfp-readiness-test` (any other value is rejected).
- `FIXTURE_TARGETS_JSON`: one to ten objects containing `run_id`, `workflow_name`,
  and `workflow_uid`, captured from the fixture's suspended source runs.

Use a dedicated service account with only `get` and `delete` for the explicit
Workflow `resourceNames` in `kfp-readiness-test`. It does not need list/watch,
secrets, tokens, or cluster-wide access. Keep the Service internal and restrict
access to the disposable fixture. This proxy has no independently authenticated
RPC boundary; its deletion authority must never be granted over real workloads.

Port 8887 forwards ReportService workflow/schedule requests and RunService metrics
to `ml-pipeline:8887`. Other RPCs are unimplemented. Incoming metadata, including
the persistence agent's authorization, and the original request are forwarded;
no API-server identity is minted. Port 8888 serves `/healthz` from
`ml-pipeline:8888/healthz` for agent initialization and `/apis/v2beta1/reporting-fixture-evidence` with sanitized
per-target outcomes. It does not forward artifact HTTP requests; fixture workloads
must not depend on artifact metrics. The health endpoint is separate from evidence.

For a terminal Workflow report matching all configured identity fields, the proxy
holds that actual worker-captured request, deletes the exact Workflow with a UID
precondition, waits for Kubernetes NotFound, then forwards the unchanged request.
A missing Workflow before deletion, replacement UID, or deletion failure does not
count as successful injection. Upstream status codes are preserved. Evidence lists
`deleted`, `attempts`, `upstream_code`, and `deletion_failed` with the configured
identifiers. Require the corresponding API run to reach its correct terminal state
separately; an injected deletion alone is not acceptance.

This establishes the deletion-after-worker-capture race. It cannot establish
recovery when deletion removes the terminal snapshot before any worker captures it.
