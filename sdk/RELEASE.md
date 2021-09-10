# Current Version (Still in Development)

## Major Features and Improvements
- Add support to specify description for pipeline version #TBD.

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
* Get short name of complex input/output types to ensure we can map to appropriate de|serializer. [\#6504](https://github.com/kubeflow/pipelines/pull/6504)

## Documentation Updates

# 1.7.1

## Major Features and Improvements

* Surfaces Kubernetes configuration in container builder #6095

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Relaxes the requirement that component inputs/outputs must appear on the command line. #6268
* Fixed the compiler bug for legacy outputs mlpipeline-ui-metadata and mlpipeline-metrics. #6325
* Raises error on using importer in v2 compatible mode. #6330
* Raises error on missing pipeline name in v2 compatible mode. #6332
* Raises warning on container component without command. #6335
* Fixed the issue that SlicedClassificationMetrics, HTML, and Markdown type are not exposed in dsl package. #6343
* Fixed the issue that pip may not be available in lightweight component base image. #6359

## Documentation Updates

* N/A
