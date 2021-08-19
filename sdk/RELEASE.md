# Current Version (Still in Development)

## Major Features and Improvements
- Add support to specify description for pipeline version #TBD.

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes


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
