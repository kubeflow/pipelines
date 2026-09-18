# Live schedule prediction acceptance

The read-only baseline and observation helpers inspect **prepared schedules in an
isolated multi-user test installation**. The separate `provision_live_schedules.py`
CI helper creates fixtures and changes their activation state; it is never invoked
by the operator readiness scan.
No passing live-cluster result has yet been recorded for this fixture protocol.

The release acceptance sequence is: prepare on 2.17.2, capture predictions and
baselines, upgrade to the exact candidate, enable the prepared schedules, then
compare actual controller behavior with the predictions. The opt-in `readiness-schedules` job in the upgrade workflow provides a separate
multi-user fixture lane alongside the existing single-user persistence tests.
Integration of the scheduling policy fixes and an actual passing candidate run
remain open in #14421.

## Fixture contract

Use a fresh test namespace, a short periodic schedule interval (for example ten
seconds), and a small known-working pipeline. Prove the
pipeline and controller work on the source version before collecting the final
baseline. Disable the fixture schedules through the API before the upgrade and
re-enable them afterward; do not edit only their CRs, because the API record is
authoritative. The fixture owner must be allowed to create/enable every account
under test. The controller must retain run creation and referenced-pipeline access. Keep the
fixtures exclusive to the test: do not manually submit runs with their recurring
run IDs during observation. Run association alone does not identify the submitting
process.

For target **enforce** mode prepare three distinct recurring runs:

| Scenario | Account and target grants | Predicted status | Observed requirement |
| --- | --- | --- | --- |
| Default control | Omitted account resolving to the configured default | `no_issue_detected` | New controller-created run with that account |
| Scoped grant | Explicit custom account in allowlist; controller has `use` only for that named account | `no_issue_detected` | New controller-created run with that account |
| Denied custom | Explicit custom account in allowlist; controller lacks its `use` grant | `policy_rejection` | Fresh correlated named-account rejection Event and no new run for the whole observation window |

Run **audit** as a separate phase: capture a new prediction report and baseline
with target mode `audit`, apply that mode in the isolated installation, and expect
the denied-custom scenario to produce a run with prediction `operational_impact`.
The verifier checks that audit permits the run; it does not establish that an audit
log or metric was emitted. Keep a default control. Never change enforcement on an
operator's installation for this test.

Verify candidate image revisions, actual authenticated controller identity,
allowlist, compiler patch, independent workflow-identity mode and complete target
RBAC before interpreting results. Template/plugin identities and full workload
completion are outside this main-account firing test.

## Capture the pre-upgrade baseline

Generate a readiness JSON report using the intended target policy and source
KFP evidence. Create a local cases file with actual schedule names/UIDs and account
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
      "expected_outcome": "run_created"
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

## Observe after upgrade and activation

After upgrading the isolated installation, record a UTC activation timestamp
**before** enabling its fixtures (for example `2026-09-18T15:00:00Z`). Pass that
actual timestamp below. This excludes source-version runs that appeared between
baseline capture and upgrade; the baseline timestamp alone is insufficient.
After enabling the fixtures:

```bash
python3 tools/upgrade-readiness/live_schedule_check.py \
  --context isolated-test --namespace readiness-test \
  --kfp-endpoint https://test.example/pipeline --kfp-token-file /secure/test-token \
  --expectations baseline.json --prediction-report readiness.json \
  --not-before "$ACTIVATION_TIME" --timeout-seconds 120 > observed.json
```

The observer permits 30–600 seconds and at most 20 cases. The polling interval can overrun by one in-flight collection cycle; individual
HTTP/Kubernetes request budgets still apply. Source transport byte,
request and response budgets also apply; use small fixture sets. It follows run
pagination, verifies recurring-run association and experiment namespace, and
excludes baseline IDs and pre-observation timestamps. It verifies the service
account on each new run. A run being created is **not** successful task completion.

A blocked result requires a Warning `Failed` Event from the scheduled workflow
controller with the exact involved schedule UID/name/namespace, a fresh timestamp
and increased count, and a named core `serviceaccounts/use` PermissionDenied
signature. Raw messages are inspected in memory and never included in the output.
Generic errors, old Events, absent permissions, truncated collection, changed
message formats and timeouts cannot establish a denial. A blocked case also needs
a positive control to fire, and observation continues through the full window to
catch an unexpected run after an earlier rejection. The controller error signature
is source-derived and still needs confirmation against actual candidate Events.

Treat only a successful verifier exit as a pass for the selected firing checks.
All other exits require investigation; do not interpret an inconclusive observation
as a rejected workload or a successful upgrade. Retain candidate/source revisions,
predictions, baselines and sanitized results in CI artifacts. A release gate must
run both enforce and audit phases; documenting this protocol or running its mocked
unit tests does not satisfy live acceptance.

## Disposable CI fixture tooling

`provision_live_schedules.py` requires the exact context `kind-kfp-readiness` and
`--allow-test-cluster-mutations`. Its `rbac` phase creates a fresh
`kfp-readiness-test` namespace with a unique ownership marker; it refuses an
existing namespace. Later phases require matching private state and namespace
ownership. This is a guard against accidental use, not proof that a context name
points to a disposable cluster: create the dedicated Kind cluster first.

The fixture owner can use both custom accounts. The controller can use only
`readiness-granted`, with a namespaced `resourceNames` grant. Both custom accounts
are allowlisted so the denied case isolates the controller's RBAC check. Runner
permissions are cloned from the source installation's `pipeline-runner` Role.
The isolated setup also grants the API, scheduler, persistence agent and Argo
controller namespace-scoped workload access and gives the API cluster-scoped
TokenReview/SubjectAccessReview access. The fixture copies the test installation's
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
both target modes. Task completion and emitted audit telemetry remain outside this
lane's acceptance scope.
