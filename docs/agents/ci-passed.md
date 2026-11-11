# Release CI result authority

Release PRs use the CI Check publisher on the repository default branch. GitHub
runs `workflow_run` handlers from that branch even when the triggering workflow
tests `release-2.18`; its scheduled recovery also discovers open release PRs.
The publisher loads expected workflows from each PR's release base and verifies
the current head, workflow attempts, retarget history and all check results
before publishing `ci-passed` success and its informational label.

This authority already exists in
[master revision f1bb71a3e](https://github.com/kubeflow/pipelines/tree/f1bb71a3e/.github/resources/scripts)
and does not depend on the separate artifact-readiness or completion-coalescing
changes. The release branch's former `CI Check` workflow was a redundant,
long-running poll. Its `ci-check-result` artifact had no consumer on the default
branch. Keep that workflow retired rather than adding a second success publisher.

`add-ci-passed-label.yml` retains only immediate label invalidation when a PR is
synchronized, reopened, labeled `needs-ok-to-test`, or loses `ok-to-test`.
It never adds a label. The default-branch publisher owns positive results and
recovery after CI completion. Missing or lost completion events are recovered
by its scheduled sweep; no release-local polling job or cached success is needed.

Release branch protection still requires DCO and pre-commit. Tide's review,
hold and constituent-check policy remains independent; this change does not add
`ci-passed` to branch protection, remove any test workflow, or authorize merges.
If the default-branch publisher is unavailable, repair or rerun that publisher;
do not restore a blind-success replacement or bypass failing constituent checks.
