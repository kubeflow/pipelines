# Live schedule migration and target-policy acceptance

The read-only baseline and observation helpers inspect **prepared schedules in an
isolated multi-user test installation**. The separate `provision_live_schedules.py`
CI helper creates fixtures and changes their activation state; it is never invoked
by the operator readiness scan.
No passing live-cluster result has yet been recorded for this fixture protocol.

The release acceptance sequence is: prepare on 2.17.2, preserve scanner reports and
baselines, upgrade to the exact candidate, and verify that the old schedules
are rejected for missing trusted scheduling state. Recreate the controlled fixtures
through the target API from the same reviewed local pipeline, then compare their
main-account behavior with checks derived from the recreated target records.
The source scanner correctly reports embedded workflows as unresolved. This lane
therefore does **not** validate pre-upgrade scanner predictions; it validates the
legacy migration boundary and subsequent target behavior. The `readiness-schedules`
job contains scaffolding for a separate multi-user fixture lane alongside the
single-user persistence tests. The schedule lane requires either the repository variable
`KFP_218_READINESS_SCHEDULES=enabled` or the explicit manual dispatch input
`run_readiness_schedules=true`. Enable it only after the
scheduling prerequisites land and activation has been reviewed. Preflight rejects
a candidate missing those prerequisites. The reviewed manual dispatch input permits
a single isolated acceptance run without changing the repository-wide gate. Integration and an actual passing candidate run remain open in #14421.

## Fixture contract

Use a fresh test namespace, a short periodic schedule interval (for example ten
seconds), and a small known-working pipeline. Prove the
pipeline and controller work on the source version before collecting the final
baseline. Disable the fixture schedules through the API before the upgrade. First re-enable
them only to establish the expected legacy migration rejection, then disable them
again and recreate replacements through the API from the reviewed local fixture.
Do not edit only their CRs, because the API record is authoritative. The fixture owner must be allowed to create/enable every account
under test. The controller must retain run creation and referenced-pipeline access. Keep the
fixtures exclusive to the test: do not manually submit runs with their recurring
run IDs during observation. Run association alone does not identify the submitting
process.

For target **enforce** mode prepare three distinct recurring runs:

| Scenario | Account and target grants | Predicted status | Observed requirement |
| --- | --- | --- | --- |
| Default control | Omitted account resolving to the configured default | `no_issue_detected` | New run with that account reaches `SUCCEEDED` |
| Scoped grant | Explicit custom account in allowlist; controller has `use` only for that named account | `no_issue_detected` | New run with that account reaches `SUCCEEDED` |
| Denied custom | Explicit custom account in allowlist; controller lacks its `use` grant | `policy_rejection` | Fresh correlated named-account rejection Event and no new run for the whole observation window |

Run **audit** as a separate phase: capture a new prediction report and baseline
with target mode `audit`, apply that mode in the isolated installation, and expect
the denied-custom scenario to produce a run with prediction `operational_impact`.
The verifier checks that audit permits a successful run. Audit-log emission is a
separate bounded fixture check; logs currently identify namespace/account/operation,
not a run ID or schedule UID. Keep a default control. Never change enforcement on an
operator's installation for this test.

Verify candidate image revisions, actual authenticated controller identity,
allowlist, compiler patch, independent workflow-identity mode and complete target
RBAC before interpreting results. Template/plugin identities and broader workload
representativeness remain outside this main-account fixture. Successful completion
of these small fixtures is required but cannot stand in for representative pipelines.

The 2.17.2 embedded-workflow scheduling path does not persist the run API's
`service_account` field. Source preparation and completion therefore also verify
the live Workflow: exact API run-ID label and display-name match, namespace,
fresh creation time, schedule controller owner UID/name, and expected service
account. A conflicting API account still fails. Target verification continues to
require the API account directly. Failed source preparation and completion retain
sanitized workload counts, enumerated pod/workflow/node phases, and failed-container
exit codes and known reasons. At most six failed-container log tails are inspected
in memory, each limited to 64 KiB and 15 seconds, and only fixed error categories
(such as authorization, DNS, metadata, or object store) are retained. No raw
specifications, server error text, credentials, or logs are included. Diagnostic
categories are clues for investigation, not proof of a root cause or a passing run.

After source fixtures are disabled, completed, and captured, the CI restores the
scheduler and persistence-agent `NAMESPACE` environment entries to their canonical
downward-API form and waits for rollout. This prevents a fixture-only literal value
from conflicting with `valueFrom` during candidate manifest application. No fixture
schedule is active during this transition; the target phase restores its namespace
scope before enabling the legacy rejection checks.

## Legacy migration acceptance

The CI lane separately checks all three source schedules for fresh, UID-correlated
controller `FailedPrecondition` Events naming missing trusted scheduling state.
No new associated run may appear during the full observation window, including
a final collection at or after its deadline. Missing Events, transport failures,
and generic failures are inconclusive. This is an expected migration requirement,
not successful continuation of old schedules and not a main-account prediction.
Audit mode does not bypass missing trusted state; the fixture observes this boundary
in enforce mode, with audit non-bypass separately covered by backend policy behavior.

