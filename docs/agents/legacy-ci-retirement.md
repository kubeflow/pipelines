# Legacy v2 workflow retirement

Tracking: [#14950](https://github.com/kubeflow/pipelines/issues/14950).
The [assertion ledger](../../backend/test/v2/api/legacy-migration.md) links the
existing execution evidence, including investigated failures. No additional
retention period or repeated-comparison requirement applies.

Two merges preserve the existing trusted CI gate:

1. Keep legacy tests running while the trusted publisher learns the exact
   master-only replacement: `legacy-v2-api-integration-tests.yml` is satisfied
   by an applicable `api-server-tests.yml` run. Missing, failed, stale, or
   wrong-head API runs still block. Release branches retain their legacy rule.
2. After that publisher change merges, delete the legacy workflow and regenerate
   the inventory and completion selector. Keep service-account and migration
   manifests enforced in the API workflow. Preserve webhook and shared fixtures.

A head-only policy change cannot activate itself: the publisher executes trusted
repository code. Validate the deletion PR against the deployed publisher before
merging. If replacement coverage regresses, restore the legacy workflow and its
required status behavior; do not disable branch protection.
