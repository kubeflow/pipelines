# Merge queue admission and Prow cutover

The queue preparation is inactive by default. `.github/merge-queue-policy.json`
ships with `enabled_branches: []`. CI retains its existing live-base freshness
requirement, push-triggered stale revocation, and Tide merge behavior. The
admission job reads this trusted policy before requesting an App token or calling
queue APIs. Nothing in this repository changes live protection or Tide settings.

## Admission and candidate validation

When a branch is explicitly enabled, the publisher first reads GitHub's effective
branch rules. It requires an active `merge_queue` rule from a repository or
organization ruleset. A missing rule or unreadable response cannot authorize
success. An existing queue object or an evaluate-mode ruleset is insufficient.
Classic branch protection alone is not the activation contract for this code.

In this mode, PR `ci-passed` checks current-head executions and attempts, expected
workflow coverage, external checks, and durable base-retarget history. It still
loads trusted workflow policy from the current target branch. Only the requirement
that executions start after every new base commit is relaxed. Push revocation
skips that branch only while the queue rule is verified. Queue admission statuses
identify their target branch, and recovery revisits greens when policy mode or
base policy changes. Queue candidates continue to run all applicable lanes against
the exact combined SHA, with no relaxation of candidate freshness or image identity.

The trusted `queue_admission` job runs after PR CI reconciliation, on approval,
hold, draft, retarget, and head-change events, and on the existing scheduled sweep.
Ordinary events inspect the affected PR and all current queue members; scheduled
recovery alone sweeps all open PRs. A replaced pending event is therefore not the
only opportunity to remove a held PR. Admission is excluded from the aggregate's test
checks to avoid a dependency cycle; it does not constitute CI evidence.

Eligibility preserves Prow's `lgtm` plus `approved` requirement for normal PRs.
Only the exact `dependabot[bot]` Bot identity receives the existing development
branch review exception. Draft/closed PRs, `needs-ok-to-test`, `needs-rebase`,
`do-not-merge`, and every `do-not-merge/*` hold block admission. Prow remains the
owner of approval labels and OWNERS interpretation. The bridge never adds approval
labels or waives checks. It uses `enqueuePullRequest` with `expectedHeadOid` and
`jump: false`; it never directly merges or enables auto-merge as a fallback.
GitHub still enforces DCO and the other configured required checks.

The bridge rereads PR policy around admission and attempts dequeue when policy
changes or admission becomes uncertain. It invalidates affected queue-candidate
statuses before removal where possible; status-write failure does not suppress
an independent dequeue attempt. Queue CI also rereads policy before and after
publishing success. GitHub exposes queue entries rather than an authoritative
merge-group members connection, so validation conservatively checks all entries
and requires the candidate SHA to appear as a current entry head. A held later
entry can temporarily block an earlier candidate until removal. Incomplete,
malformed, or more-than-100-entry snapshots fail closed. Initial rollout must
confirm the API's entry commit/position semantics using real candidates.

Prow labels and GitHub mutations are asynchronous. This preserves the existing
Prow label-based approval policy; it does not introduce cryptographic binding of
review approval to a head SHA or atomic hold-versus-merge semantics. Test removal
of `lgtm` on head changes in the deployed Prow installation. Event handling and
scheduled recovery reduce missed-event exposure but have no strict latency SLA.

## Credentials and status ownership

Before activation, provision a dedicated repository-scoped GitHub App with
Contents write and Pull requests write permissions, installed only where needed.
Configure `MERGE_QUEUE_APP_ID` and `MERGE_QUEUE_APP_PRIVATE_KEY`. The pinned
`actions/create-github-app-token` step requests only those permissions for the
current repository and revokes the token after the job. Do not grant rule bypass.

