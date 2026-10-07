# Completing dependency updates

Dependabot proposes source versions. Some source versions also own checked-in
exports, generated workflows, image references, or compiler pins. The completion
workflow supplies those mechanical outputs; CI, source review, DCO, and the
normal merge policy still decide whether the resulting update can merge.

## Dependency ownership

- Root `pyproject.toml`, `sdk/python/pyproject.toml`, and `uv.lock` belong to the
  root uv workspace. pip manages only the independent Python projects.
- Root and SDK `requirements.txt` files are generated, excluded from direct uv
  updates, and exported by `.github/resources/scripts/export_python_requirements.sh`.
  Run that helper from the repository root after `uv lock`. CI and release use
  the same frozen, no-hashes exports with uv 0.10.3.
- Lock-only transitive updates remain eligible. A generated-requirements-only
  proposal is rejected instead of exporting an old lock over the intended bump;
  ask Dependabot to recreate it against the source ownership configuration.
- React, ReactDOM, and their typings have both version-update and security-update
  groups. Grouping does not waive peer compatibility or application tests.
- Go builder/compiler updates reuse `update_go_version.py`. Proposed immutable
  digests are retained and validated on Linux AMD64 and ARM64; other flavors and
  tracked modules are synchronized. Source changes outside this bounded policy
  require manual completion.
- Argo module or controller/executor proposals reuse `sync_argo_versions.py`
  with the full scope. The compatibility lane remains intact. Major migrations
  and changes requiring a different Go compiler are rejected.
- The gh-aw source compiler pin is `.github/actions/setup-gh-aw/action.yml`.
  Dependabot updates its `setup-cli` SHA/comment. Completion verifies the upstream
  release tag, installs that exact compiler through the trusted wrapper, and
  regenerates the workflow and action locks from unchanged trusted Markdown.
  Generated `github/gh-aw-actions/setup` refs are excluded from independent bumps.

## Completion workflow and credentials

`dependabot-completion-request.yml` is an unprivileged `pull_request` signal. It
checks out no code and uploads no files. `dependabot-completion.yml` responds on
`workflow_run`, using scripts and actions from its trusted workflow revision.

The read-only generation job identifies a unique, open, same-repository
Dependabot branch targeting master, verifies the bot commits, and fetches the
exact head. Only supported input deltas are processed. No proposal Makefile,
script, local action, or build backend runs. Generator outputs are stored as a
bounded JSON artifact. The publisher runs on a separate runner, treats that
artifact only as data, checks its event identity and output paths, and repeats
the live proposal checks before committing with GitHub's atomic
`expectedHeadOid` condition. A moved head or base aborts publication.

Install a dedicated GitHub App on this repository with **Contents: write**,
**Pull requests: read**, and **Workflows: write** (for generated workflow files).
Configure repository variable `DEPENDABOT_COMPLETION_APP_ID` and Actions secret
`DEPENDABOT_COMPLETION_APP_PRIVATE_KEY`. The App must not have branch-protection
bypass or administration access. Its token is minted only on the isolated writer
runner and scoped to this repository. The App identity is used for the DCO
sign-off. No personal token or `GITHUB_TOKEN` write fallback is supported:
App-authored commits trigger normal fresh-head CI, whereas ordinary workflow
`GITHUB_TOKEN` pushes would suppress it.

Until that App is configured, a needed completion produces an artifact and an
explicit setup error without changing the branch. After configuring it, rerun
**all jobs** of the completion workflow; publication-only reruns cannot select a
previous attempt's artifact. After a concurrent base change, rerun all jobs too.
The workflow never approves PRs, removes holds, merges, or retries failed tests.

Generated commits contain `[dependabot skip]`. Dependabot can discard these
reproducible outputs when it rebases; the next bot head is completed again. A
completion commit itself is ignored, preventing recursion. Human-edited bot
branches are left for manual review. Unsupported mixed updates and deliberate
application migrations remain manual work.

## Compatibility fixes and rollout

The frontend explicitly includes `es2022.array` declarations for its existing
`Array.at` calls. Run `node --test frontend/scripts/typescript-libs.check.mjs`
after installing frontend dependencies. The obsolete `flake8-black` plugin is
removed; Black and flake8 remain independent tools.

The multi-user profile controller has its own Python/botocore image instead of
relying on tooling incidentally installed in `alpine/k8s`. Its requirements are
hash-pinned and updated by the independent pip entry. The existing native image
build and publication workflows include it, and the Docker build exercises its
actual HTTP startup. Multi-user CI loads the locally built image. Publish the
master image before deploying the new development manifest outside CI; tagged
release tooling includes it in subsequent releases.

Local verification covers real uv exports and a lock-only transitive update,
real gh-aw compiler regeneration/idempotence, bounded generator and publication
failures, and profile-controller HTTP startup. A representative hosted direct
and transitive/security Dependabot update plus fresh-head CI is the final rollout
check after these default-branch workflows and the App configuration are active.
