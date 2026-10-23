# kfpr release CLI

`kfpr` is the Kubeflow Pipelines release automation CLI. It can run the full checkpointed release
flow or any individual release step.

## Quick start

From the repository root:

```bash
python3 -m pip install -e release
kfpr --help
kfpr doctor --release-type patch --version 3.2.1 --fork-remote USER
kfpr run --dry-run --state-file /tmp/kfp-release-state.json
```

Without installing, run from source:

```bash
PYTHONPATH=release python3 -m kfpr.cli --help
```

## Full release flow

```bash
kfpr run \
  --release-type patch \
  --version 3.2.1 \
  --fork-remote USER \
  --patch-prs 12345,12346 \
  --include-backend \
  --include-sdk
```

`run` stores prompt answers and completed steps in `release-state.json` by default. Pass
`--state-file` to use a different checkpoint file. If the checkpoint already exists, `run` prints
the saved status and asks before resuming; pass `--force` to resume without that prompt.
For major and minor releases, pass `--release-source-branch BRANCH` to skip the source
branch prompt.

Patch releases update the persistent `release-<major>.<minor>` documentation
branch. Unified SDK releases include Kubernetes API documentation in the SDK
docs. Split-package releases also create or reuse
`kfp-kubernetes-<major>.<minor>` and update the separate ReadTheDocs project.

For `--fork-remote`, you can pass either a full remote URL or just your GitHub username. `kfpr`
expands `droctothorpe` to `https://github.com/droctothorpe/pipelines.git`.

## Preparing generated release artifacts

`update-version-tags` supports ordinary clones and linked Git worktrees. For
worktrees, it mounts Git metadata read-only into the release container so
changelog generation can resolve the previous release.

Backend preparation regenerates the API versions present on the release branch,
pipeline-spec/cache-key Go bindings, and Kubernetes extension Go bindings. For
legacy split-package SDK releases, it generates Python bindings before building
all four wheels (`kfp-pipeline-spec`, `kfp-server-api`, `kfp`, and
`kfp-kubernetes`). Before committing, it checks wheel versions and required Python
modules to catch successful builds with missing generated sources.

Legacy dependency resolution reuses the existing `requirements.txt` pins with
`pip-compile --output-file`; it does not truncate the lockfile before resolving.
Review the resulting lockfile diff, including environment markers, on each
supported Python version. If changelog filtering produces an empty release
section, kfpr includes commit subjects from the release range for review.

## Qualification and publication gates

Preparation creates the version PR without creating or pushing release tags.
The explicitly recorded `version_pr_url` may identify a follow-up fixes PR on a
differently named branch. It must still belong to `kubeflow/pipelines`, be merged
into the intended release branch, and have successful CI. Existing recorded head
and merge SHAs cannot change silently; selecting a new release commit requires
fresh qualification for that commit.
`merge-version-pr` waits for CI and a maintainer merge; kfpr does not merge the PR.
It records the tested PR head and merged commit. The next checkpoint,
`qualify-release`, dispatches the read-only `release-qualification.yml` workflow
on upstream `master` for that exact commit. All builds, installed-wheel smoke
tests and vulnerability scans run in CI:

- Source dependencies are scanned with checksum-pinned OSV Scanner 2.5.0 before
  wheel generation. The report must cover the root Go module and frontend lockfile.
- The SDK job builds release wheels once, installs them in fresh environments for
  every Python minor advertised by the selected source commit, checks distribution
  versions and dependency consistency, and compiles a pipeline using Kubernetes
  configuration. OSV scans the exact installed distribution inventory from each
  environment, including local release packages; missing package coverage fails.
- Native AMD64 and ARM64 jobs build every image/platform combination from that
  source commit's release matrix, including its exclusions. Each image is exported
  to a Docker archive whose config digest must match the immutable built image ID.
  OSV scans that archive directly without registry fallback.
- Every scan retains complete JSON package and vulnerability data. Empty scanner
  configuration and explicit all-package/all-vulnerability flags prevent ignore
  settings from hiding findings. SARIF alone is insufficient for qualification
  because it does not supply the full package-specific fix information.

