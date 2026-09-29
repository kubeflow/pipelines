# CI merge gate

CI Check is the intended publisher of the `ci-passed` and `ci-passed-release`
commit statuses. The `ci-passed` label is informational. Tide remains the
merge authority on `master` and branches without a GitHub merge queue;
human PRs still require review. Automatic Dependabot creation holds remain
configured for every ecosystem; existing PR-specific holds remain separate
maintainer decisions and are never removed by the publisher. Removing creation
holds remains a separate change after the deployed merge policy is verified.
Required `ci-passed` protection is intended for `master`; release-2.18 uses
the distinct release context and rollout described below.

## Release 2.18 merge queue

The release rule requires `ci-passed-release` on both PR heads and GitHub's
temporary merge-group SHA. Its separate name prevents a stale `ci-passed`
success from another base branch from admitting a PR immediately after a
retarget. CI Check also keeps publishing the legacy `ci-passed` context on
release PR heads for Tide while the queue uses only `ci-passed-release`. It
uses the existing repository `GITHUB_TOKEN`; no new organization App or Tide
change is needed. The `ci-passed` label still serves Tide and remains
informational for this queue.

GitHub runs `merge_group` workflows on a temporary commit containing PR code.
The documented read-only token downgrade for fork-origin `pull_request`
events does not apply to `merge_group`. Before reporting release PR-head
success, CI Check resolves a fork PR's merge base from the exact current base
and head SHAs, then checks the complete `.github/workflows` tree at every
PR-only commit and every parent. Ordinary fork commits cannot change that tree.
A merge commit may import a trusted release ancestor's workflow tree only when
that ancestor is its own merge base with the current release branch. This
allows a fork PR to merge newer release commits without a false CI rejection,
while rejecting fork-authored workflow changes, change-then-revert histories,
and rollback to an older release workflow version. The final PR head must
still match its merge base's workflow tree. Incomplete or mismatched API
evidence blocks the PR. The queue publisher repeats the check against the
current base for every included fork PR; the PR-head success description is
also stamped with its checked base SHA. A fork PR with more than 100 ahead
commits, 256 unique compared revisions, or eight trusted workflow imports
remains blocked until its history is reduced or the guard is expanded. Review
the baseline `merge_group` **and queue-ref `push`** workflows for write
permissions and exposed credentials
before activation. Repository writers can create their own privileged
workflows and status writers, so this guard relies on the existing trust in
people with write access. Fork authors who need to change release workflows
must work with a maintainer on a same-repository branch.

The trusted default-branch CI Check publisher requires the release PR's `lgtm` and
`approved` labels (with the existing exact Dependabot exception), eligible
author, and absence of Tide's exclusion labels before reporting PR-head
success. It maps a queue entry's exact head commit to its queue position and
checks every current PR
through that position. Later temporary commits can contain PRs ahead of them
in the queue. The queue must use `ALLGREEN`, build concurrency `1`, and merge
at most one PR per operation. GitHub defines build concurrency as the maximum
number of `merge_group` webhooks dispatched at once; it does not promise that
only one completed entry retains a `headCommit`. The publisher rejects more
than one visible built head before reading workflow runs. Merge limits do not
limit a temporary commit to one PR's changes.
The publisher checks every enabled release workflow against the trusted release
branch's trigger inventory. `build-tools-images` and `runtime-base-images` use separate
read-only queue workflows. Missing, running, cancelled, or failing runs cannot
produce success; workflow start and completion events, relevant PR events, and
a scheduled sweep reconcile late or changed results. A PR label event
discovers every cumulative queue commit that includes that PR; a bounded
per-SHA invalidation matrix marks all affected statuses pending before the
validation matrix starts. The queue remains blocked if the entry or
workflow evidence cannot be verified.

An `in_progress` workflow event writes a pending `ci-passed-release` marker with its
run ID and attempt from the discovery job before per-SHA validation. Later
validation sees its own pending marker in commit-status history, verifies
every fenced attempt through the exact-attempt API, and compares selected
workflow runs with their current attempts. A stale Actions run listing or
delayed status read leaves the queue pending. A later successful attempt can
supersede a completed failed attempt. The publisher reserves one write before
GitHub's 1,000-status limit per SHA and context so it can revoke success. A
stalled entry near that limit must be recreated. An undelivered webhook or a
merge decision made before invalidation starts remains a canary risk.
With one visible built head, a green reconciliation makes `N + 4F` direct
Actions run lookups, where `N` is the number of expected workflows and `F`
is the number of distinct fenced run attempts in status history. The current
31-lane inventory normally creates at least one fence per lane, so a green
reconciliation makes 155 direct lookups. Both recovery sweeps run hourly;
workflow and PR events still trigger immediate reconciliation. Rerunning all 31 lanes once
raises a green queue reconciliation to 279 direct lookups. These figures
exclude queue, PR, branch, inventory, workflow-list, and status-history
reads, plus event-driven reconciliations.
GitHub's [`GITHUB_TOKEN` REST limit](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api#primary-rate-limit-for-github_token-in-github-actions)
is 1,000 requests per hour per repository. Before activation, measure the
complete API budget with a live canary, including events and shared-token use.
Stop if remaining capacity or the number of visible built heads falls outside
the verified envelope. A failed status write can leave an earlier success on
the queue SHA, so rate exhaustion during a rerun or label change is a merge
risk; an hourly sweep only lowers routine demand. Keep strict protection until
the canary demonstrates enough headroom for event bursts, then monitor the
token budget during queue operation.

