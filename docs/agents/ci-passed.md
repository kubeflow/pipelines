# CI merge gate

CI Check is the sole publisher of the `ci-passed` commit status. The label with
the same name is informational. Tide remains the merge authority on `master`
and branches without a GitHub merge queue; human PRs still require review.
This change does not touch Dependabot creation holds: every
ecosystem keeps `do-not-merge/hold`, and existing PR-specific holds are preserved.
Removing the automatic creation holds is a separate follow-up, merged only after the
publisher and protection below are verified in production.

## Release 2.18 merge queue

When enabled, the `release-2.18` queue uses the same required `ci-passed`
context on both PR heads and GitHub's temporary merge-group SHA. The trusted
default-branch CI Check publisher requires the release PR's `lgtm` and
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

An `in_progress` workflow event writes a pending `ci-passed` marker with its
run ID and attempt from the discovery job before per-SHA validation. Later
validation sees its own pending marker in commit-status history, verifies
every fenced attempt through the exact-attempt API, and compares selected
workflow runs with their current attempts. A stale Actions run listing or
delayed status read leaves the queue pending. A later successful attempt can
supersede a completed failed attempt. The publisher reserves one write before
GitHub's 1,000-status limit per SHA and context so it can revoke success. A
stalled entry near that limit must be recreated. An undelivered webhook or a
merge decision made before invalidation starts remains a canary risk.
With one visible built head, a green reconciliation makes one direct lookup
per expected workflow: at most 31 with both release E2E workflows enabled, or
124 direct lookups per hour from the 15-minute sweep. This count
excludes queue, PR, and workflow-list reads and event-driven checks. During
the canary, inspect the token's remaining API budget and the number of
visible built heads; stop if either exceeds the planned envelope.

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

1. Land the trusted publisher on the default branch and all release workflow
   `merge_group` triggers or read-only equivalents on `release-2.18`. Keep the
   current branch protection while any workflow is missing.
2. Inspect a live PR-head `ci-passed` status creator and its GitHub App ID.
   Configure `ALLGREEN`, build concurrency `1`, and at most one PR merged per
   operation. Enable the
   queue and require `ci-passed` in the exact `release-2.18` branch rule while
   keeping the up-to-date head requirement active. Preserve DCO, pre-commit,
   conversation resolution, and linear history requirements. Once queue
   enforcement is active, remove the up-to-date head requirement.
3. Enqueue a controlled canary PR first. Verify its
   `workflow_run.head_branch`, GraphQL
   `mergeQueue.entries.headCommit.oid`, and published `ci-passed` SHA agree.
   Queue A and B together: confirm A has the only built head while B waits
   with a null `headCommit`, then verify the next build and any cumulative
   commit. If two heads remain visible, the publisher fails closed. Revoking
   A's label must retire its known SHA and dequeue it. Confirm the token can
   perform that mutation and read back the absent entry. Check missing/failed
   CI, rerun start, and label/hold revocation before normal queue use.

If queue validation fails, restore the previous strict branch rule while the
queue remains active, then disable the queue. Only afterward remove the new
required context if rolling back fully. Keep the publisher and merge-group
workflows available until protection is restored.

## Contract and enforcement

| Condition | Required behavior | Enforcement |
| --- | --- | --- |
| `needs-ok-to-test` present | No success, regardless of author | Publisher eligibility and Tide query exclusions |
| No `ok-to-test` | Only exact Dependabot or MEMBER/OWNER/COLLABORATOR authors eligible | Publisher eligibility |
| Expected workflow absent, pending, cancelled, or unsuccessful | No success | Trusted PR base workflow definitions and current Actions run state |
| Other discovered check unsuccessful | No success | Pinned allcheckspassed action and Tide contexts |
| Head changes | Old events cannot publish onto the new head | Event SHA binding and PR rereads |
| Base retargets | Old workflow executions cannot authorize the new base | Durable PR timeline cutoff; original run creation must follow the retarget |
| Same-SHA rerun | Reconcile current run attempts, including unsuccessful terminal outcomes | requested/in_progress/completed events, current Actions state, and Tide contexts |
| PR or CI changes during publication | Undo stale success | Full PR snapshot and workflow evidence reread |
| Fresh complete CI for an eligible open PR | Recover to success | Same reconciliation path for all events |

