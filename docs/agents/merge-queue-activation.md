# Merge queue admission and Prow cutover

The activation policy selects `~DEFAULT_BRANCH`, resolved from live repository
metadata to either `master` or `main`. It never enables both names at once.
Support changes can merge with the current policy unchanged. Additional branch
activation requires the coordinated sequence below. The trusted default-branch controller requires both the enabled policy
and an active queue rule; enabling policy without that rule fails closed.
The nondefault name and unconfigured release branches retain existing behavior.
`release-2.18` is explicitly enabled alongside the default selector; its workflow
prerequisites landed in #15702 and #15703. See the release activation section below. Unexpected
default branch names, unreadable metadata, or mismatched repository identity fail
closed. The selector is resolved afresh at enforcement boundaries.

The JSON files linked below are review artifacts, not automatically applied
settings. No workflow applies them or changes live Tide.

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

The trusted `queue_admission` job runs independently of the status-recovery
matrix. Each newly verified queue-ready success wakes the default-branch
recovery workflow with its exact PR head. Every surviving serialized controller
catches up other likely ready PRs, including on the five-minute recovery
schedule, and freshly verifies policy and CI before admission. It also runs on approval,
hold, draft, retarget, and head-change events, and during scheduled recovery.
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

Before automatic admission, the controller checks for dependency manifests or
lockfiles (including GitHub workflow and action definitions) changed by both the incoming PR and an earlier PR in the same queue.
Two PRs can edit the same file and merge cleanly. For an overlap, the controller
tests the combined changes with Git and defers the incoming PR only if Git
reports a conflict on an overlapping dependency file. Other ready PRs can still
enter the queue. Rename aliases are retained on both sides. File discovery uses only immutable
base/head comparisons; a truncated comparison is inconclusive rather than
falling back to a moving PR file list. An inconclusive probe leaves the merge
decision to GitHub.

A confirmed conflict is recorded against the PR head, target branch tip and
ordered queue snapshot. Recovery can reuse that decision while those inputs
match and checks again when they change. Deferral does not consume the retry
reserved for a PR removed after failed queue CI. If a conflicting change has
already merged, the incoming PR may still need to update its branch.

The controller also checks the live base and predecessors after enqueue. A race
causes a compensating dequeue before admission consumption is recorded. A
reserved retry is released only after removal is confirmed, using a bot-authored
receipt bound to the original admission ID, CI fingerprint and confirmed
removal event ID. The removal must remain the latest queue event before
reservation and again before enqueue; later manual queue actions invalidate
the receipt. Uncertain or stale removal evidence keeps the retry consumed.
An acknowledged enqueue that disappears before observation still records its
admission attempt; it cannot bypass the retry cooldown or limit on the next
sweep. The receipt retains GitHub’s enqueue timestamp so removal chronology
does not depend on when the status write finishes; earlier-admission removals
remain ineligible. Read failures after acknowledgment also attempt to record
consumption while retaining the independent cleanup path. A new sweep must recheck the queue and all admission
requirements before retrying; the original admission budget is not reset.

The bridge rereads PR policy around admission and attempts dequeue when policy
changes or admission becomes uncertain. It invalidates affected queue-candidate
statuses before removal where possible; status-write failure does not suppress
an independent dequeue attempt. Queue CI also rereads policy before and after
publishing success. Each boundary fetches queue membership and all member policy fields in two
fresh GraphQL requests, avoiding per-entry REST requests during dispatch fanout.
These mutable snapshots are never cached between boundaries.
GitHub exposes queue entries rather than an authoritative
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

## Builtin token and explicit queue CI

Admission, dispatch, and `ci-passed` publication use the repository's builtin
`GITHUB_TOKEN`. No additional App installation, App private key, or PAT is
required. Keep the existing GitHub Actions binding for the required `ci-passed`
status, and do not grant rule bypass. The trusted controller needs permission to
mutate the queue, dispatch Actions workflows, and write commit statuses. Candidate
code and downloaded PR artifacts never execute in this controller.