`legacy-migration.json` preserves the observed result. Original source reports and
baseline remain under `source-*` names. The legacy baseline verifies exact schedule
identities and existing run/Event IDs independently of unresolved source predictions. Recreation requires disabled original
identities and the same digest of the reviewed local compiled pipeline; it never
copies execution inputs from a source CR or API row. The provisioner retains
`legacy-state.json` and creates new disabled schedules. `check_fixture_policy.py`
then collects only the fixture namespace's persisted target recurring runs and
experiments, matches the new CR identities, and evaluates the unchanged account
policy model. Its report explicitly states `post_recreation_target_policy_check`
and `pre_upgrade_prediction_validated=false`. Unknown or incomplete target evidence
fails; source `unknown` findings remain intact. Fresh target baselines precede
functional observation. The fixture owner receives no access to system-namespace
recurring runs merely to satisfy evidence collection.

## Capture a functional baseline after recreation

Use the explicit target-policy report for the recreated fixtures. Create a local cases file with actual schedule names/UIDs and account
names (these are fixture selectors, not credentials):

```json
{
  "cases": [
    {
      "scenario": "default-control",
      "schedule_name": "replace-with-control-name",
      "schedule_uid": "replace-with-control-uid",
      "service_account": "pipeline-runner",
      "expected_prediction": "no_issue_detected",
      "expected_outcome": "run_succeeded"
    },
    {
      "scenario": "custom-denied",
      "schedule_name": "replace-with-denied-name",
      "schedule_uid": "replace-with-denied-uid",
      "service_account": "readiness-denied",
      "expected_prediction": "policy_rejection",
      "expected_outcome": "blocked"
    }
  ]
}
```

Add the scoped-grant case for full enforce coverage. The helpers validate selectors
and prediction-report agreement. Baseline capture also verifies current schedule
UID/name/namespace and reads all existing run IDs and Event counts. It writes only
sanitized identifiers/counts and the observation start time:

```bash
python3 tools/upgrade-readiness/capture_live_schedule_baseline.py \
  --context isolated-test --namespace readiness-test \
  --kfp-endpoint https://test.example/pipeline --kfp-token-file /secure/test-token \
  --cases cases.json --prediction-report readiness.json > baseline.json
```

The token needs KFP run/experiment reads in the fixture namespace. The Kubernetes
context needs ScheduledWorkflow and Event list access there. Neither helper reads
Secrets or controller logs. Failed baseline collection exits nonzero; never reuse
a stale/partial baseline after failure. Reports and baselines contain resource
identifiers and should remain access-controlled.

## Observe after upgrade, recreation, and activation

After upgrading and recreating the fixtures in the isolated installation, record a UTC activation timestamp
**before** enabling its fixtures (for example `2026-09-18T15:00:00Z`). Pass that
actual timestamp below. This excludes source-version runs that appeared between
baseline capture and upgrade; the baseline timestamp alone is insufficient.
After enabling the fixtures:

```bash
python3 tools/upgrade-readiness/live_schedule_check.py \
  --context isolated-test --namespace readiness-test \
  --kfp-endpoint https://test.example/pipeline --kfp-token-file /secure/test-token \
  --expectations baseline.json --prediction-report readiness.json \
  --not-before "$ACTIVATION_TIME" --require-run-success \
  --timeout-seconds 120 > observed.json
```

The observer permits 30–600 seconds and at most 20 cases. It collects final evidence
at or after the deadline, so an in-flight collection and the final collection can
both extend the observation window; individual HTTP/Kubernetes request budgets
still apply. Source transport byte,
request and response budgets also apply; use small fixture sets. It follows run
pagination, verifies recurring-run association and experiment namespace, and
excludes baseline IDs and pre-observation timestamps. It verifies the service
account on each new run. `--require-run-success` upgrades `run_created` expectations
to `run_succeeded`. Failed/canceled/skipped runs fail; still-running or unknown
states remain inconclusive at the deadline. The older `run_created` expectation
remains available for narrower diagnostics, but does not establish task completion.

A blocked result requires a Warning `Failed` Event from the scheduled workflow
controller with the exact involved schedule UID/name/namespace, a fresh timestamp
and increased count, and a named core `serviceaccounts/use` PermissionDenied
signature. Raw messages are inspected in memory and never included in the output.
Generic errors, old Events, absent permissions, truncated collection, changed
message formats and timeouts cannot establish a denial. A blocked case also needs
a successful positive control in the completion mode, and observation continues through the full window to
catch an unexpected run after an earlier rejection. The controller error signature
is source-derived and still needs confirmation against actual candidate Events.

