# CI merge gate

CI Check is the sole publisher of the `ci-passed` commit status. The label with
the same name is informational. Tide remains the merge authority; human PRs
still require review. Dependabot creation holds remain until live enforcement
is verified.

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

## Validation and rollout

1. Confirm all current Dependabot PRs remain held and every applicable Tide
   query excludes `do-not-merge/hold`.
2. Run workflow validation and behavioral tests, including missing workflows,
   reruns, base retargets, eligibility revocation, and publication drift.
3. Exercise the revised publisher in a controlled repository. Record the exact
   trusted workflow revision; a green base-defined `pull_request_target` check
   on the implementation PR does not exercise its proposed publisher.
4. Land the publisher while holds remain. Verify real status success,
   invalidation, and recovery before applying the required status configuration.
5. Apply oss-test-infra#2674 and inspect live master protection and Tide:
   `ci-passed` required, strict protection retained, unattended Dependabot
   matching limited to master, and trust/hold labels excluded in all matching
   queries. Confirm pending and failed constituent checks prevent merging.
6. Remove default creation holds in a separate follow-up only after recording
   that evidence. Preserve existing PR holds for individual review.

If controlled validation fails, stop deployment and retain holds. If production
validation fails, restore containment before changing enforcement. Do not remove
the publisher while leaving its status required: coordinate any rollback of
the publisher and protection to avoid blocking unrelated human PRs.