GitHub's merge queue normally starts configured Actions CI through
`merge_group: checks_requested`. Explicit dispatch is specific to our automatic
admission path using `GITHUB_TOKEN`, not an inherent merge-queue requirement.
GitHub suppresses most workflow triggers caused by this token to prevent recursive
runs; `workflow_dispatch` and `repository_dispatch` are explicit exceptions
([GitHub documentation](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow)).
In the isolated native-admission probe ([#15562](https://github.com/kubeflow/pipelines/pull/15562)),
token-based enqueue succeeded but candidate Actions CI was not observed; the
human-token control started CI. The explicit-dispatch probe
([#15565](https://github.com/kubeflow/pipelines/pull/15565)) successfully ran
[candidate CI](https://github.com/kubeflow/pipelines/actions/runs/43376145061)
and merged through the native queue after verified status publication.

This preserves automatic admission without an additional App credential,
long-lived PAT, or manual enqueue step. If admission later uses a separate App
credential, verify native `merge_group` delivery end to end before retiring the
dispatch workaround; candidate validation and required-check publication still
need to be preserved.

The controller therefore discovers the current native queue candidate and
explicitly dispatches every missing required workflow on its exact queue ref.
Native `merge_group` runs remain supported; a matching registered native or
dispatched run suppresses duplicate dispatch. The trusted publisher verifies the
complete workflow inventory, exact candidate, latest attempts, live base, and
approval/hold policy before writing `ci-passed` to the candidate SHA. Dispatching
CI does not replace GitHub's queue or directly merge a PR.

Every explicitly dispatched queue workflow receives `queue_head_sha`, `queue_base_ref` (plain `master`
or `main`), and `queue_base_sha`. The queue ref itself forces queue mode; absent
or invalid inputs cannot fall back to ordinary manual execution. A required
validation job checks the queue ref, actual workflow SHA, target branch, and live
base SHA against those inputs. Queue runs execute the full automatic test lanes,
ignoring manual test filters, and do not publish images. Artifact reuse requires
matching queue ref, candidate SHA, and base identity across native and dispatched
producers and consumers.

Recovery is scheduled every five minutes to revisit admission, discover newly
created or rebuilt candidates, dispatch missing lanes, and reconcile candidate
statuses even when events are missed. GitHub can delay scheduled workflows; this
is not a five-minute latency guarantee. Before each dispatch, the controller
writes a per-candidate/workflow commit-status receipt. It waits at least 120
seconds for run registration before retrying a missing run, with a maximum of
three dispatch attempts. These receipts are not passing CI evidence. Registered
failed or cancelled runs are never automatically retried; use an intentional
Actions rerun after investigating. Exhausted registration retries require
operator investigation rather than an unbounded dispatch loop.

Successful enqueue also records `queue-admission-attempt` on the PR head,
consuming the workflow-attempt fingerprint bound to the successful PR status.
Admission compares current evidence with that verified fingerprint, so a running
or failed retry cannot reuse an older green. Recovery permits one automatic
re-admission after GitHub removes a candidate for `failed_checks`, following a
15-minute cooldown and fresh head, CI, approval, hold and enforcement checks.
Manual removals do not qualify. The `queue-admission-retry` receipt consumes this
retry before enqueue, so an ambiguous API response cannot cause a retry loop.
If that retry fails, investigate and deliberately rerun PR CI to obtain a new
verified workflow fingerprint. Merely restamping the existing success does not
reset the budget. This is separate from rerunning a failed candidate workflow
while its queue entry still exists.

For an eligible PR blocked by failed admission CI, recovery can rerun PR Gate's
membership-lookup infrastructure failure after 15 minutes. It considers only the
authoritative matching run, ordered by current attempt time rather than original
creation time, retries attempt 1 once, and reserves at most two distinct reruns
per PR head. Authority is rechecked before each write; historical failures are
never retried as a fallback. Failures in issue-admission rules or other steps are not
retryable. Scheduled CI reconciliation must verify success before queue entry;
recovery does not grant approval or bypass checks.

The isolated full-CI test merged [PR #15620](https://github.com/kubeflow/pipelines/pull/15620)
and [PR #15621](https://github.com/kubeflow/pipelines/pull/15621) through the native
queue without changing either PR head. Both candidates passed all 32 workflows
and real DCO; merge SHAs matched tested candidate SHAs. The earlier isolated test
also exercised revoked approval, hold, consumed-attempt rejection, and queue rebuild
after candidate failure. Temporary rules and branches were removed. See
[CI fixes #15619](https://github.com/kubeflow/pipelines/pull/15619) for retry evidence
and quota/runtime measurements. Explicit controller wakeups and temporary branch
aliases mean production event routing and scheduled recovery still need a cutover
smoke test; isolated success does not establish every lifecycle case below.

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

## Activation artifacts and sequence

Prerequisites: merge [#15619](https://github.com/kubeflow/pipelines/pull/15619),
observe default-branch CI and scheduled recovery, and obtain passing strict Prow
configuration validation and review on
[oss-test-infra #2792](https://github.com/GoogleCloudPlatform/oss-test-infra/pull/2792).
Keep both PRs that change merging responsibility held until a staffed cutover.
Before cutover, read the live default branch via `GET /repos/kubeflow/pipelines`
and require it to be `master` or `main`. Use that exact returned name for all
branch-specific API paths below, never an assumed name. Re-read all effective
rules, required-check App bindings and bypass actors. If they differ from this proposal, stop and update the
reviewed artifacts. Do not blindly replay a saved protection snapshot.

- [merge-queue-ruleset.json](merge-queue-ruleset.json) is a proposed repository
  ruleset scoped to `~DEFAULT_BRANCH`, with no bypass actors. It requires
  squash/ALLGREEN queue merging, at most five building and five merging entries,
  a 120-minute check timeout, and `ci-passed` (Actions App 15368) plus DCO (1861).
  Five concurrent candidate builds follow the CI arrival-rate analysis and the
  seven-PR integration replay, which verified the five-candidate cap and slot
  replacement. The maximum merge group is five PRs, with a minimum of one: a
  ready PR need not wait for a partner. Merge limits apply after checks pass;
  they do not combine candidate CI builds. ALLGREEN still requires every
  candidate's checks to pass. Build concurrency and merge limits are separate
  from the controller's four concurrent workflow dispatch operations. Monitor shared API
  quota and candidate latency after activation; the replay does not establish
  installation-wide hourly API headroom.
- [merge-queue-required-checks.json](merge-queue-required-checks.json) is a narrow
  PATCH payload for the current default branch’s classic `required_status_checks`
  endpoint. It changes strict freshness to false while preserving both existing check/App bindings.
  It must not replace the full branch-protection object. Preserve linear history,
  conversation resolution, push restrictions, deletion/force-push restrictions,
  organization rules, and all other settings.

Execution order (not authorized by preparing these artifacts):

1. Freeze merges into the current default branch and retain a hold on this activation PR until
   its one controlled merge. Resolve who will perform that merge using existing
   repository permissions. If no permitted path exists, stop and obtain explicit
   authorization for a reviewed bootstrap path; do not silently add a bypass.
2. Release the hold on #2792, merge it, and verify `config-updater` deployed the
   exclusions. Check all Tide queries/instances exclude Pipelines `master` and
   `main`, including bot and organization-wide queries. Prow label plugins and
   release-branch merging remain enabled.
3. Under the freeze, merge the reviewed activation PR using the agreed maintainer
   path **before requiring the queue**. Existing strict checks still apply to that
   merge. Policy without an active rule fails closed during this short interval;
   do not reopen normal merging. This avoids requiring the queue before its
   admission controller is enabled. Reverify the live policy and controller SHA.
4. Create the ruleset from `merge-queue-ruleset.json` using
   `POST /repos/kubeflow/pipelines/rulesets`. Resolve the current default branch
   again and verify its effective rules contain the enforced queue and the expected App-bound checks, with no queue
   bypass for Prow or any other actor. Record its returned ruleset ID for rollback.
   Then PATCH only
   `/repos/kubeflow/pipelines/branches/{default_branch}/protection/required_status_checks`
   with `merge-queue-required-checks.json`. Verify `strict: false`, both App
   bindings, and preservation of all unrelated protection settings.
5. Verify production events and scheduled recovery publish fresh queue-admission
   statuses and automatically enqueue two approved PRs. Check exact candidate
   workflow/image identities, DCO and `ci-passed` on queue SHAs, and a second merge
   without author rebasing. Test revoked approval/hold after admission, updated
   head, draft/retarget, failed/cancelled lane, rebuilt candidate, intentional retry,
   and the Dependabot exception. Confirm dispatch-registration grace/retry bounds
   and that rejected admission never falls back to direct merge. The live Prow
   installation must remove `lgtm` on head changes as intended.
6. Reopen normal merging only after the smoke test passes. A later rename between
   `master` and `main` needs no policy or ruleset edit: both follow the repository
   default. Drain the queue before renaming, then recheck effective protections,
   App bindings, event routing and a smoke merge. The selector does not perform
   the rename or migrate unrelated classic branch protection settings.

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
  merge_queue_dispatch_test merge_queue_dispatch_workflows_test \
  prepare_tide_merge_queue_test
```

These deterministic fixtures exercise disabled and enabled policy, API failures,
queued policy revocation, branch/head identity, and external Tide scope. They do
not establish a live queue rollout or deployment of the external configuration.

## Release-2.18 opt-in

Release support is explicitly scoped to `release-2.18`; other release branches
retain their current behavior. The active policy includes the release after its
prerequisites landed. The controller accepts an explicit release alongside
`~DEFAULT_BRANCH`, preserves the live default selector across renames, and requires
an active queue rule independently for each enabled branch. Verified Dependabot PRs use the same automatic admission exception as the
default branch: passing CI is required, but `lgtm` and `approved` are not. Holds
still block admission; human-authored PRs retain both review labels.

The trusted controller and publisher run on the default branch, using the live
release tip for workflow policy and the exact queue candidate for execution.
Release image producers have explicit read-only queue companions; both the PR
producer and matching queue companion must appear in trusted workflow inventory.
Default-branch dispatch registrations for release-only workflow paths fail closed
if run there; the actual implementations execute at the release queue ref.

Activation order (Tide configuration remains unchanged):

1. Merge default-branch controller/registration support, then the release workflow
   PR. Verify every required release lane has exact-ref/SHA dispatch guards,
   automatic test selection and no image publication from queue candidates.
2. Resolve the registered release-only workflow IDs and verify dispatch uses the
   implementation on the release candidate, not the default-branch registration.
3. Apply the explicit release policy from
   [merge-queue-release-2.18-policy.json](merge-queue-release-2.18-policy.json) in a
   reviewed default-branch activation commit. Coordinate its merge with the rule
   below; the controller fails closed while an enabled branch has no rule.
4. Create [the release ruleset](merge-queue-release-2.18-ruleset.json), scoped to
   `refs/heads/release-2.18`, with five candidate builds, one-to-five merging PRs,
   ALLGREEN, squash and a 120-minute timeout. Verify effective enforcement and
   only the repository administrator role (ID 5) with `pull_request` bypass,
   including no Tide App bypass, before relaxing freshness.
5. Read the current release protection and reconcile any drift. Apply only
   [the required-checks patch](merge-queue-release-2.18-required-checks.json):
   preserve existing App-bound DCO and pre-commit checks, add App-bound ci-passed,
   and set strict freshness false. Preserve every unrelated protection.
6. Observe a normally approved release PR automatically enter the queue, run all
   required exact-candidate lanes, publish DCO/pre-commit/ci-passed, and merge.
   Verify scheduled recovery and a hold/revoked approval blocks merging. Tide
   can still attempt direct merges but has no App bypass in the queue rule.
   Repository administrators can explicitly bypass through a PR; this does not
   grant direct-push bypass. Other organization and branch protections still apply.

Do not enable the release rule while its dispatch prerequisites are missing.
For rollback, freeze and drain release candidates, remove the explicit release
policy entry and restore strict release freshness before removing queue
protection. Preserve main's policy and queue rule throughout.
