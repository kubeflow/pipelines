# CI merge gate

CI Check is the sole publisher of the `ci-passed` commit status. The label with
the same name is informational. Tide remains the merge authority; human PRs
still require review. This change does not touch Dependabot creation holds: every
ecosystem keeps `do-not-merge/hold`, and existing PR-specific holds are preserved.
Removing the automatic creation holds is a separate follow-up, merged only after the
publisher and protection below are verified in production.

## Contract and enforcement

| Condition | Required behavior | Enforcement |
| --- | --- | --- |
| `needs-ok-to-test` present | No success, regardless of author | Publisher eligibility and Tide query exclusions |
| No `ok-to-test` | Only exact Dependabot or MEMBER/OWNER/COLLABORATOR authors eligible | Publisher eligibility |
| Expected workflow absent, pending, cancelled, or unsuccessful | No success | Trusted workflow inventory and current Actions run state |
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

`.github/resources/ci-workflow-inventory.json` records trusted `pull_request`
branch/path filters. Every applicable workflow must register and complete
successfully for the PR head, branch, and repository. Unknown filter syntax,
incomplete API responses, and stale inventory block success. The inventory
checks registration at workflow level; individual jobs and matrix conditions
remain the responsibility of each workflow and the discovered-check poller.

When changing workflow names or event triggers, regenerate the inventory:

```bash
# Requires PyYAML in the development environment, not in the status job.
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
`ci-passed` status and runs the same reconciler for each captured number/head
pair, with at most four jobs running in parallel. Green PRs rely on the existing
event-driven invalidation path and do not allocate recovery runners.
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