Only enqueue/dequeue mutations use this App token, allowing resulting events to
start queue CI. Reads and `ci-passed` writes use the ordinary Actions token. This
preserves the existing required status's GitHub Actions app binding; do not
silently rebind `ci-passed` to the admission App. Credentials are never requested
while the checked-in policy is disabled. No candidate code or downloaded PR
artifact executes in the admission job.

## Prepared Tide change

[merge-queue-tide.patch](merge-queue-tide.patch) targets
`GoogleCloudPlatform/oss-test-infra` revision
`b4a9e3965407f6da8974cdd4875a689dd224bd74`, `prow/oss/config.yaml`.
It is a review artifact, not applied deployment configuration. It:

- Excludes Pipelines `master` and `main` from its explicit normal-PR Tide query.
- Removes its development-branch Dependabot direct-merge query.
- Excludes Pipelines from the overlapping Kubeflow organization query.
- Preserves Pipelines release-branch merging through the explicit query and
  preserves other repositories' policies. Prow review plugins stay enabled.

Regenerate against a fresh external checkout before proposing deployment:

```bash
python3 .github/resources/scripts/prepare_tide_merge_queue.py \
  /path/to/oss-test-infra/prow/oss/config.yaml --patch /tmp/merge-queue-tide.patch
```

The script only writes the specified patch file. It refuses malformed query
scopes, checks all overlapping queries, and preserves unrelated configuration.
Review the patch in an external infrastructure PR; run that repository's Prow
configuration validation there. Its tests compare normal/bot, approval/hold,
branch, and repository combinations and check that no development merge route
survives. An infrastructure PR must not deploy ahead of the coordinated cutover.

## Activation sequence

1. Merge the preparation with `enabled_branches: []`. Verify existing PR CI and
   merges continue normally. Finish the default-branch rename first if desired.
2. Provision the admission App and verify DCO supports synthetic queue SHAs.
   Preserve existing required checks and their App bindings. Prepare an active
   branch-specific ruleset requiring a merge queue, squash merging, and low build
   concurrency; keep release branches on their existing policy.
3. Freeze merges during cutover. Deploy and verify the reviewed external Tide
   query exclusions, covering both names and every overlapping query. Keep Prow
   approval/hold plugins running. No Tide instance should directly merge the
   selected development branch after this step.
4. Enable the queue ruleset on the actual development branch and replace the
   strict up-to-date-branch requirement with queue enforcement. Keep `ci-passed`
   and DCO required. Under the freeze, deploy the explicitly reviewed policy
   change adding only the active branch to `enabled_branches`. The policy change
   itself may need the maintainer's controlled cutover path because normal merges
   are frozen; this preparation does not authorize a protection bypass.
5. Verify fresh PR statuses show queue-admission mode. Queue two approved PRs,
   verify entry API identities, both required queue-SHA statuses, and exact
   candidate image sources. Confirm the second merges without an author rebase.
   Test a failing lane, a rebuilt queue, revoked approval, a hold after admission,
   an updated head, a draft/retarget, and the intended Dependabot exception.
   Verify the bridge never falls back to direct merge when admission is rejected.
6. Reopen normal merges only after those observations. Supporting both branch
   names does not migrate protection, App configuration, or the enabled-branch
   policy during a rename. Avoid simultaneously activating an unused branch.

For rollback, freeze and drain/remove queued candidates first. Restore disabled
admission policy and strict PR base-freshness enforcement, verify fresh required
statuses, and only then restore the reviewed Tide direct-merge configuration and
remove queue enforcement. Do not turn off the queue rule while relaxed admission
is still the intended merge path.

## Local verification

```bash
PYTHONPATH=.github/resources/scripts python3 -m unittest \
  ci_merge_queue_policy_test merge_queue_admission_test \
  merge_queue_admission_workflow_test ci_merge_group_test \
  prepare_tide_merge_queue_test
```

These deterministic fixtures exercise disabled and enabled policy, API failures,
queued policy revocation, branch/head identity, and external Tide scope. They do
not establish a live queue rollout or deployment of the external configuration.