Treat only a successful verifier exit as a pass for the selected firing checks.
Observer collection failures include only an allowlisted local reason, fixed collection
stage, elapsed time, completed collection count, and HTTP request/byte counters.
These diagnostics preserve a nonzero inconclusive result and never expose backend
responses or credentials.
All other exits require investigation; do not interpret an inconclusive observation
as a rejected workload or a successful upgrade. Retain candidate/source revisions,
predictions, baselines and sanitized results in CI artifacts. A release gate must
run both enforce and audit phases; documenting this protocol or running its mocked
unit tests does not satisfy live acceptance.

After disabling each phase, the CI script rechecks all fresh associated runs
against its activation/baseline. Every expected run must succeed, and a blocked
scenario must still have no run. It writes `source-completion.json`,
`enforce-completion.json` and `audit-completion.json`; a failed/canceled/skipped
run or drain deadline fails the phase.

The audit phase then invokes `verify_live_audit.py`. It requires the successful
three-scenario audit completion report and reads only the isolated API container
logs after activation, with a 16 MiB in-memory collection cap and 30-second
process budget. Oversized or incomplete collection fails closed. An exact main-account audit record for `kfp-readiness-test/readiness-denied` must be
present. Only a matching count and the limited evidence scope are retained in
`audit-emission.json`. Raw logs are neither printed nor saved. This establishes
emission in that isolated namespace/account/window, not per-run correlation or
metrics coverage; the backend producer does not include run/schedule identifiers.

## Disposable CI fixture tooling

`provision_live_schedules.py` requires the exact context `kind-kfp-readiness` and
`--allow-test-cluster-mutations`. Its `rbac` phase creates a fresh
`kfp-readiness-test` namespace with a unique ownership marker; it refuses an
existing namespace. The namespace also carries the standard
`app.kubernetes.io/part-of=kubeflow-profile` label required by the installed
SeaweedFS NetworkPolicy for cross-namespace artifact traffic. The fixture does not
remove or relax that policy. Later phases require matching private state and namespace
ownership. This is a guard against accidental use, not proof that a context name
points to a disposable cluster: create the dedicated Kind cluster first.

The fixture owner can use both custom accounts. The controller can use only
`readiness-granted`, with a namespaced `resourceNames` grant. Both custom accounts
are allowlisted so the denied case isolates the controller's RBAC check. Runner
permissions are cloned from the source installation's `pipeline-runner` Role.
The isolated setup also grants the API, scheduler, persistence agent and Argo
controller namespace-scoped workload access and gives the API cluster-scoped
TokenReview/SubjectAccessReview access. A separate fixture ClusterRole gives only
the persistence agent `report` on the synthetic `pipelines.kubeflow.org`
`workflows` and `scheduledworkflows` resources: both source and target ReportServer
authorize these requests without a namespace. Two exact SubjectAccessReviews
verify this permission before fixture preparation. No additional execution or
service-account-use permission is granted by that role. The fixture copies the test installation's
artifact Secret and launcher ConfigMap into the fresh namespace without printing
or writing their contents to reports. These mutations and Secret reads belong
only to this disposable CI provisioner; the operator scanner never performs them.
The `prepare` phase creates three disabled recurring runs; `enable` records an
activation timestamp before API requests, and `disable` stops future submissions.
These operations use authenticated POSTs through a literal loopback port-forward.
Never supply production credentials or run these phases against an operator cluster.

`build_live_policy.py` combines a complete source RBAC snapshot with the candidate
manifests: matching objects are replaced and other source grants remain. It is
specific to this additive, non-pruning fixture upgrade and asserts complete RBAC
and an RBAC-only authorization model. It is not a general completeness detector.
Keep policy assembly separate from the read-only operator scan.

Local validation has exercised nine real Kubernetes SubjectAccessReviews: the
controller's named grant, its denied account, both owner grants, and rejection in
another namespace, API review permissions and controller workload access.
Namespace adoption was also rejected. These checks validate
fixture permissions only; they do not establish KFP schedule firing or upgrade
success. Those require the full candidate lane, including source run creation and
both target modes. The lane requires successful V2 fixture completion and bounded audit-log emission.
V1 schedules, permission revocation after a successful tick, template/plugin
identities, mixed-version rollouts, and representative production workloads remain
outside this lane's coverage.

The fixture owns and stops the actual kubectl port-forward process across API
rollouts. Startup requires that process to report its own loopback listener
before accepting health checks; an unrelated healthy listener cannot satisfy
readiness. Transport failures remain inconclusive and are not retried by the
observer.

The enforce window remains 180 seconds. The audit transition is observed for
600 seconds: the controller can retain up to 360 seconds of retry backoff after
an enforce-mode denial, followed by execution grace. Toggling a schedule does
not clear that queue delay. The lane does not restart the controller or promise
immediate execution after a policy change. Each observation collection uses a
fresh bounded HTTP client; request and byte limits still fail that collection
closed, and failure diagnostics report cumulative counters across collections.
The final collection still starts after the observation deadline.

If audit-emission verification fails, its report includes an allowlisted local
reason, validation/collection stage, elapsed time, and available collected-byte
and process-exit counters. Raw log lines and subprocess error text remain private;
collection limits and exact audit-record requirements remain mandatory.
