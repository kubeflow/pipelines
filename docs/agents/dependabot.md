# Dependency updates

Scheduled Dependabot updates target the root and SDK Python workspace through uv.
Independent Python projects retain normal pip version and security updates.
The workspace tracks `pyproject.toml` manifests and `uv.lock`; root and SDK
`requirements.txt` exports are no longer committed. Dependency fixes must update
the source constraints and lockfile, including the vulnerable locked version.
There are no duplicate workspace pip exports for Dependabot to update separately.

CI checks lock consistency, installs the frozen workspace, builds SDK distributions,
and installs the wheel in a clean Python 3.11 pip environment. Optional pip consumers
can generate dependency-only artifacts with uv 0.13.4:

```bash
bash .github/resources/scripts/export_python_requirements.sh /tmp/kfp-python-exports
```

The explicit output directory receives `workspace-requirements.txt` and
`kfp-requirements.txt`. These frozen, unhashed exports omit local workspace projects,
so they can be used outside the source checkout. Install the built SDK wheel separately.
Do not commit exports. `hack/update-all-requirements.sh` updates the workspace lock;
modern release preparation builds distributions without recreating exports. Older
release source checkouts that contain tracked exports retain their original behavior.

React, ReactDOM, and their typings are grouped together for both version and
security updates. Grouping does not waive peer compatibility, source review,
or the normal CI and merge requirements.

Vitest and `@vitest/*` packages are grouped for version and security updates:
coverage providers require the exact runner release. Keep these groups before
the generic npm group and update the frontend manifest and lock together.
Security grouping does not guarantee a non-vulnerable peer is updated; verify
the resolved peer versions and the coverage suite before merging.
The locked-version guard covers both `frontend` and `frontend/server`; each
package installs its own runner and coverage provider and must stay aligned.

`@typescript-eslint/*` packages are grouped for version and security updates
because the ESLint plugin requires a compatible parser release. Keep these
groups before the generic npm group and update the manifest and lock together.
Security grouping may leave non-vulnerable peers unchanged; verify the resolved
parser satisfies the plugin's peer range before merging. Keep the directly
pinned parser and plugin on the same release in the frontend manifest and lock;
the dependency configuration tests guard that alignment.

## Frontend dependency security constraints

The Kubernetes client override patches its nested `js-yaml` independently of
the server's direct YAML dependency. The `gaxios` override keeps `uuid` on
patched major 11, which retains CommonJS support; major 12 changes to ESM.
The server dependency regression test exercises gaxios's multipart UUID path.

The frontend selector-parser override covers nested Tailwind/PostCSS copies.
Tailwind 3.0.11 generated identical CSS with parser 7.1.7 during verification;
keep validating the CSS build when changing this cross-major override.

### Remaining braces advisory

[GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)
affects `braces` through 3.0.3, with no patched release listed on 2026-10-23.
Frontend and server retain `http-proxy-middleware → micromatch → braces`;
frontend build tooling also retains it through Tailwind's glob/file watchers.
These build tools consume repository-controlled patterns, not request inputs.

Production proxies omit `pathFilter` or use an artifact namespace callback.
Request URLs are not glob patterns in these configurations. The runtime
configuration test `frontend/server/proxy-path-filter.test.ts` enables optional
proxies and checks every registration for absent or function-valued filters.
Reassess reachability if a proxy adds string/array filters, callbacks start
parsing glob patterns, or the matching implementation changes.

This is a reachability assessment, not a patched dependency. The two lockfile
alerts remain pending an upstream fix or removal of their dependency chains.
The integration test project instead removes `braces` through coordinated
WebdriverIO 10/Mocha 12 upgrades.

## Release 2.18 maintenance

Dependabot reads its schedule from the default branch. Weekly entries target six ecosystems on
`release-2.18`: Go modules, Docker/Kubernetes image references, npm, pip, GitHub
Actions and pre-commit. The release-local configuration remains an inventory; it
does not independently schedule these PRs. Keep the default-branch release entries
aligned when adding maintained release projects, and remove them when that line
is no longer maintained.

Release Python uses independent requirements projects, including the SDK,
Kubernetes platform, legacy metadata/visualization services, proxy, readiness
tools and the PostgreSQL retained-data upgrade harness. The release schedule
covers 20 maintained Python projects, including the harness's MLMD, NumPy and
PyYAML requirements. It does not use master's uv workspace. The two generated
backend HTTP clients remain excluded; update them through API regeneration. TensorBoard's pip
directory and Dockerfile are present on release-2.18 after #15152 merged.

Release pip runs in four disjoint jobs: SDK/integration (seven directories, three
open PRs), runtimes (five, three), docs/release tooling (three, two), and support
tools/samples (five, two). This retains a combined ten-PR budget. Keep immediate
parent/child projects together because pip fetches child requirements too.
Shared dependency grouping applies within each job; updates crossing jobs can
produce separate PRs and still require compatibility review.

The [55-minute hosted timeout](https://github.com/kubeflow/pipelines/actions/runs/41309055846)
occurred after 837 directory visits across 42 dependency-group passes. The 56
pip-compile subprocesses took about six minutes combined; repeatedly visiting
all 20 directories dominated the work. Splitting the schedule bounds that
repeated traversal without excluding projects or freezing more dependencies.
Verify each new job completes after the schedule merges; this partition does
not guarantee every future dependency update resolves or meets the time limit.

Each release ecosystem retains weekly updates, dependency labels and
`do-not-merge/hold`. Non-pip ecosystems retain a ten-PR limit per entry. Grouping preserves golang.org/x modules, paired Argo
controller/executor images, MUI peers and shared package updates. Argo remains
outside generic Go minor/patch batches. The protobuf >=7 and docformatter 1.7.8
ignores retain explicit release compatibility constraints; they are not security
waivers.

Remove a release PR's hold only after current-head CI and compatibility review:

- Use the release's coordinated Go and Argo update commands; generated clients,
  runtime images, manifests, CRDs and test versions must agree.
- Preserve frontend Node/npm engine pins, Python package support metadata and
  peer constraints. A proposed version bump does not authorize dropping support.
- Review each database line separately. The branch retains MySQL 8.4, legacy
  8.0.3 and CI 8.0, plus PostgreSQL 18 and 14 overlays. Master's global MySQL
  8.4-only range would incorrectly cross these boundaries, so it is not copied.
  The generic Docker schedule proposes updates; it does not enforce a per-path
  database support matrix. Legacy MySQL and MLMD migrations remain explicit
  release decisions, and PostgreSQL production support remains deferred to 3.0.
- Qualify changed runtime images, including retained data and TensorBoard
  local/S3/GCS storage behavior, before publishing them. Restore the hold if the
  proposed head changes.

`target-branch` schedules version-update PRs, not security-update PRs: GitHub's
security updates apply to the default branch. Continue release-specific dependency
and image scans and advisory triage. These schedules do not enable automatic
merging, suppress findings or replace release acceptance.

Release 2.18 freezes `ml-metadata`: the active release pip schedule and release-local
inventory ignore all direct MLMD version updates. Keep the other dependencies in
those projects scheduled. This does not remove MLMD from dependency resolution or
waive advisory triage; any necessary MLMD change requires explicit review.
