# CI merge gate

CI Check is the sole publisher of the `ci-passed` commit status. The label with
the same name is informational. Tide remains the merge authority; human PRs
still require review. Default Dependabot creation holds are removed with this publisher. Deployment
requires the coordinated cutover below; existing PR-specific holds are preserved.

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
the deployed Tide configuration blocks that window before lifting holds.

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

GitHub can delay or drop scheduled executions, so 15 minutes is a requested
cadence, not a recovery SLA. More than 256 recovery candidates fails the discovery job
explicitly rather than silently omitting PRs. Investigate failed or missing
scheduled runs; manually rerunning CI Check remains the fallback. Do not rely
solely on `check_run`: GitHub suppresses that trigger when the check suite's
head SHA is associated with GitHub Actions. See the
[GitHub event documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows).

## Validation and coordinated rollout

The publisher and removal of default creation holds ship together. Neither
implementation PR #14111 nor oss-test-infra#2674 may be deployed independently
without containment. The following is a proposed maintainer-run procedure;
record the accepting owner and evidence before starting.

1. Jeff (or the designated infra maintainer) temporarily removes the
   `author: dependabot[bot]` Tide query for `kubeflow/pipelines` from deployed
   config. Retain the human `lgtm`/`approved` queries and their hold exclusions.
   Verify no other matching query allows review-free Dependabot merges.
   This restores review requirements even for newly created, unheld PRs.
   Keep the three existing PR-specific holds. Record the deployed config SHA.
2. Anthony supplies local and hosted regression evidence for the exact final
   implementation SHA. Exercise real publisher success, invalidation, late
   external-check recovery, and stale-head rejection in a controlled repo.
   A green base-defined `pull_request_target` check on #14111 does not validate
   its proposed publisher. If lifecycle validation fails, retain containment
   and fix the PR; do not proceed with deployment.
3. Jeff reviews and lands #14111 using the existing human review/CI path.
   Required `ci-passed` protection is not enabled yet, so this avoids a
   bootstrap deadlock. Default creation holds stop, but review-free matching
   remains disabled by step 1. Verify the deployed trusted revision and real
   publisher lifecycle on upstream before proceeding.
4. The infra maintainer applies #2674's protection and query exclusions while
   retaining the temporary removal of the Dependabot query. Inspect live
   master protection: `ci-passed` required, `strict: true`; verify pending and
   failing constituent contexts block Tide, including queued same-SHA reruns.
   Human PRs retain reviews; release branches do not inherit this new context.
5. Only after recording that evidence, the infra maintainer restores the
   master-only Dependabot query from #2674. Verify an eligible unheld Dependabot
   PR can merge without review while failing/incomplete CI, needs-ok-to-test,
   and an explicit PR hold block it. Preserve existing holds for individual
   maintainer decisions. Record the actual merge and negative-case evidence.

The temporary config edit and its restoration need an infra-maintainer owner
and deployment mechanism agreed with Jeff; they are not included in #2674's
final desired configuration. If they require another PR, discuss that before
creating it. Do not substitute an informal promise to label new PRs for the
verified Tide query change.

Abort if the live configuration cannot be inspected, any lifecycle/negative
case fails, or another query bypasses containment. Keep review-free matching
disabled. If #14111 must be reverted, first restore the default creation holds
and retain the temporary Tide containment, then coordinate removal of the
required status before removing its publisher. Never leave a required status
without a working publisher. The revert trigger for temporary containment is
successful completion and recorded evidence of steps 2–4, not elapsed time or
merely green implementation-PR checks.