Qualification requires the workflow merged on upstream `master` and an
authenticated `gh` session authorized to dispatch it. There is no local
qualification mode or local Python-interpreter override. Local Trivy, pip-audit,
Docker and per-minor Python installations are not required for qualification.
SDK-only releases skip image builds; backend-only releases skip installed-wheel
tests. Source scanning always runs. The workflow does not publish images or tags.

The client downloads the complete attempt-bound artifact and checks its source
and workflow commits, request and run identities, selected scope, full image and
Python inventories, raw reports and retained wheel checksums. It also rechecks the
live workflow run before accepting evidence. Failed lanes retain diagnostics;
incomplete evidence blocks publication. Rerunning a workflow attempt invalidates
its earlier evidence, so run `qualify-release` again to obtain fresh evidence.

The default security policy is zero scanner-reported fixable vulnerabilities,
across all severities. Findings without a published fix remain in the report. Missing tools,
scanner errors, incomplete dependency coverage and unrecognized reports block
publication. A scanner-reported fix requiring a newer Python still blocks; kfpr
does not silently waive an advisory or change supported Python versions.

For explicitly accepted findings, select `--security-exceptions /path/to/exceptions.json`.
The [exception file](SECURITY_POLICY.md) identifies each exact artifact, advisory,
package and installed version, with a justification and UTC expiry date. This
selects `approved-exceptions` and binds the file contents to qualification.
Publication checks those contents and expiry again. All findings remain in the
report; scanner errors and incomplete coverage always block. Selecting
`--security-policy zero-fixable` clears the exception selection and requires fresh
qualification. The historical `2.17.3-approved-exceptions-v1` policy remains
available only to verify its original release evidence.

Wheel environments remove bootstrap-only pip/setuptools after installation and
dependency checks, before exercising SDK imports and pipeline compilation.
Packages reachable through declared runtime dependencies are retained. Final
dependency freezes and removed installer names are saved for audit.

Qualification prints a persistent evidence directory with raw scanner reports,
wheel hashes, dependency freezes, compiled pipelines and logs. The checkpoint
records the summary path and SHA256, release version, merged SHA and selected
scope. Evidence expires after 24 hours; rerun `qualify-release` to refresh it.
Failed requalification invalidates prior evidence. Generated sources must match
the merged commit or qualification stops for a new reviewed version update.

Tag creation, image/package publication and GitHub release creation each recheck
live PR CI, qualification evidence and the qualification workflow run, including
when invoked individually. Tags point to
the qualified merge commit. A stale unpublished local tag can be corrected;
published tags are never replaced. Workflow dispatch and completion checks are
bound to the same source commit. Patch image releases leave the global `latest`
tag unchanged. `--dry-run` describes gates without generating
evidence or publishing. The image publication workflow may rebuild from that
source; qualification is not a claim that published image digests are
identical. Keep the workflow's post-publication validation checkpoint.

## Individual steps

List the selected flow:

```bash
kfpr steps --release-type patch --include-backend --no-include-sdk
kfpr steps --diagram
```

Run one step without marking it complete:

```bash
kfpr run update-version-tags \
  --release-type patch \
  --version 3.2.1 \
  --fork-remote USER \
  --patch-prs 12345,12346 \
  --dry-run
```

Mark a step complete after it succeeds:

```bash
kfpr run update-version-tags \
  --release-type patch \
  --version 3.2.1 \
  --fork-remote USER \
  --patch-prs 12345,12346 \
  --done
```

Single-step commands mark completion only with `--done`. Version PR identity and
qualification evidence are saved immediately so publication guards also work when
running individual steps.
`update-version-tags` cuts `<version>-update-version-tags` from the release branch
before committing version changes. When SDK release steps are enabled, this same PR also updates
SDK package metadata, docs versions, and `sdk/RELEASE.md`.
The unified workspace requires uv: release updates regenerate the Python bindings
and lockfile, then build only `kfp`. Bundled modules share `kfp.version`,
independently of backend `VERSION`. This also applies to SDK-only releases.
Older checkout layouts retain their legacy build and tracked-export behavior.

