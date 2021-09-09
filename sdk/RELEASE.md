# Current Version (Still in Development)

## Major Features and Improvements

* Support container environment variable in v2. [\#6515](https://github.com/kubeflow/pipelines/pull/6515)

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Remove dead code on importer check in v1. [\#6508](https://github.com/kubeflow/pipelines/pull/6508)
* Fix issue where dict, list, bool typed input parameters don't accept constant values or pipeline inputs. [\#6523](https://github.com/kubeflow/pipelines/pull/6523)
* Fix passing in "" to a str parameter causes the parameter to receive it as None instead. [\#6533](https://github.com/kubeflow/pipelines/pull/6533)
* Depends on `kfp-pipeline-spec>=0.1.10,<0.2.0` [\#6515](https://github.com/kubeflow/pipelines/pull/6515)
* Depends on kubernetes>=8.0.0,<19. [\#6532](https://github.com/kubeflow/pipelines/pull/6532)

## Documentation Updates

# 1.8.0

## Major Features and Improvements

* Add "--detail" option to kfp run get. [\#6404](https://github.com/kubeflow/pipelines/pull/6404)
* Support `set_display_name` in v2. [\#6471](https://github.com/kubeflow/pipelines/issues/6471)

## Breaking Changes

* Revert: "Add description to upload_pipeline_version in kfp" [\#6468](https://github.com/kubeflow/pipelines/pull/6468)

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Fix bug in PodSpec that overwrites nodeSelector [\#6512](https://github.com/kubeflow/pipelines/issues/6512)
* Add Alpha feature notice for local client [\#6462](https://github.com/kubeflow/pipelines/issues/6462)
* Import mock from stdlib and drop dependency. [\#6456](https://github.com/kubeflow/pipelines/issues/6456)
* Update yapf config and move it to sdk folder. [\#6467](https://github.com/kubeflow/pipelines/issues/6467)
* Fix typing issues. [\#6480](https://github.com/kubeflow/pipelines/issues/6480)
* Load v1 and v2 component yaml into v2 ComponentSpec and convert v1 component
  spec to v2 component spec [\#6497](https://github.com/kubeflow/pipelines/issues/6497)
* Format all Python files under SDK folder. [\#6501](https://github.com/kubeflow/pipelines/issues/6501)

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
