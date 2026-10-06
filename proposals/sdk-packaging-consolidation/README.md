# KEP-12548: KFP SDK Packaging Consolidation

<!-- toc -->

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Package Consolidation](#package-consolidation)
  - [Build System Modernization](#build-system-modernization)
  - [Version Alignment](#version-alignment)
- [Design Details](#design-details)
  - [Directory Structure](#directory-structure)
  - [Namespace Handling](#namespace-handling)
- [Test Plan](#test-plan)
- [Migration Strategy](#migration-strategy)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This proposal outlines the consolidation of the Kubeflow Pipelines (KFP) Python SDK into a single, unified package. Currently, the SDK is split across multiple packages (`kfp`, `kfp-pipeline-spec`, `kfp-server-api`, `kfp-kubernetes`), which creates complexity in versioning, dependency management, and release processes. This KEP proposes merging these into the main `kfp` package and modernizing the build system to use `pyproject.toml`.

## Motivation

The current split of the KFP SDK into multiple packages has led to several issues:
1.  **Dependency Management**: Users often face conflicts where `kfp` requires specific versions of `kfp-pipeline-spec` or `kfp-server-api`, leading to installation failures.
2.  **Release Testing Complexity**: Cutting a release of the SDK requires coordinating releases across multiple packages where certain testing depends on specific versions for potential new features, where some pipeline tests require the to-be-released version of KFP to be installed at runtime.
3.  **Backward Compatibility Maintenance**: Maintaining backward compatibility for the `kfp` package requires careful management of version dependencies and testing across multiple packages.

Consolidating these packages will simplify the user experience ("just install kfp") and streamline the development and release process.

### Goals

1.  Consolidate `kfp-pipeline-spec`, `kfp-server-api`, and `kfp-kubernetes` into the `kfp` package.
2.  Migrate the build system from `setup.py` to `pyproject.toml` (PEP 621).
3.  Ensure all existing functionality and tests are preserved.
4.  Simplify the installation process for end-users.

### Non-Goals

1.  Major refactoring of the internal logic of these components (other than what's needed for consolidation).
2.  Changing the public API surface significantly (backward compatibility should be maintained where possible, though import paths may change).

## Proposal

### Package Consolidation

The following packages will be merged into `kfp`:

*   **`kfp-pipeline-spec`**: Will move to `kfp.pipeline_spec`.
*   **`kfp-server-api`**: Will move to `kfp.server_api`.
*   **`kfp-kubernetes`**: Will move to `kfp.kubernetes`.

### Build System Modernization

The prerequisite uv migration introduced `pyproject.toml` and the workspace lockfile. Consolidation leaves one SDK distribution in `sdk/python/pyproject.toml`; the root `pyproject.toml` manages development tooling. Root and SDK `requirements.txt` files remain generated compatibility exports.

### Version Alignment

The consolidated package will use the SDK's version for all bundled modules. The version bump is deferred to the release process; this implementation does not publish packages or change existing release branches.

## Design Details

### Directory Structure

The new structure within `sdk/python/kfp/` will be:

```
sdk/python/kfp/
├── pipeline_spec/       # Formerly kfp-pipeline-spec
├── server_api/          # Formerly kfp-server-api
├── kubernetes/          # Formerly kfp-kubernetes
├── dsl/
├── client/
└── ...
```

### Namespace Handling

We will update internal imports to use relative imports or full `kfp.*` paths. For example, `import kfp_server_api` will become `from kfp import server_api` or `import kfp.server_api`.

## Test Plan

1.  **Unit Tests**: Migrate existing unit tests from the separate repositories/directories into the `kfp` test suite.
    *   Specifically, `kubernetes_platform` tests will be moved to `sdk/python/test/kubernetes/`.
2.  **Integration Tests**: Verify that the consolidated package works with existing integration tests.
3.  **Installation Tests**: Verify that `pip install .` and `pip install kfp` work correctly in a fresh environment.
4.  **Generation and Artifacts**: Preserve CI's explicit generate, build, then test sequence. Commit generated Python bindings so Git installs work from a clean checkout. Regenerate them in CI and reject drift; verify wheel and standalone sdist installs include all bundled modules.

## Migration Strategy

Users upgrading an existing split-package installation must remove the old file owners **before** installing the unified SDK. Set `KFP_VERSION` to the release containing consolidation, then run:

```bash
python -m pip uninstall -y kfp-pipeline-spec kfp-server-api kfp-kubernetes &&
python -m pip install --upgrade --force-reinstall "kfp==$KFP_VERSION"
```

Pip does not automatically remove distributions that are no longer dependencies. Removing them after installing the unified SDK can delete shared files; recover by force-reinstalling `kfp`. Restart Python processes and notebook kernels after upgrading. Fresh environments need only `kfp`. Historical packages remain on PyPI for older releases; third-party dependencies that require them must be updated before migration.

**Breaking Changes**:
*   Users directly importing the v2 `kfp_server_api` client must use `kfp.server_api`. The existing `kfp.pipeline_spec` and `kfp.kubernetes` import paths remain unchanged.

## Alternatives

1.  **Keep packages separate**: Continue with the current multi-package approach. This maintains the status quo but doesn't solve the dependency/versioning issues.
### Optional Dependencies

**Question**: Should `kfp.kubernetes` be an optional dependency?

**Decision**: Preserve existing runtime behavior in this consolidation. The base SDK already depends on the `kubernetes` Python client, so Kubernetes helpers are included without an additional install. Keep `kfp[kubernetes]` as a compatibility extra. Making the client optional would be a separate behavioral change.
