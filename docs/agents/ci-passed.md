# CI merge gate

CI Check is the publisher of the `ci-passed` commit status and the only writer
of `success` or `pending`. A separate push-triggered workflow
(`ci-passed-stale-revoke.yml`) is a second writer that revokes a stale `success`
to `failure` the moment `master` or `main` advances, before the scheduled sweep runs; it
narrows the stale-base window and does not close it (see the section below). The
label with the same name is informational. Tide remains the merge authority;
human PRs still require review. Automatic Dependabot creation holds have been
removed; merging now relies on the required `ci-passed` status and the deployed
Tide policy. Existing PR-specific holds remain separate maintainer decisions and
are never removed by the publisher. Required `ci-passed` protection is intended
for the development default branch: older release branches do not publish this
commit status. The publisher supports both `master` and `main`; external Tide
and GitHub protection must cover `main` before renaming the default branch.

## Push-triggered stale revocation

A push to `master` or `main` runs `ci-passed-stale-revoke.yml`, which flips a stale
`ci-passed` success to `failure` as soon as the base advances instead of waiting
up to 15 minutes for the scheduled sweep. It reuses the same freshness
determination as recovery: a green is fresh only when its description still
carries the stamp for the live base tip. This is mitigation, not a hard
guarantee: the window between a base advance and the status write is narrowed
but not closed, and merge-time freshness enforcement (a merge queue that checks
its synthetic commit) is tracked separately. It resolves the live triggering branch tip and
skips revocation when the tip has already moved past the triggering push, so an
out-of-order job does not overwrite a green stamped for a newer base. It runs only
trusted base-branch code and writes only the commit status (`contents: read`,
`pull-requests: read`, `statuses: write`); the scheduled sweep still reconciles
the informational label. Revocation concurrency is scoped by `github.ref`:
a push to one development branch must not replace the other branch's pending
run. `cancel-in-progress: false` preserves running jobs but does not prevent
replacement of a pending job within the same group.

## Contract and enforcement

| Condition | Required behavior | Enforcement |
| --- | --- | --- |
| `needs-ok-to-test` present | No success, regardless of author | Explicit publisher veto; current Tide queries rely on this protection |
| Any author, membership, or bot status; no `ok-to-test` | Assess the same current-head CI evidence | Author-independent publisher and recovery |
| Expected workflow queued, running, or waiting | Pending; no success label | Trusted PR base workflow definitions and current Actions run state |
| Expected workflow absent | Pending for 15 minutes from the first status on this SHA (or a later base retarget), then failure | Durable status history and PR timeline |
| Expected workflow cancelled or unsuccessful | Failure, even while other work is pending | Current Actions run state |
| Other discovered check pending or unsuccessful | Pending or failure respectively; known failures take precedence | Structured check-run assessment, pinned allcheckspassed action, and Tide contexts |
| API or malformed evidence | Failure; no inferred success | Strict evidence validation |
| Head changes | Old events cannot publish onto the new head | Event SHA binding and PR rereads |
| Base retargets | Old workflow executions cannot authorize the new base | Durable PR timeline cutoff; original run creation must follow the retarget |
| Same-SHA rerun | Reconcile current run attempts, including unsuccessful terminal outcomes | requested/in_progress/completed events, current Actions state, and Tide contexts |
| PR or CI changes during publication | Undo stale success | Full PR snapshot, workflow evidence, and external checks reread |
| Fresh complete CI for an open PR without the explicit hold | Recover to success | Same reconciliation path for all events |

Publishing a commit status is not atomic with PR or CI updates. Tide must still
reject pending/failing constituent contexts and honor strict branch protection.
In particular, GitHub does not emit `workflow_run: requested` for reruns: do not
claim immediate queued-rerun protection from this publisher alone. Verify that
the deployed Tide configuration blocks that window before making the status
required, and whenever changing the surrounding merge policy.

The writer needs `pull-requests: write` to synchronize PR labels, in addition to
its commit-status permission; recovery discovery remains read-only. Normal
reconciliations check out the trusted event SHA. Closed PR events instead check
out the base repository's fully qualified default branch: after a fork PR merges,
its trusted event SHA is also the PR merge SHA that checkout's safety guard
rejects. Neither path checks out a PR ref or disables the guard. Current PR/head
validation still governs status and label changes after checkout.

