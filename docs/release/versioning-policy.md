# Kubeflow Pipelines Versioning Policy

The Kubeflow Pipelines version format follows [the semantic versioning](https://semver.org/).
The Kubeflow Pipelines versions are in the format of `X.Y.Z`, where `X` is the major version,
`Y` is the minor version, and `Z` is the patch version.

As a guideline, we increment the:
* MAJOR version when we make breaking changes on [stable features](./feature-status.md#stable-features),
* MINOR version when we add functionality in a backwards compatible manner, and
* PATCH version when we make backwards compatible bug fixes.

In reality, we are not following the guideline strictly, e.g. PATCH versions might have backward compatible new functionalities that is low-risk.

Additionally, we do pre-releases as an extension in the format of `X.Y.Z-rc.N`
where `N` is a number. The appendix indicates the Nth release candidate before
an upcoming public release named `X.Y.Z`.

The Kubeflow Pipelines version `X.Y.Z` refers to the version (git tag) of the released
Kubeflow Pipelines. It versions all released artifacts, including:
* `kfp` and `kfp-server-api` python packages
* install manifests
* docker images on gcr.io/ml-pipeline
* first party components
