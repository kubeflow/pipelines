# Current Version (Still in Development)

## Major Features and Improvements

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

## Documentation Updates

# 1.7.2

## Major Features and Improvements

* Add support to specify description for pipeline version [\#6395](https://github.com/kubeflow/pipelines/issues/6395).
* Add support for schema_version in pipeline [\#6366](https://github.com/kubeflow/pipelines/issues/6366)
* Add support for enabling service account for cloud scheduler in google client [\#6013](https://github.com/kubeflow/pipelines/issues/6013)

## Breaking Changes

* `kfp.v2.components`no longer imports everything from `kfp.components`. For instance, `load_component_from_*` methods are available only from `kfp.components`, but not from `kfp.v2.components`.
* No more ['_path' suffix striping](https://github.com/kubeflow/pipelines/issues/5279) from v2 components.

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Refactor and move v2 related code to under the v2 namespace [\#6358](https://github.com/kubeflow/pipelines/issues/6358)
* Fix importer not taking output from upstream [\#6439](https://github.com/kubeflow/pipelines/issues/6439)
* Clean up the unused arg in AIPlatformCient docstring [\#6406](https://github.com/kubeflow/pipelines/issues/6406)
* Add BaseModel data classes and pipeline saving [\#6372](https://github.com/kubeflow/pipelines/issues/6372)

## Documentation Updates

* N/A

# 1.7.1

## Major Features and Improvements

* Surfaces Kubernetes configuration in container builder [\#6095](https://github.com/kubeflow/pipelines/issues/6095)

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Relaxes the requirement that component inputs/outputs must appear on the command line. [\#6268](https://github.com/kubeflow/pipelines/issues/6268)
* Fixed the compiler bug for legacy outputs mlpipeline-ui-metadata and mlpipeline-metrics. [\#6325](https://github.com/kubeflow/pipelines/issues/6325)
* Raises error on using importer in v2 compatible mode. [\#6330](https://github.com/kubeflow/pipelines/issues/6330)
* Raises error on missing pipeline name in v2 compatible mode. [\#6332](https://github.com/kubeflow/pipelines/issues/6332)
* Raises warning on container component without command. [\#6335](https://github.com/kubeflow/pipelines/issues/6335)
* Fixed the issue that SlicedClassificationMetrics, HTML, and Markdown type are not exposed in dsl package. [\#6343](https://github.com/kubeflow/pipelines/issues/6343)
* Fixed the issue that pip may not be available in lightweight component base image. [\#6359](https://github.com/kubeflow/pipelines/issues/6359)

## Documentation Updates

* N/A