CI result reporting does not authorize workflow execution. The publisher does not
read author identity, GitHub `author_association`, or organization membership, and
does not require `ok-to-test`. A contributor whose current-head workflows all
pass receives the same result as a member or bot. Missing, unapproved, pending,
failed, and stale workflows still cannot produce success. Workflow approval
remains owned by `gh-workflow-approve.yml` and the existing admission mechanisms.

The explicit `needs-ok-to-test` veto remains because current Tide queries do not
independently exclude that label on every merge path. Removing this veto requires
coordinated protection in every overlapping Tide query. The publisher never
removes this label, `do-not-merge`, or `do-not-merge/hold`; reviews and merge policy
remain independent requirements.

## Expected workflow inventory

`.github/resources/ci-workflow-inventory.json` records local `pull_request`
branch/path filters for regression checks. At runtime, the publisher loads workflow
YAML from the PR's immutable base SHA in the upstream repository. It fetches the
workflow directory in one GraphQL query and parses its contents as data with the
trusted publisher's inventory generator and pinned PyYAML dependency. It never
executes or checks out PR code. This lets release branches retain their own
workflow filters, even when they predate the committed inventory.

Every applicable workflow must register and complete successfully for the PR head,
branch, and repository. Unknown filter syntax, incomplete or malformed API data,
and changes to the PR snapshot block success. Matching runs require valid IDs,
attempt numbers, and creation/start timestamps before latest-attempt selection;
a malformed competing run cannot be discarded in favor of an older success.
Inventory checks operate at workflow level; individual jobs and matrix conditions remain the responsibility of each
workflow and the discovered-check validators.

The sole paused lane is `.github/workflows/upgrade-test.yml` when every job has
an explicit checked-in `if: false` or `if: ${{ false }}` condition. The publisher
reports this pause in its evidence. The workflow is paused pending #14029;
re-enabling it requires a reviewed change that removes the pause guards and
updates the inventory. There is no repository-variable override. Other guards,
mixed enabled/disabled jobs, and all other workflows still require success;
a `skipped` conclusion does not satisfy an expected workflow.

This policy belongs to the PR's immutable base SHA. After the enabling change
merges, update PR branches to include that base policy and trigger fresh upgrade
coverage. A successful status records a bounded fingerprint of its validated
base branch and SHA. Scheduled recovery revisits older successes without that
stamp and successes for a different base branch or SHA; publication also rechecks
the base before and after writing success. Branch
protection must continue to require an up-to-date base as described above.

When changing workflow names, event triggers, or the checked-in upgrade pause,
regenerate the inventory:

```bash
# Requires PyYAML (pinned in .github/scripts/requirements.txt).
python3 .github/resources/scripts/generate_ci_workflow_inventory.py
python3 -m unittest discover -s .github/resources/scripts -p '*_test.py'
```

The inventory hashes only `name` and `on` blocks. Dependency version changes
inside jobs do not require regeneration. New PR workflows must also appear in
CI Check's explicit `workflow_run.workflows` selector; a regression test checks
the selector against the inventory. This list excludes CI Check itself.

After retargeting, rerunning an old execution is insufficient: GitHub retains
its original GITHUB_SHA/GITHUB_REF. Update the PR head to trigger fresh CI.
If any expected workflow does not run, inspect its trigger and approval state.
The registration deadline is derived from the earliest `ci-passed` status on the
head SHA, bounded by the most recent base retarget. Repeated events, reruns, and
changed status explanations do not extend it. API propagation can briefly leave
that first status unreadable; the next reconciliation starts from its durable
timestamp. Existing registered workflows have no aggregate runtime timeout;
their own workflow timeouts remain authoritative.

## External-check recovery

