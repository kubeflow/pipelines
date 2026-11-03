# Upgrading Argo Workflows

Argo upgrades must keep the Go client and CLI, controller and executor images,
upstream manifests and CRDs, CI matrices, and runtime preload images aligned.
Use the complete update command for both Dependabot proposals and manual upgrades.

## KFP 3.0 support boundary

Master supports Argo Workflows 4.0.x and 4.1.x, with exact patch versions pinned in
`COMPATIBILITY_VERSION` and `VERSION`. The default runtime uses 4.1.x; API and
end-to-end CI also exercise the 4.0.x compatibility runtime. Keep each lane on its
latest stable patch through coordinated upgrades, not floating image tags.

[Upstream maintains the two most recent minor releases](https://argo-workflows.readthedocs.io/en/latest/releases/).
As of October 2026 these are 4.0 and 4.1. Reassess the support floor when upstream
introduces a new release line; do not silently drop a supported line in a patch upgrade.

Argo 3.x is not supported by KFP 3.0. KFP 2.x release branches retain their
documented 3.x compatibility coverage, independently of upstream maintenance.
Before upgrading KFP 2.x to 3.0, operators must upgrade the Argo controller and
executor together to a supported 4.x patch and follow the
[upstream upgrade guide](https://argo-workflows.readthedocs.io/en/latest/upgrading/).
Track the broader migration work in
[the KFP 3.0 compatibility issue](https://github.com/kubeflow/pipelines/issues/14139).

## Upgrade the current release

From the repository root, select an exact stable tag:

```bash
ARGO_TAG=v4.1.6
make -C third_party/argo update ARGO_VERSION="${ARGO_TAG}"
```

The command updates `VERSION` and the maintained runtime, CI, and upgrade-guide
references, including `go.mod` and `go.sum` through `go mod tidy`. It validates the
complete edit plan and resolves Go dependencies before writing tracked files; failed validation or dependency
resolution leaves them unchanged. It uses the installed Go compiler and rejects
an implicit compiler or Argo module-major change. Handle major migrations and
compiler upgrades explicitly before using this command.

The existing workflow also remains available: edit [VERSION](./VERSION), then run
`make -C third_party/argo update`. To change the older supported release, edit
[COMPATIBILITY_VERSION](./COMPATIBILITY_VERSION) before running the command. Keep
it older than the current release. Both supported CI lanes remain enabled.

Dependabot deliberately proposes the Argo Go module separately from the generic
Go minor/patch batch. Its module and image proposals still need this command to
update repository-specific references, including the executor command argument
and upstream CRD Git refs. Run it with the proposal's target version and review
one complete upgrade before approval. Normal version and security proposals
remain enabled.

## Verify the upgrade

1. Refresh the tested patch versions in the public [compatibility notice](../../README.md)
   for every upgrade; this notice is checked by the consistency tests and is not
   rewritten by the update command. Update the compatibility matrix when either
   supported release line changes. Review upstream release notes and migration
   requirements; pin synchronization does not establish compatibility.
2. Check the synchronized references and review the full diff:

   ```bash
   python3 .github/resources/scripts/sync_argo_versions.py --scope all --check
   python3 -m unittest discover -s .github/resources/scripts -p '*argo*test.py'
   git diff --check
   git diff
   ```

3. Build the backend images with `make -C backend image_all` and render the Argo
   Kustomize overlays. CRDs are referenced from the upstream release rather than
   vendored, so their Git refs must be included in the upgrade.
4. Run the API and end-to-end CI suites for both supported Argo versions and test
   the deployment before merging. Keep the previous release available for rollback.

The component targets `update_ci` (also `update_tests`), `update_manifests`,
`update_backend`, and `update_docs` are available for focused maintenance. They
read `VERSION`; a partial target does not constitute a complete upgrade. CI uses
`update_manifests` to select an older supported Argo runtime without changing the
Go module or CI version matrix. The module-major guard applies to `update` and
`update_backend`. Use `make update` for a complete release change.

`release.sh` is a no-op retained for consistency with other third-party dependencies.