The release PR dequeue helper runs in CI Check's release queue discovery job.
Its `pull-requests: write` token still needs a live canary to confirm GitHub
accepts `dequeuePullRequest` before queue enforcement is enabled. It must mark
every known affected queue SHA pending with a durable retired marker before
removing an ineligible PR, then read back that PR's absent `mergeQueueEntry`.
A later queued entry may have no `headCommit` yet; the missing required status
blocks it while the earlier known SHA is retired. A retired SHA cannot return
to green, even if the PR regains eligibility before the dequeue mutation.
Remove and re-enqueue it to obtain a fresh queue SHA. Mutation denial or
readback failure keeps known SHAs pending and needs maintainer intervention.

Maintainers enqueue release PRs through GitHub. Tide remains configured and
may report direct-merge failures after queue enforcement; it does not enqueue
PRs. GitHub's queue tests each PR with the latest release branch without
rewriting contributor heads. Label revocation is event-driven and cannot be
atomic with GitHub's merge decision, so observe this window during rollout.

Roll out in this order:

1. Land all release workflow `merge_group` triggers or read-only equivalents
   on `release-2.18`, then land the trusted publisher on the default branch.
   The workflow PR changes `.github/workflows` from a fork, so merge it under
   the existing strict rule before the new release guard runs. Keep current
   branch protection while any workflow is missing.
2. Confirm a live release PR-head `ci-passed-release` status comes from CI
   Check after the publisher lands. Test a fork PR that changes a workflow:
   it must remain non-successful. Confirm the repository's Actions event
   policy permits CI Check's trusted `pull_request_target` path, and audit
   queue-ref `push` workflows alongside `merge_group` permissions. GitHub's
   documented default policy for public repositories changes on 2026-11-02;
   configure an explicit repository-level Actions event policy before then
   so the publisher keeps receiving `pull_request_target` events.
3. Configure `ALLGREEN`, build concurrency `1`, and at most one PR merged per
   operation. Enable the queue and require `ci-passed-release` from GitHub
   Actions in the exact `release-2.18` branch rule while keeping the
   up-to-date head requirement active. Preserve DCO, pre-commit, conversation
   resolution, and linear history requirements.
4. Enqueue a controlled canary PR. Verify its
   `workflow_run.head_branch`, GraphQL
   `mergeQueue.entries.headCommit.oid`, and published `ci-passed-release` SHA
   agree.
   Queue A and B together: confirm A has the only built head while B waits
   with a null `headCommit`, then verify the next build and any cumulative
   commit. If two heads remain visible, the publisher fails closed. Revoking
   A's label must retire its known SHA and dequeue it. Confirm the token can
   perform that mutation and read back the absent entry. Check missing/failed
   CI, rerun start, and label/hold revocation. Confirm that the existing DCO
   App reports its required `DCO` check on the queue SHA, and measure remaining
   API budget during this run. Only after those checks pass, remove the
   up-to-date head requirement and begin normal queue use.

If queue validation fails, restore the previous strict branch rule while the
queue remains active, then disable the queue. Only afterward remove the new
required context if rolling back fully. Keep the publisher and merge-group
workflows available until protection is restored.

## Contract and enforcement

| Condition | Required behavior | Enforcement |
| --- | --- | --- |
| `needs-ok-to-test` present | No success, regardless of author | Publisher eligibility and Tide query exclusions |
| No `ok-to-test` | Only exact Dependabot or MEMBER/OWNER/COLLABORATOR authors eligible | Publisher eligibility |
| Expected workflow queued, running, or waiting | Pending; no success label | Trusted PR base workflow definitions and current Actions run state |
| Expected workflow absent | Pending for 15 minutes from the first status on this SHA (or a later base retarget), then failure | Durable status history and PR timeline |
| Expected workflow cancelled or unsuccessful | Failure, even while other work is pending | Current Actions run state |
| Other discovered check pending or unsuccessful | Pending or failure respectively; known failures take precedence | Structured check-run assessment, pinned allcheckspassed action, and Tide contexts |
| API or malformed evidence | Failure; no inferred success | Strict evidence validation |
| Head changes | Old events cannot publish onto the new head | Event SHA binding and PR rereads |
| Base retargets | Old workflow executions cannot authorize the new base | Durable PR timeline cutoff; original run creation must follow the retarget |
| Same-SHA rerun | Reconcile current run attempts, including unsuccessful terminal outcomes | requested/in_progress/completed events, current Actions state, and Tide contexts |
| PR or CI changes during publication | Undo stale success | Full PR snapshot, workflow evidence, and external checks reread |
| Fresh complete CI for an eligible open PR | Recover to success | Same reconciliation path for all events |

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
protection on `master` must continue to require an up-to-date base. The release
queue rollout changes that setting only after queue enforcement is active.

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

A scheduled sweep every hour selects eligible open PRs without a successful
status for their current base branch and SHA (`ci-passed` normally,
`ci-passed-release` on release-2.18), and runs the same reconciler for
each captured number/head pair, with at most four jobs running in parallel.
Successes from a different base, or legacy successes without a base stamp, are
revalidated. Green PRs with a matching base stamp rely on event-driven
invalidation and do not allocate recovery runners.
Each job holds the same SHA-scoped writer lock as event-driven reconciliation
only while checking and publishing; no sleep or long poll holds that lock.
A late external check such as DCO can therefore recover after the final
constituent workflow event. Every sweep rechecks eligibility, head, retarget
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

GitHub can delay or drop scheduled executions, so one hour is a requested
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