A scheduled sweep every 15 minutes selects open PRs without `needs-ok-to-test` or a successful
`ci-passed` status for their current base branch and SHA, and runs the same reconciler for
each captured number/head pair, with at most four jobs running in parallel.
Successes from a different base, or legacy successes without a base stamp, are
revalidated. Green PRs with a matching base stamp rely on event-driven
invalidation and do not allocate recovery runners.
Each job holds the same SHA-scoped writer lock as event-driven reconciliation
only while checking and publishing; no sleep or long poll holds that lock.
A late external check such as DCO can therefore recover after the final
constituent workflow event. Every sweep rechecks the explicit hold, head, retarget
history, expected workflows, and discovered checks. Stale sweep entries do not
write to newer heads; existing holds are never removed by the publisher.
Preliminary invalidation preserves existing non-success on every event. The final
assessment can replace an old failure with pending when a rerun starts. Status
writes are deduplicated by state and the published 140-character explanation to
avoid exhausting GitHub's 1,000-status limit per SHA/context. A changed explanation
updates its run link; a new run link alone does not create another status. The
`ci-passed` label remains absent until success. Label synchronization errors do
not replace an already selected failure explanation.

The structured reader paginates check runs and head-scoped Actions runs, matches
GitHub Actions checks to workflow suite IDs, and ignores superseded executions.
Latest workflow states count even before their first check record registers.
The publisher is excluded by its exact workflow path and associated suite, so
its own execution cannot hold the aggregate pending.
For reruns, it reads the current attempt's job check IDs and retains successful
jobs from earlier attempts so failed-jobs-only reruns can recover. Checks from
different apps remain distinct. Individual neutral/skipped checks and the existing
name exclusions retain their prior behavior; expected workflows must still finish
successfully. Empty, incomplete, unknown, or inconsistent evidence cannot be green.
Commit statuses are excluded, avoiding a dependency on `ci-passed` or `tide` itself.

The pinned checker remains an independent success requirement. If it was skipped
before workflow evidence recovered, the result remains pending until another
reconciliation validates all checks. Its failed outcome cannot authorize success.
Both expected workflows and external check evidence are reread after publication
to revoke a success that raced with a rerun. This adds API reads, including one
paginated jobs lookup per relevant rerun attempt; it adds no waiting loop or runner
held for the suite's duration. A queued candidate that is now successful is still
invalidated before revalidation. Status lookup traverses all combined-status pages.

GitHub can delay or drop scheduled executions, so 15 minutes is a requested
cadence, not a recovery SLA. More than 256 recovery candidates fails the discovery job
explicitly rather than silently omitting PRs. Investigate failed or missing
scheduled runs; manually rerunning CI Check remains the fallback. Do not rely
solely on `check_run`: GitHub suppresses that trigger when the check suite's
head SHA is associated with GitHub Actions. See the
[GitHub event documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows).

## Missing base merge record

The freshness check establishes when the base revision became reachable on
the PR's development base branch (`master` or `main`) by reading the merged-PR record for that commit. A commit merged
through a pull request carries an authoritative `merged_at` timestamp. A direct
push of a pre-existing commit has no merged PR, so its arrival cannot be
established and the check fails closed to pending rather than passing stale
evidence as fresh.

The commit-to-PR association endpoint can omit a real merge. If it supplies no
verified match, CI Check independently scans up to 500 recently updated closed
PRs targeting the base branch (five pages of 100). Discovery only selects
candidates whose merge SHA exactly equals the base SHA. Each candidate, including
associated PRs, is reread through the individual PR endpoint and must have the
same number, `merged: true`, the expected base repository and branch, the exact
merge SHA, and a parseable string `merged_at`. Only that verified timestamp is
used by the existing freshness check. A rerun of an execution created before
that timestamp remains stale; commit dates, titles, and messages are never
substituted for merge provenance.

API failures remain pending with a retry diagnostic. Missing matches, including
exhaustion of the bounded recent-PR window, remain pending and request maintainer
investigation; malformed timestamps get a distinct diagnostic. Read the
`ci-passed` status and CI Check logs to tell them apart. A missing association
alone does not prove a direct push: inspect the authoritative merged PR record.
Rerunning tests cannot establish missing provenance or expand the search window.

