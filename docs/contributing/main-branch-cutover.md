# Default branch cutover checklist

This is the post-rename reference update. It must not be used as the exceptional
bootstrap merge. Keep the cutover changes in draft until the gates below pass.

## Before the maintenance window

- Merge [Pipelines preparation](https://github.com/kubeflow/pipelines/pull/15274)
  and [Prow/Tide preparation](https://github.com/GoogleCloudPlatform/oss-test-infra/pull/2762).
- Verify the deployed Prow/Tide configuration accepts `main` and requires
  `ci-passed`; a merged configuration PR alone is not deployment evidence.
- Verify branch protections, rulesets, environments, trusted publishing, source
  mirrors, Reviewable, and FOSSA with their owners. Preserve ReadTheDocs release
  pins and existing rolling image tags.
- Establish the narrowly authorized bootstrap actor and rollback owner before
  renaming. If that authority is unavailable, postpone the rename.
- Refresh this draft against the prepared default branch and rerun its checks.

## During the maintenance window

1. Freeze normal merges and drain publications. Record the old default-branch
   SHA and the current external configuration revision.
2. Rename `master` to `main`, verify the SHA is unchanged, and verify protections
   and open PR bases. Do not recreate a divergent `master` branch.
3. Follow the preparation change's
   [controlled bootstrap procedure](../agents/ci-passed.md#controlled-bootstrap-after-the-default-branch-rename).
   Use a separate minimal documentation-only PR. Historical merged records still
   naming `master` are not proof of arrival on `main`.
4. After the bootstrap establishes authoritative `main` provenance, refresh this
   cutover PR and obtain fresh required CI against the new base before merging
   it normally. Verify each changed raw URL and Git install reference resolves
   on `main`; validate backend sample links are pinned to the build commit.
5. Land the external periodic checkout change only after `main` exists. Verify
   the deployed configuration and successful runs of both Pipelines periodics.
   The cleanup job's `BRANCH_NAME=master` targets `kubeflow/kubeflow` and stays.
6. Verify required status revocation, a normal PR's fresh `ci-passed`, smoke-test
   defaults, and publication provenance before unfreezing normal activity.

## Rollback and deferred cleanup

Keep merges and publications frozen on failure. Revert the periodic checkout
change before removing `main`; restore the prior default name and SHA only with
all owners coordinated, then verify protections, PR bases, and required statuses.
If reference changes have landed, revert them as part of that rollback. Do not
relax checks or fabricate success to get past a failed cutover.

Generated OpenAPI license metadata and broad browser-link cleanup are separate
follow-ups. Update generated metadata through the API generator, not by editing
its outputs. External repository branches, release branches, legacy GCPC docs,
and image tags are outside this cutover.
