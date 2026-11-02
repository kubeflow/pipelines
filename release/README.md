# kfpr release CLI

`kfpr` is the Kubeflow Pipelines release automation CLI. It can run the full checkpointed release
flow or any individual release step. This is the canonical maintainer runbook for
backend, Python SDK, and Kubernetes integration releases. For published notes and
upgrade guidance, start at [Releases](../RELEASE.md).

## Prerequisites and access

Use a clean, dedicated checkout with the intended release branch and comparison
tags available locally. The flow switches branches and creates version commits;
keep unrelated work in another checkout. Configure your Git name and email for
DCO sign-offs before preparing a release.

Install Python 3.11 or newer, Git, GitHub CLI (`gh`), Docker, and `sed`. SDK
preparation also requires `uv` on branches with `uv.lock`, or `pip-compile` on
older branches. Generated API and documentation build steps can require the
branch's Node.js, Go, Java, `jq`, and `yq` tools; follow the
[development guide](../developer_guide.md) for the selected branch. Image
publication and qualification run in GitHub Actions.

Authenticate with `gh auth login`, then check `gh auth status`. The release manager
needs access to push release branches/tags and create GitHub releases in
`kubeflow/pipelines`, dispatch and inspect publication workflows, and push the
version/cherry-pick branches to their fork. Package and image publication use
workflow-managed credentials. Read the Docs activation requires project access
and an API token (or the manual checklist); do not put tokens in checkpoint files.

The CLI uses a remote named `upstream` for the canonical repository. Inspect
existing remotes before adding one:

```bash
git remote -v
# Only when the upstream remote is absent:
git remote add upstream https://github.com/kubeflow/pipelines.git
git fetch upstream --tags
gh auth status
```

`--fork-remote USER` means `https://github.com/USER/pipelines.git`; it is not a
Git remote alias. A full fork URL is also accepted. Fetch the selected branch
and tags before previewing notes, and run `kfpr doctor` after installation.
The doctor checks local tools, authentication, and checkout cleanliness; it does
not prove that every remote publishing permission is available.

## Quick start

Use Python 3.11 or newer. From the repository root:

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

## Choose the release scope