Recovery for a missing record is a maintainer action. A maintainer should
confirm whether the base revision landed through a pull request. If it did,
investigate why the merge record is absent or malformed and correct it. If it
landed through a direct push (for example a fast-forward integration), the
maintainer decides the next step, such as merging a follow-up through a pull
request to restore provenance. Do not weaken the freshness check to admit
stale evidence.

For stacked pull requests, retarget each dependent PR to `master` after its
parent merges, then merge it through that PR. Merging into a staging branch and
fast-forwarding `master` to the resulting commit does not create a merge record
for `master`. The staging PR's timestamp proves arrival only on its own base
branch and must not be reused as the arrival time on `master`.

A recovery PR targeting `master` is subject to the same missing-base blocker.
After reviewing its diff and available CI evidence, a maintainer must explicitly
authorize any exceptional merge required to restore provenance. Keep required
checks and the freshness validator intact; do not synthesize a successful
`ci-passed` status. Verify the recovery PR's authoritative merged record names
`master` and its merge SHA matches the new base tip. Then update affected PR
branches to include that tip and trigger new CI executions. Rerunning executions
created before the recovery merge does not establish freshness against the new
base.

## Controlled bootstrap after the default branch rename

**The rename is not ready for normal merging until this bootstrap is complete.**
Renaming `master` to `main` retargets open PRs, but the last base commit's
historical merged PR still identifies `master`. The verifier deliberately rejects
that record as proof of arrival on `main`; even newly successful CI remains
pending with `no merged PR for base ...`. Do not treat another branch's merge
record as trusted, synthesize a `ci-passed` success, or wait for the sweep to repair
this. The sweep uses the same provenance requirement.

Use one explicitly authorized bootstrap PR to establish a real merge record on
`main`. This is a cutover operation, not an automatic exception in the validator:

1. **Before renaming**, identify the maintainer authorized to perform one exceptional
   PR merge with pending `ci-passed`, and verify the applicable repository and
   organization rules permit that actor's narrowly scoped bypass. If that authority
   is unavailable, stop before renaming; this preparation PR alone cannot complete
   the migration. Do not disable required checks or grant broad bypass permissions.
2. Freeze all normal merges, including every overlapping Tide query and Dependabot,
   and drain publication jobs. Record the final `master` SHA `B`, its authoritative
   merged PR, current rules/check app bindings, and the intended bootstrap reviewer.
3. Rename through GitHub and verify `main` still points to `B`, the old historical
   record still names `master`, and main protection/Tide requirements are active.
   Keep normal merges frozen throughout the following steps.
4. Open a minimal documentation-only bootstrap PR against `main`, based on `B`.
   For example, record the branch migration checkpoint in the migration guide.
   Review its exact diff and head SHA `H`; require DCO, the normal review/label
   requirements, and all applicable constituent CI and external checks to pass on
   fresh executions for `H` against `main`. Confirm `ci-passed` is pending solely
   because `B` lacks a verified main merge record. Any other failure blocks the
   bootstrap. An empty diff or a direct push is not a substitute for a merged PR.
