# Default branch cutover checklist

This is the post-rename reference update. It must not be used as the exceptional
bootstrap merge. Keep the cutover changes in draft until the gates below pass.

## Read-only readiness report

Run this from a trusted checkout before the rename, with authenticated `gh`,
Python 3.10+ and PyYAML (already in `.github/scripts/requirements.txt`):

```bash
python .github/resources/scripts/main_branch_readiness.py
python .github/resources/scripts/main_branch_readiness.py --json > readiness.json
```

To bind a later run to a previously recorded `master` commit, add
`--expected-sha <full-40-character-sha>`. The checker reads GitHub using explicit
GET requests and fetches the public deployed Prow configuration. It never
renames branches, edits settings, posts comments, or dispatches jobs.

Exit **1** means `NO-GO`: one or more checks are blocked or unknown. Exit **2**
means `OWNER_REVIEW_REQUIRED`: the automated checks pass, but the listed owner
confirmations remain mandatory. Neither result authorizes the rename or the
exceptional bootstrap merge. An unavailable endpoint or insufficient access is
unknown evidence, not a pass; verify `gh` authentication/access and rerun.

The report checks preparation PR merges, the live Tide policy for both branch
names, the current protection/ruleset baseline, PyPI environment policies, OIDC
subject defaults, and stable default-branch/SHA observations. It records the
Prow deployed revision and configuration digest. A merged Prow PR alone cannot
satisfy the deployment check. Policy drift, including an unexpected Tide merge
route, requires investigation rather than weakening the checker.

This is a pre-rename, point-in-time report, not an atomic snapshot or a complete
security policy audit. It deliberately blocks after the rename or if `main`
already exists. It does not establish bootstrap authority, private integration
readiness, CI health, publication drain, or merge freeze. Rerun in the frozen
window and complete the owner gates below. Preserve the sanitized report with
cutover evidence; do not commit live reports to the repository.

Offline regression tests:

```bash
PYTHONPATH=.github/resources/scripts python -m unittest main_branch_readiness_test
```

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