Backend and SDK versions are independent. In the examples below, replace `USER`,
versions, and patch PR numbers with the intended release values. Use a separate
checkpoint file for each release. First append `--dry-run` to inspect the selected
flow; remove it to execute the release and follow its review checkpoints.
A dry run does not create branches, merge PRs, or publish artifacts.
The full-flow `kfpr run --dry-run` plans every remaining step and defers release
note previews until the release refs and contents exist. Use the individual
[preview commands](#review-release-notes-before-publishing) afterward to generate
and review Markdown from the actual release range.

| Release artifact | Version/tag | Branch |
| --- | --- | --- |
| Backend images and GitHub release | `MAJOR.MINOR.PATCH` | `release-MAJOR.MINOR` |
| Python packages and SDK GitHub release | `sdk-MAJOR.MINOR.PATCH` tag; packages use `MAJOR.MINOR.PATCH` | `release-MAJOR.MINOR` for the SDK version |
| Kubernetes integration docs | Included in the unified SDK; separate for split-package maintenance releases | SDK docs, or legacy `kfp-kubernetes-MAJOR.MINOR` |

For major releases the release branch is `release-MAJOR.0`. Current SDK releases
publish the unified `kfp` distribution. Split-package maintenance branches publish
`kfp`, `kfp-pipeline-spec`, `kfp-server-api`, and `kfp-kubernetes` at the SDK version. A combined run uses one version for both backend and SDK;
when the versions differ, use separate runs and checkpoints.

### Backend-only release

```bash
kfpr run \
  --release-type patch --version 3.2.1 --fork-remote USER \
  --patch-prs 12345,12346 \
  --include-backend --no-include-sdk \
  --previous-release 3.2.0 \
  --state-file /tmp/kfp-backend-3.2.1.json
```

This selects backend version preparation, image publication and qualification,
backend GitHub notes, and the applicable website/announcement checkpoints.

### SDK-only release

```bash
kfpr run \
  --release-type patch --version 3.2.1 --fork-remote USER \
  --patch-prs 12345,12346 \
  --no-include-backend --include-sdk \
  --previous-sdk-release sdk-3.2.0 \
  --state-file /tmp/kfp-sdk-3.2.1.json
```

This selects Python package preparation, SDK tagging, package publication,
Read the Docs updates, and the SDK GitHub release. It does not publish backend
images or change the backend `VERSION`.

Patch releases update the persistent `release-<major>.<minor>` documentation
branch. Unified SDK releases include Kubernetes API documentation in the SDK
docs. Split-package releases also create or reuse
`kfp-kubernetes-<major>.<minor>` and update the separate ReadTheDocs project.

### Combined backend and SDK release

```bash
kfpr run \
  --release-type patch --version 3.2.1 --fork-remote USER \
  --patch-prs 12345,12346 \
  --include-backend --include-sdk \
  --previous-release 3.2.0 --previous-sdk-release sdk-3.2.0 \
  --state-file /tmp/kfp-combined-3.2.1.json
```

`run` stores prompt answers and completed steps in `release-state.json` by default.
If the checkpoint already exists, it prints the saved status and asks before
resuming; pass `--force` to resume without that prompt. Review notes again at
publication even when resuming a checkpoint.

For major or minor releases, change `--release-type`, omit `--patch-prs`, and
pass `--release-source-branch BRANCH` to skip the source branch prompt. Leaving
the prompt blank resolves the upstream repository default branch at preparation
time. Explicit or previously saved source branches remain unchanged. For patch
releases, list the PRs to cherry-pick or follow the prompt when the desired changes
are already merged. Patch releases also update the persistent SDK documentation branch; split-package
releases additionally update the Kubernetes documentation branch.

## Select explicit release-note ranges

Supply `--previous-release` for the backend comparison tag and
`--previous-sdk-release` for the SDK comparison tag. The two bases can differ;
only the selected release components need a base. If a required base is omitted,
the CLI asks for it. The answer is persisted when a full run saves its checkpoint
or an individual step completes with `--done`. It never selects the globally
newest SDK tag. Choose the previous published version on the release line you are
preparing, including when a newer minor version has already shipped.

Before generating notes in an individual step, fetch the intended release branch
and its tags:

```bash
git fetch upstream --tags
```

During preparation, generation defaults to
`refs/heads/<version>-update-version-tags`, the version branch checked out by
`kfpr`. Before editing version files, the CLI verifies that the selected target
matches the checked-out commit. At publication it uses the local release tag
when available, otherwise `refs/remotes/upstream/release-<major>.<minor>`.

Use `--release-target REF` to explicitly select another local ref. For preparation,
that ref must resolve to the checked-out commit. For publication, its resolved SHA
must match the authoritative GitHub release tag if that tag already exists. SDK
publication also requires that the SDK tag already exists on GitHub; complete
`create-sdk-tag` first. The target must also match the qualified release commit recorded in the checkpoint.
A mismatch stops publication. This flag controls the
note-generation and GitHub-release target; it does not select the source branch
for release preparation (`--release-source-branch` does that).

The CLI resolves each base and target to immutable commit SHAs, validates the
range, and includes the resolved comparison link in the notes. Divergent release
branches are supported when they share history; reversed and unrelated histories
are rejected. Review that link
and the selected target before publishing. Git errors and invalid ranges stop
generation rather than silently producing an empty changelog.

## Preparing generated release artifacts

`update-version-tags` supports ordinary clones and linked Git worktrees. Changelog
generation reads Git history in the host CLI; generated artifacts use the release
container.

Backend preparation regenerates the API versions present on the release branch,
pipeline-spec/cache-key Go bindings, and Kubernetes extension Go bindings. For
legacy split-package SDK releases, it generates Python bindings before building
all four wheels (`kfp-pipeline-spec`, `kfp-server-api`, `kfp`, and
`kfp-kubernetes`). Before committing, it checks wheel versions and required Python
modules to catch successful builds with missing generated sources.

Version preparation is not a dependency upgrade. Legacy resolution reuses the
existing `requirements.txt` pins with `pip-compile --output-file`; uv releases
reuse `uv.lock`. The four release packages may advance together, but third-party
package additions, removals, or version changes stop preparation before the
resolved SDK is built. For uv, changing a package source also stops preparation.
The CLI restores existing lockfiles on failure or interruption, reports the drift,
and requires a separately reviewed dependency update before retrying. It does not
automatically accept a newer dependency just because the resolver needs it.
Review environment markers and generated exports as well; these checks do not
replace installed-wheel qualification on each supported Python version.

Authored `CHANGELOG.md` Unreleased guidance and SDK Current Version notes are
carried into the release alongside generated changes. SDK preparation is
idempotent. Backend preparation saves the exact prepared section and comparison
SHAs in the checkpoint before writing it. It writes the changelog to a temporary
file and atomically replaces the original only after the write succeeds, so a
partial write leaves existing guidance and history intact. Retrying with that
checkpoint preserves curated guidance without duplicating it. Keep the checkpoint
across interruptions. If the range changes or a prepared backend section is edited,
preparation stops without overwriting the notes; review the candidate and its
notes explicitly rather than deleting guidance to make generation pass. If
changelog filtering produces no user-facing entries, preparation stops so the
maintainer can verify the selected range.

## Continue 2.18 from an existing version PR

Use the current default-branch CLI to finish a maintenance release, even when the target
branch contains older release automation. Install a non-editable copy from the
reviewed default-branch checkout into a separate environment; this keeps switching the
release checkout to `release-2.18` from replacing the running tool's source:

```bash
python3.11 -m venv /tmp/kfpr-tools
/tmp/kfpr-tools/bin/pip install /path/to/current-default-branch-checkout/release
```

For 2.18.0, the version PR
[#15217](https://github.com/kubeflow/pipelines/pull/15217) merged as
`97561db87ad2b82c7dc6b8a29284c1f0ca0fd7e7`. Do not regenerate that PR or create
a tag merely to initialize kfpr. In a clean dedicated release
checkout, configure `upstream`, fetch the release branch and tags, then create a
new checkpoint outside the checkout (replace the fork URL with your actual fork):

```json
{
  "answers": {
    "release_type": "minor",
    "version": "2.18.0",
    "fork_remote": "https://github.com/USER/pipelines.git",
    "release_source_branch": "release-2.18",
    "include_backend": true,
    "include_sdk": true,
    "previous_release": "2.17.3",
    "previous_sdk_release": "sdk-2.17.3",
    "version_pr_url": "https://github.com/kubeflow/pipelines/pull/15217"
  },
  "completed_steps": ["prepare-release-branch", "update-version-tags"]
}
```

Save it as `/tmp/kfp-2.18.0.json`, then run from that release checkout:

```bash
/tmp/kfpr-tools/bin/kfpr run preflight --state-file /tmp/kfp-2.18.0.json --done
/tmp/kfpr-tools/bin/kfpr run merge-version-pr --state-file /tmp/kfp-2.18.0.json --done
/tmp/kfpr-tools/bin/kfpr run qualify-release --state-file /tmp/kfp-2.18.0.json --done
```

A recorded `version_pr_url` resumes CI/merge verification directly: it does not
push or recreate a version branch, create another PR, or create a tag. A canonical
pending Tide merge-queue status does not block CI verification; actual CI still
has to pass. The merge
checkpoint records the actual tested head and merge commit; do not fill those
SHAs in by hand. Qualification is read-only. Complete the final live acceptance
and residual-finding review before proceeding to the publication steps below.
The default zero-fixable policy still applies; previous release exceptions or
tracker dispositions are not automatic approvals for 2.18.

If a later fix changes the candidate, use a fresh checkpoint with that merged
PR's URL and only the already-completed preparation steps. Do not copy
`release_sha`, `version_pr_head_sha`, qualification receipts, or publication
checkpoints from the previous candidate. Reverify merge/CI and requalify. A
published tag is immutable: selecting a different candidate after publication
requires a new release version, not a forced tag update.

## Qualification and publication gates

Preparation creates the version PR without creating or pushing release tags.
The explicitly recorded `version_pr_url` may identify a follow-up fixes PR on a
differently named branch. It must still belong to `kubeflow/pipelines`, be merged
into the intended release branch, and have successful CI. Existing recorded head
and merge SHAs cannot change silently; selecting a new release commit requires
fresh qualification for that commit.
`merge-version-pr` waits for CI and a maintainer merge; kfpr does not merge the PR.
When reusing its usual version branch without a recorded PR, it pushes local
commits normally before finding the PR; divergent history fails instead of
silently qualifying an older remote head.
It records the tested PR head and merged commit. The next checkpoint,
`qualify-release`, dispatches the read-only `release-qualification.yml` workflow
on the upstream default branch for that exact commit. All builds, installed-wheel smoke
tests and vulnerability scans run in CI:

- Source dependencies are scanned with checksum-pinned OSV Scanner 2.5.0 before
  wheel generation. The report must cover the root Go module and frontend lockfile.
  Source scans audit declared versions without speculative dependency resolution or
  reachability filtering; installed environments and images supply runtime coverage.
  Extraction errors block acceptance, even if OSV returns a findings exit code.
  Unpinned source requirements and local Go replacements without a version remain
  explicitly listed as unaudited declarations in the evidence, not clean packages.
  They cannot satisfy required manifest coverage; installed Python environments
  and image packages still require exact versions and complete inventory coverage.
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

Release dispatch resolves the authenticated upstream repository's default branch
(`master` or `main`) and captures its workflow commit before starting CI. The
matching run must retain that branch, commit, repository, and attempt. Lookup
failures stop dispatch; there is no fallback to the old branch name. Promotion
also requires execution on the current upstream default branch.

The `sync-master` checkpoint keeps its existing name for saved-state compatibility,
but opens its sync PR against the current default branch. Tooling image tags such
as `kfp-release:master` remain unchanged by the Git branch migration. Offline dry
runs show `<repository-default>` for unresolved branch selections and do not save
that placeholder.

Qualification requires the workflow merged on the upstream default branch and an
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
Ubuntu Pro-only fixes are labeled separately from standard Ubuntu fixes: those
findings remain visible residuals requiring support-channel review, and do not
claim an available fix in the image's standard package channel.

Backend qualification derives its external-image inventory from the exact
release commit: all environment and legacy metadata profiles, configured Argo
executor and cache helper where present, the UI/controller TensorBoard defaults,
and separately identified pinned cluster prerequisites. It preserves source inputs and rendered manifests,
scans each distinct image on Linux AMD64, and revalidates complete coverage with
the receipt. Arbitrary workloads, operator overrides and non-default TensorBoard
menu selections are outside this inventory. Findings use the same
fixable-vulnerability policy as first-party images. A scanner with no recognizable
package inventory produces unknown coverage, never a zero-vulnerability result.
Collection retains failed-scan diagnostics and attempts the remaining images;
any incomplete coverage still blocks qualification and publication.
Missing third-party coverage blocks qualification; older receipts must be
regenerated. Scheduled image-comparison reports cannot substitute for this
candidate-bound evidence.

The inventory is derived from the selected source, not a fixed image count.
Release 2.18 now uses the KFP-built TensorBoard runtime and a digest-pinned Alpine
cache helper. TensorBoard belongs to the native image inventory and needs matching
candidate publication records; a source default tag does not prove that the
version-stamped image exists or is publicly pullable. The Alpine helper has APK
package metadata, but qualification must still verify package extraction and scan
the exact selected image. Historical unknown coverage for the previous BusyBox
helper does not describe the new default, and earlier successful candidate scans
do not certify a later merged or version-stamped release.

Image-upgrade compatibility checks are separate evidence. Passing the persistent
SeaweedFS/profile checks or the PostgreSQL/TensorBoard acceptance lanes does not
waive scan findings or authorize publication. Rebuild the selected release source,
verify the published image identities and supported platforms, and regenerate the
complete qualification receipt after the final image and dependency changes.

For explicitly accepted findings, select `--security-exceptions /path/to/exceptions.json`.
The [exception file](SECURITY_POLICY.md) identifies each exact artifact, advisory,
package and installed version, with a justification and either a UTC expiry date
or an exact final release version. This
selects `approved-exceptions` and binds the file contents to qualification.
Publication checks those contents and the calendar or release bound again. All findings remain in the
report; scanner errors and incomplete coverage always block. Selecting
`--security-policy zero-fixable` clears the exception selection and requires fresh
qualification. The historical `2.17.3-approved-exceptions-v1` policy retains its
original release-specific exception set. Every policy requires fresh OSV CI
evidence; historical Trivy/local receipts cannot authorize publication.

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
Release notes and final communications link to the policy. A successful development-branch smoke is not a substitute
for validating the release images. Existing checkpoint IDs are unchanged; when
resuming a release whose backend-release step was already marked done, review the
new checklist explicitly rather than recreating an existing release.

The remaining **2.18 release does not acquire ARM64 support or these validation
requirements**. Use the current CLI with version `2.18.0` (or a 2.18 patch version):
it dispatches `image-builds-release.yml` from `release-2.18`, using that branch's
existing inputs and image inventory. Do not dispatch the development-branch workflow for 2.x
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
in a disposable repository; release-note generation is tested in the host CLI. It compares generated
source bytes across architectures before publishing the shared tooling tags;
PR builds do not publish. The release image uses the generator built in the
same job, so a stale registry tag cannot supply its parent image.

This concerns the machine running maintainer tools, not the supported platforms
of a KFP installation. Existing 2.x branches are not changed or backported, and
the remaining 2.x release does not acquire ARM runtime support. The AMD64 tools
retain their pinned versions. These image changes do not dispatch a release or
alter the release CLI's existing image-tag and branch-selection behavior.

## Review release notes before publishing

### Build a release-wide editorial inventory

Before selecting highlights, create a local evidence bundle from the actual release
range. These commands do not publish, change tags, or update release checkpoints:

```bash
kfpr prepare-editorial --version 3.2.1 --base 3.2.0 \
  --target RELEASE_COMMIT --output /tmp/kfp-3.2.1-editorial
```

Use explicit fetched refs. When SDK and backend baselines or versions differ,
create separate bundles with their own ranges and use the corresponding document.
Never use today's master history to fill gaps in a maintenance release.

The bundle contains `inventory.json`, `backend.md`, `sdk.md`, and `upgrade.md`.
Every selected commit is retained, including nonconventional titles and CI changes.
Consolidated backport subjects are exposed with their parent SHA and body line;
they are **review candidates**, not proof that every mentioned upstream change
shipped. Fenced examples are ignored. Inspect the released implementation before
turning a candidate into a feature or compatibility claim.

Edit only the review fields in `inventory.json`:

- `decision`: `pending`, `include`, or `omit`; explain omissions in `reason`.
- `notes`: reviewed benefits or fixes with `products`, `section`, and `text`.
- `facts`: verified statements shared across documents, with the same fields and
  a `commit` from the range or either boundary. Use this for Python support,
  packaging, defaults, prerequisites, and retained limitations.
- `coverage`: summarize coverage or intentional omission for features, bug fixes,
  security, resilience, compatibility, and operator actions.

For example, an included change can have this note:

```json
{
  "products": ["sdk"],
  "section": "bug_fixes",
  "text": "Explain the previously failing user operation and its corrected behavior."
}
```

Allowed sections are `highlights`, `bug_fixes`, `security`, `resilience`,
`compatibility`, `limitations`, `action`, `prepare`, `rollout`, `verify`, and
`custom`. Assign every shared fact to all affected products (`backend`, `sdk`,
`upgrade`) so their wording stays consistent. A `commit` is an evidence pointer,
not an automated proof of the statement; reviewers must check its contents.

Render the reviewed inventory into a **new** directory:

```bash
kfpr render-editorial /tmp/kfp-3.2.1-editorial/inventory.json \
  --output /tmp/kfp-3.2.1-reviewed
kfpr validate-notes /tmp/kfp-3.2.1-reviewed/backend.md --product backend
kfpr validate-notes /tmp/kfp-3.2.1-reviewed/sdk.md --product sdk
kfpr validate-notes /tmp/kfp-3.2.1-reviewed/upgrade.md --product upgrade
```

Rendering verifies that evidence still matches the immutable Git range and refuses
to overwrite an existing directory. It includes only reviewed `include` notes and
shared facts; pending decisions and incomplete category coverage are reported.
Unfilled template prompts remain visible. Finish the review before publishing;
the tool does not infer prose, release support policy, or approval from a clean lint.

Generate the normal checkpoint-bound previews below, then use the reviewed prose
to edit their Markdown files. Keep their `.json` provenance sidecars and generated
installation instructions intact. Do not copy the bundle's `inventory.json` over
a preview sidecar. Submit the completed upgrade guide as a normal docs PR, preserving
existing anchors and verifying the Sphinx build. A link to an existing source file
is insufficient if that file is excluded from the documentation build.

### Editorial and generated sections

Backend and SDK templates include explicit **Bug fixes** sections. Backend prompts
cover security and resilience; SDK prompts focus on authoring, APIs, installation,
and compatibility. The upgrade template follows prepare → roll out → verify,
with separate custom-configuration guidance. Describe normal manifest upgrades as
normal upgrades; require special coordination only when the released implementation
requires it. Keep future-release plans and release-team qualification history out
of operator instructions.

Aim for about 500–800 words for backend notes and 400–600 for SDK notes, excluding
credits and code. Cover each benefit once, link detailed guidance, and select
representative bug fixes rather than pasting the complete commit log. The generated
commit reference is collapsed and can be removed after review. Do not import older
version sections from `CHANGELOG.md` or `sdk/RELEASE.md`.

New standard previews also save a range-specific inventory beside the Markdown.
Generated contributor and source-archive sections use marked blocks. Credits use
Git author names and coauthor trailers linked to commits; the offline tool does not guess GitHub handles
or claim first-time contributor status. Source links become usable after tag
publication; the release page links to actual uploaded assets. This does not claim
that wheels, container images, or other binaries are attached. Add verified package
or image links in editorial prose outside the managed blocks when appropriate.

Deleting or editing a complete managed block is repaired from its generated
baseline during preview/publication review or regeneration. Malformed/duplicate
markers fail with an actionable error. Existing previews without these markers
retain their legacy behavior; regenerate them to adopt the new sections. Rewrites
must preserve credits and downloads while leaving the editorial text editable.


Write user-facing SDK changes and upgrade instructions under `Current Version
(in development)` in `sdk/RELEASE.md`. During version preparation, `kfpr` moves
that text into the versioned release section and appends commit-derived entries
under `Generated changes`. Handwritten text is preserved; the new development
section starts with an empty template. Review the version PR for duplicate
entries and accuracy before merging it.

Both `create-backend-release` and `create-sdk-release` write an editable Markdown
preview before creating a GitHub release. With the default checkpoint, previews
are `release-state-notes/<version>.md` and
`release-state-notes/sdk-<version>.md`. With a custom checkpoint, the directory is
`<state-file-stem>-notes` beside that file. The default directory is Git-ignored;
keep custom checkpoint files and previews outside the checkout.

To prepare previews without publishing, run these commands with the intended
release version and previous backend/SDK tags available in your local clone:

```bash
kfpr run create-backend-release \
  --release-type patch --version 3.2.1 --fork-remote USER \
  --previous-release 3.2.0 \
  --state-file /tmp/kfp-release-state.json --dry-run

kfpr run create-sdk-release \
  --release-type patch --version 3.2.1 --fork-remote USER \
  --previous-sdk-release sdk-3.2.0 \
  --state-file /tmp/kfp-release-state.json --dry-run
```

These dry runs write local preview files and print their contents, but do not
publish or ask for publication approval. Note generation reads local Git history;
fetch the intended release branch and tags after the release changes are merged.
The [explicit range options](#select-explicit-release-note-ranges) select each
component's previous tag and the intended target commit.

Edit the printed files, then repeat the command without `--dry-run`, adding
`--done` to checkpoint successful completion. The step displays the file again
and asks for review confirmation. You may also edit it while the prompt is
waiting. Answering no stops publication and retains the preview. Empty files
cannot be published. GitHub receives the reviewed file through `--notes-file`.
Backend architecture qualification still applies, and SDK release replacement
does not delete an existing release until the notes review is accepted.

Existing previews are reused without regeneration so retries preserve your
edits. A `<tag>.json` sidecar beside each `<tag>.md` preview records the resolved
range; when the base or target commit changes, the CLI refuses to reuse the stale
preview. A missing or invalid sidecar also blocks reuse. Full-flow dry runs defer
note generation; individual preview commands require existing release refs.

To refresh an existing preview while preserving edits, run:

```bash
kfpr regenerate-notes --product backend --version 3.2.1 \
  --previous-release 3.2.0 --release-target refs/tags/3.2.1 \
  --state-file /tmp/kfp-release-state.json

kfpr regenerate-notes --product sdk --version 3.2.1 \
  --previous-sdk-release sdk-3.2.0 --release-target refs/tags/sdk-3.2.1 \
  --state-file /tmp/kfp-release-state.json
```

Version and range options default to the saved checkpoint when available. This
command changes only local preview files; it does not publish, create tags, or
mark release steps complete. Targets must still match any qualified commit in
the checkpoint.

Each new preview stores its generated baseline in the JSON sidecar. Regeneration
merges your edited Markdown with fresh generated notes using that baseline, and
backs up the existing preview and sidecar in a uniquely named directory beside
them. Non-overlapping edits are retained. Overlapping edits produce conflict
markers and a nonzero exit status; edit the preview to resolve them. Publication
and further regeneration reject unresolved markers. Then rerun the individual
preview command to review the result before publishing.

Older previews without a stored baseline cannot be merged safely. The command
leaves them untouched: preserve their handwritten edits, move both files aside,
create a fresh individual preview, and reapply the edits once. Subsequent
regenerations use the saved baseline.
Compare the notes with the final release contents before approving publication.
Clearing a checkpoint does not delete previews.

New previews begin with an editorial scaffold for highlights, required actions,
compatibility, and known issues. Replace its TODO comments with reviewed prose,
or remove sections that do not apply. Describe breaking changes as before →
after → required action; record deprecation replacements and planned removal
versions when known. Compatibility notes must distinguish tested combinations
from supported-by-policy combinations and unevaluated configurations. Do not
infer compatibility or claim there are no known issues from an empty change list.

Validate any local Markdown preview without publishing or changing files:

```bash
kfpr validate-notes /tmp/kfp-release-state-notes/3.2.1.md
```

Validation also runs when previewing or regenerating notes. It warns about
unfinished editorial prompts, empty editorial sections, duplicate bullet entries,
breaking/deprecation entries without explanatory guidance or migration links,
multiple version-history headings, and excessive prose. With `--product`, it also
checks for missing Bug fixes, Contributors, and Assets release sections.
These checks are heuristics, not a substitute for review. Editorial warnings are
advisory and return success; unreadable input fails. Existing empty-note, stale
range, qualification, and merge-conflict gates still block publication. Resolve
warnings and review the actual file before accepting the publication prompt.
Removing or completing the scaffold is preserved by preview regeneration.

Use the preview to explain what users need to know, in this order where relevant:

1. **Highlights:** the most important changes and their user impact.
2. **Upgrade actions and breaking changes:** affected users, required steps,
   and migration links.
3. **Security fixes:** affected components, qualification scope, and exceptions.
4. **Changes by component:** SDK, backend, UI, and Kubernetes integration.
5. **Compatibility and known issues:** supported versions and workarounds.
6. **Install/upgrade links and full changelog.**

Omit empty sections. The [release-note policy](../CONTRIBUTING.md#release-note-policy)
uses the same commit parser for the detailed changelog, SDK entries, and release
previews. Routine maintenance is omitted unless it carries breaking, security,
or deprecation impact. Add concise highlights and migration guidance to the
generated entries; preserve the link to the versioned `sdk/RELEASE.md`.

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

Maintenance patches still leave development-branch version files, changelog and Read the Docs
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

### Preserve backend changelog guidance

Backend preparation moves handwritten `## Unreleased` notes into the new version
section and leaves an empty `Unreleased` heading for future work. Historical
sections remain unchanged. Repeating preparation is a no-op only when the existing
version section exactly matches generated output. A section containing authored
notes or later edits is left untouched and preparation stops for manual review;
keep those notes and resume after verifying the release preparation changes.
SDK-only notes require an SDK or shared API scope; unknown and unscoped commits
are included only in backend notes.