5. Record explicit authorization on that PR for **this PR, head `H`, and base `B`**.
   Immediately re-read both SHAs and the evidence, then have the authorized actor
   perform the one exceptional PR merge, pinning the reviewed head (for example,
   `gh pr merge <number> --repo kubeflow/pipelines --squash --admin --match-head-commit <H>`
   only where existing policy permits that actor's bypass). Abort if either SHA
   changed, the pending reason changed, or the merge cannot be performed under
   existing policy. This is not authorization to merge any other pending PR.
6. Read the merged PR through GitHub's API. Require `merged: true`, upstream
   repository identity, `base.ref: main`, a valid `merged_at`, and
   `merge_commit_sha` equal to the current main tip `C`. Preserve these records
   with the cutover evidence. A failed or ambiguous read keeps the freeze in place.
7. Update subsequent PRs to include `C` and trigger **new** CI executions strictly
   after the bootstrap merge and any retarget event. Rerunning an old execution is
   insufficient. Verify CI Check produces success through the ordinary validator,
   and that missing/failing CI, stale base evidence, and delayed revocation events
   remain blocked. Verify both branch concurrency groups are independent.
8. Reconfirm protections and resume Tide/normal merges only after the normal path
   is proven. If bootstrap cannot be completed, retain the freeze and use the
   coordinated rollback procedure; do not weaken the validator to recover.

The regression `test_rename_requires_controlled_bootstrap_then_fresh_main_ci`
models the transition: a historical master record passes on master and stays
pending on main; a real main bootstrap record allows only strictly newer CI.
GitHub documents [branch rename behavior](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-branches-in-your-repository/renaming-a-branch)
and [pending-run replacement](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency).

## Validation and coordinated rollout

Publisher code always executes from the trusted base revision. A PR changing the
publisher therefore does not change the status display on its own CI run; the new
behavior becomes active after merge. Validate the structured reader with fixture
regressions and read-only live snapshots, then exercise pending, failure, success,
reruns, base retargets, and late external-check recovery on upstream after merge.
The release tool's `watch_pr_ci` must keep waiting for pending and still stop on
actual failure.

The coordinated infrastructure change is
[GoogleCloudPlatform/oss-test-infra#2674](https://github.com/GoogleCloudPlatform/oss-test-infra/pull/2674).
It scopes required `ci-passed` protection to `master`, preserves strict protection,
adds `needs-ok-to-test` exclusions to overlapping Tide queries, and makes Tide
honor GitHub's blocked state. Release branches keep their existing requirements;
requiring `ci-passed` before their publisher is upgraded would block releases.
The infrastructure PR needs its own maintainer testing, review, and approval.

Inspect live GitHub protection and deployed Tide behavior; configuration in a PR
or in the Prow repository is not evidence that protection was applied. Verify
`master` requires `ci-passed` with `strict: true`, existing required checks retain
their app bindings, release protections remain unchanged, and pending/failing
constituent contexts and queued reruns block merges. Do not claim the external
Tide rollout complete until its deployed effect is verified.

Never remove or revert a required status's publisher without coordinating the
corresponding protection and merge policy. If rollback is needed, retain a working
publisher and restore containment before changing merge requirements. Holds only
protect PRs that actually carry them; audit existing PRs as well as creation labels.

## Recover a pending aggregate after a release merge

Ordinary reconciliation preserves a merged PR's published status. If a release
PR merges while `ci-passed` is pending, later successful constituent runs cannot
repair that aggregate through the open-PR sweep. Release qualification still
requires green CI; do not bypass the gate or manually create a success status.

After verifying the PR number and exact tested head, dispatch **CI Check** from
the default branch with `merged_pr` and `head_sha`. For example:

```sh
gh workflow run ci-checks.yml --ref master -f merged_pr=12345 -f head_sha=<40-character-head-SHA>
```

This explicit recovery accepts only merged `release-X.Y` PRs in this repository
whose aggregate is pending, or whose leftover success/failure is proven to come
from an interrupted recovery run. It binds the head, merge commit, repository and branch,
and accepts only single-commit PRs whose tested head and merged commit have
identical trees and identical sole parents. This proves the immutable pre-PR
base without trusting a subsequently advancing PR base field. Multi-commit PRs,
merge commits with multiple parents, and changed parents or trees fail closed;
a first parent alone is not proof because a multi-commit rebase can put it inside
the PR. The required workflow inventory comes from this verified parent. Other green or failed aggregate statuses are left untouched. The pinned independent checker,
required-workflow validation and external-check reader must all pass. Identity
changes, unavailable evidence and pending or failed checks cannot authorize a
success. Recovery does not change PR labels. No release tag or package is published by this workflow.

Cleanup retries the exact status-ID ownership lookup/write three times. A
transient error cannot silently skip rollback; a persistent API outage fails the
job explicitly. Once GitHub is available, start a **fresh dispatch**, rather than
rerunning the failed run. Recovery authenticates the old status URL against the
Actions run's repository, workflow path, dispatch event, default branch and
completed failure/interruption outcome, then resets only that exact status to
pending before checking evidence. A running owner fails explicitly, including a
rerun of the same run ID. Newer writers are preserved. Do not proceed with release
publication while cleanup is outstanding.