Publishing a commit status is not atomic with PR or CI updates. Tide must still
reject pending/failing constituent contexts and honor strict branch protection.
In particular, GitHub does not emit `workflow_run: requested` for reruns: do not
claim immediate queued-rerun protection from this publisher alone. Verify that
the deployed Tide configuration blocks that window before making the status
required, and before any follow-up removes the automatic creation holds.

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
and changes to the PR snapshot block success. Inventory checks operate at workflow
level; individual jobs and matrix conditions remain the responsibility of each
workflow and the discovered-check poller.

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

## External-check recovery

A scheduled sweep every 15 minutes selects eligible open PRs without a successful
`ci-passed` status for their current base branch and SHA, and runs the same reconciler for
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
Scheduled checks preserve existing non-success while evaluating, and unchanged
status writes are deduplicated to avoid exhausting GitHub’s 1,000-status limit
per SHA/context. A queued candidate that is now successful is still invalidated
before revalidation. Status lookup traverses all combined-status pages.

GitHub can delay or drop scheduled executions, so 15 minutes is a requested
cadence, not a recovery SLA. More than 256 recovery candidates fails the discovery job
explicitly rather than silently omitting PRs. Investigate failed or missing
scheduled runs; manually rerunning CI Check remains the fallback. Do not rely
solely on `check_run`: GitHub suppresses that trigger when the check suite's
head SHA is associated with GitHub Actions. See the
[GitHub event documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows).

## Validation and coordinated rollout

This change ships the publisher only. Automatic Dependabot creation holds stay in
place, so no containment edit to deployed Tide configuration is required and none is
proposed. Merge order:

1. **Land the publisher.** Fix the stale workflow inventory, pass CI, and merge
   `kubeflow/pipelines#14111` / its replacement through the existing human review and
   CI path. Required `ci-passed` protection is not enabled yet, so this avoids a
   bootstrap deadlock, and every new Dependabot PR is still held.
2. **Verify the publisher on upstream.** On real PRs, exercise publisher success,
   failure and invalidation, late external-check recovery, stale-head rejection, and
   base-retarget invalidation. Confirm that ordinary eligible PRs actually receive the
   `ci-passed` status before the status is required anywhere.
3. **Merge `oss-test-infra#2674` and verify its deployed effect**, not its content:
   inspect live `master` protection (`ci-passed` required, `strict: true`, release
   branches unaffected) and Tide behavior, including pending or failing constituent
   contexts and queued reruns. A committed configuration change is not evidence that
   branch protection or Tide changed; verify the applied state directly.
4. **Merge a small follow-up that removes the automatic creation holds.** Existing
   PR-specific holds remain separate maintainer decisions and are never removed
   automatically. Demonstrate an eligible unheld Dependabot merge without review (or
   record the blocker) and the negative cases: incomplete or failing CI,
   `needs-ok-to-test`, and an explicit PR hold.

Containment note: holds are only effective for PRs created after the label
configuration that adds them. Dependabot PRs opened before such a change do not
retroactively receive the label, so audit open Dependabot PRs for missing
`do-not-merge/hold` before relying on holds as containment (36 of 36 open Dependabot
PRs carried the hold as of 2026-09-25).

Abort and keep the status non-required if the applied protection cannot be inspected,
if any lifecycle or negative case fails, or if another Tide query permits review-free
Dependabot merges. Never leave a required status without a working publisher: restore
the creation holds and re-enable review requirements before reverting the publisher,
and coordinate removal of the required status before removing its publisher.
