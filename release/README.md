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
branch. Kubernetes API documentation is included in the SDK docs.

For `--fork-remote`, you can pass either a full remote URL or just your GitHub username. `kfpr`
expands `droctothorpe` to `https://github.com/droctothorpe/pipelines.git`.

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

Single-step commands do not update the checkpoint unless `--done` is passed.
`update-version-tags` cuts `<version>-update-version-tags` from the release branch
before committing version changes. When SDK release steps are enabled, this same PR also updates
SDK package versions, requirements, docs versions, and `sdk/RELEASE.md`.
The unified workspace requires uv: release updates regenerate the Python bindings,
lockfile, and two requirements exports, then build only `kfp`. Bundled modules
share `kfp.version`, independently of backend `VERSION`. This also applies to
SDK-only releases. Older checkout layouts retain their legacy build path.

Requirements exports retain editable workspace packages and pinned dependencies,
but omit hashes because pip cannot hash editable sources. Consume these files
with `pip install -r <export>` from the repository root. The uv lockfile retains
dependency hashes; CI checks both export freshness and actual pip resolution.

Maintenance releases continue to dispatch the publishing workflow from their
release branch. The current `publish-packages.yml` also accepts pre-uv tags:
tool setup is independent of the selected checkout, and package builds use that
tag's Makefiles or a setuptools-compatible source-path build. Use `dry_run=true`
to build and validate distributions without uploading them to PyPI.
For unified tags, `all` selects only `kfp`; explicitly selecting a retired
distribution is an error. PyPI polling and new SDK release notes cover only
`kfp`. The [migration instructions](../sdk/python/README.md#migrating-from-the-split-packages)
remove the three legacy file owners before installing the unified wheel.

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
defaults. If the API call fails, `kfpr` asks whether to fall back to the manual checklist.

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
