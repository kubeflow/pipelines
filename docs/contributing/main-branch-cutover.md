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
   [controlled bootstrap procedure](https://github.com/kubeflow/pipelines/blob/main/docs/agents/ci-passed.md#controlled-bootstrap-after-the-default-branch-rename).
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

## Operator worksheet

Complete this worksheet in the cutover tracking record. Every owner and evidence
field starts **pending**; a passing readiness report does not fill them in.

| Gate | Owner to name | Evidence required before proceeding |
| --- | --- | --- |
| Window and rollback | Cutover coordinator and rollback owner | UTC window, contact path, freeze/unfreeze decision maker and rollback plan |
| Merge freeze | Prow/Tide operator and repository maintainer | All overlapping normal and bot queries paused; other merge paths controlled |
| Publications | Release operator | In-flight publishers drained; scheduled/manual publishers controlled |
| Bootstrap authority | Authorized maintainer and independent reviewer | Existing applicable permission, exact PR/head/base authorization; no new broad bypass |
| Source mirror | Copybara owner | Source/destination branch configuration and post-rename validation plan |
| Review services | Reviewable and FOSSA owners | Branch filters/default behavior verified and post-rename validation plan |
| Package publication | PyPI trusted-publisher owner | Repository, workflow and environment binding verified; main policy preserved |
| Documentation | ReadTheDocs owner | Existing release pins preserved; build verification plan |
| External CI | Prow/Tide operator | Deployed preparation revision, main policy, periodic deployment/recovery owner |

Record dated evidence and owner confirmation, not just a checkbox. Do not paste
credentials, webhook URLs, or private integration configuration into a public PR.
If an owner cannot confirm a gate, postpone the rename.

### 1. Capture the frozen baseline

Use Bash with authenticated `gh`, `git`, `jq`, and the readiness checker's Python
dependencies. Run from the trusted cutover tooling checkout. These commands only
read remote state and write local evidence. Keep the evidence directory outside
the repository; choose a new directory for each attempt. Set `KFP_BASE_PR` to
the authoritative merged PR for the frozen master tip before running the block;
its merge commit must equal B. Missing or ambiguous provenance blocks cutover.

```bash
set -euo pipefail
KFP_REPO=kubeflow/pipelines
KFP_EVIDENCE="$(mktemp -d "${TMPDIR:-/tmp}/kfp-cutover-evidence.XXXXXX")"
gh api --hostname github.com --method GET "repos/$KFP_REPO" > "$KFP_EVIDENCE/repository-before.json"
jq -e '.default_branch == "master"' "$KFP_EVIDENCE/repository-before.json"
gh api --hostname github.com --method GET "repos/$KFP_REPO/git/ref/heads/master" > "$KFP_EVIDENCE/master-before.json"
KFP_BASE_SHA="$(jq -er '.object.sha' "$KFP_EVIDENCE/master-before.json")"
gh api --hostname github.com --method GET "repos/$KFP_REPO/branches/master/protection" > "$KFP_EVIDENCE/master-protection.json"
for KFP_RULESET in 14287837 14287367 14287485; do
  gh api --hostname github.com --method GET "repos/$KFP_REPO/rulesets/$KFP_RULESET" > "$KFP_EVIDENCE/ruleset-$KFP_RULESET-before.json"
done
: "${KFP_BASE_PR:?Set the authoritative merged PR for the frozen master tip}"
gh api --hostname github.com --method GET "repos/$KFP_REPO/pulls/$KFP_BASE_PR" > "$KFP_EVIDENCE/base-merged-before.json"
jq -e --arg base "$KFP_BASE_SHA" \
  '.merged == true and .base.repo.full_name == "kubeflow/pipelines"
   and .base.ref == "master" and .merge_commit_sha == $base' "$KFP_EVIDENCE/base-merged-before.json"
if python .github/resources/scripts/main_branch_readiness.py \
  --expected-sha "$KFP_BASE_SHA" --json > "$KFP_EVIDENCE/readiness.json"; then
  echo 'Unexpected readiness exit code; stop and investigate.' >&2
  exit 1
else
  KFP_READINESS_EXIT=$?
fi
test "$KFP_READINESS_EXIT" -eq 2
jq -e '.automated_checks_passed == true and .decision == "OWNER_REVIEW_REQUIRED"' \
  "$KFP_EVIDENCE/readiness.json"
```

**Expected:** default remains master and every automated check passes. Record the
final `B = KFP_BASE_SHA`, its authoritative merged PR, the deployed Prow revision,
and the protection baseline. **Stop:** exit 1, unknown evidence, unexpected exit
code, any SHA drift, or an unconfirmed worksheet gate. Exit 2 means only that the
operator may finish reviewing the owner gates. It is not rename authorization.

### 2. Verify the authorized rename

The authorized operator performs the GitHub rename only after the worksheet and
freeze are complete. This checklist deliberately does not automate that action.
Then run the following reads in the same shell:

```bash
gh api --hostname github.com --method GET "repos/$KFP_REPO" > "$KFP_EVIDENCE/repository-after.json"
jq -e '.default_branch == "main"' "$KFP_EVIDENCE/repository-after.json"
gh api --hostname github.com --method GET "repos/$KFP_REPO/git/ref/heads/main" > "$KFP_EVIDENCE/main-after.json"
jq -e --arg base "$KFP_BASE_SHA" \
  '.ref == "refs/heads/main" and .object.sha == $base' "$KFP_EVIDENCE/main-after.json"
gh api --hostname github.com --method GET "repos/$KFP_REPO/pulls/$KFP_BASE_PR" > "$KFP_EVIDENCE/base-merged-after.json"
jq -e --arg base "$KFP_BASE_SHA" \
  '.merged == true and .base.repo.full_name == "kubeflow/pipelines"
   and .base.ref == "master" and .merge_commit_sha == $base' "$KFP_EVIDENCE/base-merged-after.json"
gh api --hostname github.com --method GET "repos/$KFP_REPO/branches/main/protection" > "$KFP_EVIDENCE/main-protection.json"
gh api --hostname github.com --method GET --paginate --slurp \
  "repos/$KFP_REPO/rulesets?includes_parents=true&per_page=100" > "$KFP_EVIDENCE/rulesets-after.json"
for KFP_RULESET in 14287837 14287367 14287485; do
  gh api --hostname github.com --method GET "repos/$KFP_REPO/rulesets/$KFP_RULESET" > "$KFP_EVIDENCE/ruleset-$KFP_RULESET-after.json"
done
```

**Expected:** main still equals B. Compare effective protection, required context
app bindings, ruleset details and exclusions with the captured baseline; the
ruleset list alone is not proof of enforcement. Verify deployed Tide policy and
open PR retargeting. **Stop:** any mismatch, missing permission, ambiguous API
response, or unexpected write. Keep the freeze in place. The pre-rename readiness
checker intentionally fails after this point; do not modify it to make it pass.

### 3. Prepare the separate bootstrap PR

Do not use this cutover PR, cherry-pick its changes, or create the bootstrap from
its branch. In a dedicated clean checkout, fetch upstream `main`, verify its SHA
is B, and create `codex/main-rename-bootstrap` from B. If that branch already
exists, inspect it instead of resetting or overwriting it.

Make exactly one documentation change to the existing
`docs/agents/ci-passed.md`: append this checkpoint, substituting the confirmed UTC
window and B. Do not claim the merge or normal CI recovery has already happened.

```markdown
## Main branch migration checkpoint

The default branch was renamed to `main` during <UTC window>, preserving
commit <B>. This documentation-only change is the separately reviewed bootstrap
checkpoint. Its merged GitHub PR record, followed by fresh CI, provides the
post-rename evidence required by the controlled bootstrap procedure above.
```

Review `git diff`, confirm only that file changed, and commit with DCO sign-off
using the author's configured Git identity. Push that branch to the author's
fork and open a **draft** PR targeting upstream `main`. Use the following body;
fill in H after committing, in the PR body rather than in the commit itself.

```markdown
Documentation-only bootstrap for the coordinated master-to-main rename.

- Cutover tracking record / UTC window: <link and window>
- Frozen upstream main base B: <full SHA>
- Reviewed bootstrap head H: <full SHA>
- Independent reviewer: <name; pending until confirmed>
- Authorized merge actor: <name; pending until confirmed>
- Existing narrow authority verified: <evidence; pending until confirmed>
- DCO and fresh applicable constituent/external checks for H: <evidence>
- ci-passed pending reason: <evidence showing only missing main arrival for B>

This PR does not authorize its own exceptional merge. Record explicit approval
for this exact PR/H/B separately before the authorized actor acts. Any changed
SHA, different pending reason, failed check or missing authority blocks it.
Keep normal merges and publications frozen until ordinary CI is proven on the
resulting main tip. No checks may be disabled or success statuses fabricated.
```

Move out of draft only for the reviewed, explicitly authorized bootstrap step.
Follow the canonical controlled-bootstrap procedure for the exceptional merge;
there is no general-purpose bypass command to run as part of this worksheet.

### 4. Verify the resulting merge and normal recovery

Set `KFP_BOOTSTRAP_PR` to the actual bootstrap PR number and `KFP_HEAD_SHA` to the
reviewed H. Before the exceptional merge, read the PR head and the live main tip
again and compare them with H and B. The actor must abort on either mismatch and
pin H in the merge operation as required by the canonical procedure.

Immediately before the exceptional merge, these read-only checks must pass:

```bash
: "${KFP_BOOTSTRAP_PR:?Set the reviewed bootstrap PR number}"
: "${KFP_HEAD_SHA:?Set the reviewed bootstrap head SHA}"
gh api --hostname github.com --method GET "repos/$KFP_REPO/pulls/$KFP_BOOTSTRAP_PR" > "$KFP_EVIDENCE/bootstrap-reviewed.json"
jq -e --arg head "$KFP_HEAD_SHA" \
  '.state == "open" and .draft == false and .base.repo.full_name == "kubeflow/pipelines"
   and .base.ref == "main" and .head.sha == $head' "$KFP_EVIDENCE/bootstrap-reviewed.json"
gh api --hostname github.com --method GET "repos/$KFP_REPO/git/ref/heads/main" > "$KFP_EVIDENCE/main-before-bootstrap.json"
jq -e --arg base "$KFP_BASE_SHA" \
  '.ref == "refs/heads/main" and .object.sha == $base' "$KFP_EVIDENCE/main-before-bootstrap.json"
```

These reads do not lock the branch. The freeze, pinned-head merge and existing
actor authority are all still required. Recheck fresh constituent CI, review/DCO
and the precise pending reason immediately before the actor acts.

After the actor reports completion, collect authoritative evidence:

```bash
: "${KFP_BOOTSTRAP_PR:?Set the reviewed bootstrap PR number}"
: "${KFP_HEAD_SHA:?Set the reviewed bootstrap head SHA}"
gh api --hostname github.com --method GET "repos/$KFP_REPO/pulls/$KFP_BOOTSTRAP_PR" > "$KFP_EVIDENCE/bootstrap-merged.json"
gh api --hostname github.com --method GET "repos/$KFP_REPO/git/ref/heads/main" > "$KFP_EVIDENCE/main-bootstrap.json"
KFP_BOOTSTRAP_SHA="$(jq -er '.object.sha' "$KFP_EVIDENCE/main-bootstrap.json")"
jq -e --arg head "$KFP_HEAD_SHA" --arg tip "$KFP_BOOTSTRAP_SHA" \
  '.merged == true and .base.repo.full_name == "kubeflow/pipelines"
   and .base.ref == "main" and .head.sha == $head
   and (.merged_at | fromdateiso8601 > 0) and .merge_commit_sha == $tip' \
  "$KFP_EVIDENCE/bootstrap-merged.json"
```

**Expected:** authoritative main merge record and current tip C agree. **Stop:**
any missing, mismatched or ambiguous record. Keep the freeze while checking fresh
CI executions created strictly after the bootstrap merge and any retarget event.
A rerun of an older execution does not qualify. Refresh the cutover PR against C
and require normal `ci-passed` success; verify stale-green revocation before
unfreezing. Deploy and verify the external periodic changes only once main exists.

### 5. Rollback decision

If bootstrap or recovery fails, keep the freeze and have the rollback owner
coordinate all affected services. Preserve B, H, C, PR records and configuration
revisions first. Revert deployed periodic references before removing main, and
reconcile any landed cutover changes and source-mirror writes. A local rehearsal's
branch reset is **not** a production rollback command: do not force-push away
landed commits. The owner must approve how to preserve or revert them and restore
the intended default branch. Verify protections, PR bases, service configuration
and fresh CI again before resuming. If any state is unclear, stop rather than
relaxing the validator or creating a competing master branch.