Modern checkouts do not commit workspace requirements exports. For optional pip
consumers, run `.github/resources/scripts/export_python_requirements.sh OUTPUT_DIRECTORY`
to generate portable, frozen dependency-only artifacts; install the SDK wheel
separately. CI validates lock consistency, frozen installation, artifact resolution,
and a clean wheel install. See [dependency maintenance](../docs/agents/dependabot.md).

Publication dispatches the workflow from the immutable release tag, with source
pinned to the qualified merged commit. The current `publish-packages.yml` also accepts pre-uv tags:
tool setup is independent of the selected checkout, and package builds use that
tag's Makefiles or a setuptools-compatible source-path build. Use `dry_run=true`
to build and validate distributions without uploading them to PyPI.
For unified tags, `all` selects only `kfp`; explicitly selecting a retired
distribution is an error. After SDK tag creation, the current CLI inspects
`sdk/python/kfp` at that tag through the GitHub API. This selects installation
instructions, PyPI polling/retries, and documentation projects from the target
layout, regardless of which checkout supplied the running CLI. Missing or
unreadable tag contents stop the step instead of guessing a layout.

Unified tags publish and poll only `kfp`; their release notes include the
[migration instructions](../sdk/python/README.md#migrating-from-the-split-packages)
to remove legacy file owners before installing the unified wheel. Split tags
(including the supported 2.18 line) retain all four package installation
instructions and PyPI checks, without uninstalling Kubernetes helpers.
`create-kfp-kubernetes-docs-branch` remains a checkpoint and individual command,
but skips branch changes for unified tags. For split tags, it requires a clean
working tree before any branch operations, including when resuming after
preflight; commit or stash staged, unstaged, and untracked changes first.
Unified-tag no-ops and dry runs do not require a clean working tree.
Both automated ReadTheDocs updates and the manual fallback select the same
projects. Status output and dry runs
show the conditional legacy checklist without querying an SDK tag that may
not exist yet.

### Architecture support and release validation

The [supported-platforms policy](../docs/operator-guides/supported-platforms.md)
commits to Linux AMD64 and ARM64 support starting with backend 3.0. The image
publication workflow for that release line verifies shared-tag resolution on native
AMD64 and ARM64 runners, then runs the native ARM64 installation/pipeline smoke
against the same run's immutable published image indexes. Validation follows image
publication; a failed validation blocks release completion, not the initial upload.
Dry-run publication builds images without running checks against published tags.

For backend 3.0 and later, the existing `create-backend-release` checkpoint asks the
release manager to review the successful release run and retain its published-index,
native image-validation, and smoke artifacts before creating the GitHub release.
Release notes and final communications link to the policy. A successful master smoke is not a substitute
for validating the release images. Existing checkpoint IDs are unchanged; when
resuming a release whose backend-release step was already marked done, review the
new checklist explicitly rather than recreating an existing release.

The remaining **2.18 release does not acquire ARM64 support or these validation
requirements**. Use the current CLI with version `2.18.0` (or a 2.18 patch version):
it dispatches `image-builds-release.yml` from `release-2.18`, using that branch's
existing inputs and image inventory. Do not dispatch the master workflow for 2.x
or backport the 3.x image inventory. SDK-only releases also have no backend-image
architecture checkpoint. All workflow watchers stop on failure, including on 2.x.

These checks prepare the tooling for future releases; no release candidate needs
to be cut now. The CLI still accepts final `MAJOR.MINOR.PATCH` versions only;
prerelease image validation can use the release workflow directly.

If you complete a step outside `kfpr` (for example, manually creating an already-existing
release branch), mark that step done before resuming:

```bash
kfpr done prepare-release-branch --state-file release-state.json
kfpr run --state-file release-state.json
```

Use the exact step ID from `kfpr steps` or the list below.

## Maintainer container architectures

The current `kfp-api-generator` and `kfp-release` tooling images are built and
tested for Linux AMD64 and ARM64. Their shared `:master` (and subsequent release
branch) tags select the Docker host's native architecture, including the Linux
VM used by Docker on Apple Silicon. No architecture-specific tag or forced
`--platform` is needed for normal use.

Tool versions are identical on both architectures. CI runs native API/client and
Python-package generation with both images, then exercises manifest versioning
and changelog preparation in a disposable repository. It compares generated
source bytes across architectures before publishing the shared tooling tags;
PR builds do not publish. The release image uses the generator built in the
same job, so a stale registry tag cannot supply its parent image.

This concerns the machine running maintainer tools, not the supported platforms
of a KFP installation. Existing 2.x branches are not changed or backported, and
the remaining 2.x release does not acquire ARM runtime support. The AMD64 tools
retain their pinned versions. These image changes do not dispatch a release or
alter the release CLI's existing image-tag and branch-selection behavior.

## Recovery helpers

```bash
kfpr status --state-file release-state.json
kfpr clear --state-file release-state.json
kfpr validate-state --state-file release-state.json
kfpr run watch-publish-images --state-file release-state.json
kfpr done update-version-tags --state-file release-state.json
kfpr reset-step update-version-tags --state-file release-state.json
```

`status` prints saved answers, completed steps, the next step, the resume command, and the active
manual checklist when the next step needs external confirmation.
`merge-version-pr` watches PR CI and fails as soon as a reported gate fails; after all reported
gates complete successfully, it continues waiting for the PR to merge.
`watch-publish-images` watches the latest image publication workflow for the saved release branch
without dispatching a new workflow run.

`confirm-rtd` prompts for a Read the Docs API token, keeps it only in process memory, and uses it
to activate release versions, trigger builds, wait for successful builds, and update project
defaults for major and minor releases. Patch releases build and activate their documentation
without changing project defaults; the manual fallback preserves defaults too. If the API call fails, `kfpr` asks whether to fall back to the manual checklist.

## Commands

```text
run
doctor
status
clear
validate-state
done
reset-step
steps
run preflight
run prepare-release-branch
run prepare-patch-branch
run cherry-pick-prs
run merge-cherry-pick-pr
run update-version-tags
run merge-version-pr
run publish-images
run create-sdk-tag
run publish-sdks
run create-kfp-kubernetes-docs-branch
run confirm-rtd
run create-sdk-release
run create-backend-release
run sync-master
run confirm-website-and-slack
run watch-publish-images
```

For the KFP 3.0 release, the `confirm-website-and-slack` checkpoint also requires the Kubeflow
website installation and upgrade documentation to announce that Argo Workflows 3.x is unsupported,
and requires the GitHub release notes to identify that removal as a breaking change.

Image `latest` and the backend GitHub latest designation follow numeric SemVer,
not release type. Before each publication, kfpr reads every page of published
upstream releases and promotes the target only when no newer stable backend
release exists. Drafts, prereleases and `sdk-*` tags do not affect the comparison.
Thus 2.17.3 can become latest after 2.17.2, but cannot replace stable 2.18.0 or 3.0.0.
Malformed or unavailable discovery stops publication; dry runs perform no query.
SDK GitHub releases always use `--latest=false` to preserve the shared backend designation.

Maintenance patches still leave master version files, changelog and Read the Docs
defaults unchanged. The final website checklist preserves its existing latest-version
pointer; these separate maintenance safeguards are not part of latest selection.

SDK publication checks PyPI before each upload. Split SDK releases publish missing
packages in dependency order: pipeline spec, server API, KFP, then Kubernetes.
Each package must become available before its dependents are dispatched. Unified
SDK releases publish only KFP. After a workflow failure, resume `publish-sdks`;
versions already available on PyPI are skipped rather than uploaded again.

### Promote an already published release

```sh
kfpr promote-latest --version 2.17.3 --dry-run
kfpr promote-latest --version 2.17.3
```

This recovery command dispatches `release-promote-latest.yml` in the upstream
repository and waits for its result. Merge the workflow upstream before using it.
CI uses its own package-write credentials; maintainers need workflow-dispatch
access, not local registry credentials. The workflow validates the stable release
and its full source-defined image/platform inventory, copies each existing image
index by digest to `latest`, and verifies the result and unchanged versioned tag.
It never rebuilds images or uploads SDK packages. Evidence includes previous and
promoted digests, including partial progress if a registry operation fails.
Only releases without a newer published stable backend release may be promoted.
