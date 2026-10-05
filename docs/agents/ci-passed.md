# CI merge gate

CI Check is the sole publisher of the `ci-passed` commit status. The label with
the same name is informational. Tide remains the merge authority; human PRs
still require review. Automatic Dependabot creation holds have been removed;
merging now relies on the required `ci-passed` status and the deployed Tide
policy. Existing PR-specific holds remain separate maintainer decisions and are
never removed by the publisher. Required `ci-passed` protection is intended for
`master` only: older release branches do not publish this commit status.

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
