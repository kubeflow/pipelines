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

We will move away from `setup.py` and `requirements.txt` files to a single `pyproject.toml` file. This aligns with modern Python packaging standards and simplifies dependency specification.

### Version Alignment

The consolidated package will use a unified version number (starting with 3.0.0) to signify this major structural change and ensure users get compatible versions of all components.

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
    *   Specifically, `kubernetes_platform` tests will be moved to `sdk/python/kfp/kubernetes/test/`.
2.  **Integration Tests**: Verify that the consolidated package works with existing integration tests.
3.  **Installation Tests**: Verify that `pip install .` and `pip install kfp` work correctly in a fresh environment.

## Migration Strategy

Users will simply need to upgrade to the new `kfp` version:

```bash
pip install --upgrade kfp
```

This will install the consolidated package. The old packages (`kfp-pipeline-spec`, etc.) will no longer be needed as dependencies, though they may remain on PyPI for older versions.

**Breaking Changes**:
*   Users directly importing `kfp_pipeline_spec`, `kfp_server_api`, or `kfp_kubernetes` will need to update their imports to `kfp.pipeline_spec`, `kfp.server_api`, and `kfp.kubernetes` respectively.

## Alternatives

1.  **Keep packages separate**: Continue with the current multi-package approach. This maintains the status quo but doesn't solve the dependency/versioning issues.
### Optional Dependencies

**Question**: Should `kfp.kubernetes` be an optional dependency?

**Recommendation**: **Yes**.
- The `kfp.kubernetes` module depends on the `kubernetes` Python client, which is a significant dependency.
- To keep the base `kfp` package lightweight and avoid unnecessary overhead in execution environments (executioners/launchers), `kfp.kubernetes` will be an optional dependency.
- Users needing Kubernetes-specific features (e.g., `use_secret_as_env`, `add_node_affinity`) must install `kfp[kubernetes]`.
- The code for `kfp.kubernetes` is included in the package, but importing it without the optional dependency installed will raise a helpful `ImportError` instructing the user to install `kfp[kubernetes]`.
