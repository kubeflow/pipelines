# Current Version (Still in Development)

## Major Features and Improvements

* Add optional support to specify description for pipeline version [\#6472](https://github.com/kubeflow/pipelines/issues/6472).
* New v2 experimental compiler. [\#6803](https://github.com/kubeflow/pipelines/pull/6803)

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Fix the the specified 'mlpipeline-ui-metadata','mlpipeline-metrics' path is overrided by default value [\#6796](https://github.com/kubeflow/pipelines/pull/6796)
* Fix placeholder mapping error in v2. [\#6794](https://github.com/kubeflow/pipelines/pull/6794)
* Add `OnTransientError` to allowed retry policies [\#6808](https://github.com/kubeflow/pipelines/pull/6808)
* Add optional `filter` argument to list methods of KFP client [\#6748](https://github.com/kubeflow/pipelines/pull/6748)
* Depends on `kfp-pipeline-spec>=0.1.13,<0.2.0` [\#6803](https://github.com/kubeflow/pipelines/pull/6803)

## Documentation Updates

# 1.8.6

## Major Features and Improvements

* Add functions to sdk client to delete and disable jobs [\#6754](https://github.com/kubeflow/pipelines/pull/6754)

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Require base and target images for components built using
  `kfp components build` CLI command to be unique
  [\#6731](https://github.com/kubeflow/pipelines/pull/6731)
* Try to use `apt-get python3-pip` when pip does not exist in containers used by
  v2 lightweight components [\#6737](https://github.com/kubeflow/pipelines/pull/6737)
* Implement LoopArgument and LoopArgumentVariable v2. [\#6755](https://github.com/kubeflow/pipelines/pull/6755)
* Implement Pipeline task settings for v2 dsl. [\#6746](https://github.com/kubeflow/pipelines/pull/6746)

## Documentation Updates

* N/A

# 1.8.5

## Major Features and Improvements

* Add v2 placeholder variables [\#6693](https://github.com/kubeflow/pipelines/pull/6693)
* Add a new command in KFP's CLI, `components`, that enables users to manage and build
  v2 components in a container with Docker [\#6417](https://github.com/kubeflow/pipelines/pull/6417)

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Fix executor getting None as value when float 0 is passed in. [\#6682](https://github.com/kubeflow/pipelines/pull/6682)
* Fix function-based components not preserving the namespace of GCPC artifact types. [\#6702](https://github.com/kubeflow/pipelines/pull/6702)
* Fix `dsl.` prefix in component I/O type annotation breaking component at runtime. [\#6714](https://github.com/kubeflow/pipelines/pull/6714)
* Update v2 yaml format [\#6661](https://github.com/kubeflow/pipelines/pull/6661)
* Implement v2 PipelineTask [\#6713](https://github.com/kubeflow/pipelines/pull/6713)
* Fix type_utils [\#6719](https://github.com/kubeflow/pipelines/pull/6719)
* Depends on `typing-extensions>=3.7.4,<4; python_version<"3.9"` [\#6683](https://github.com/kubeflow/pipelines/pull/6683)
* Depends on `click>=7.1.2,<9` [\#6691](https://github.com/kubeflow/pipelines/pull/6691)
* Depends on `cloudpickle>=2.0.0,<3` [\#6703](https://github.com/kubeflow/pipelines/pull/6703)
* Depends on `typer>=0.3.2,<1.0` [\#6417](https://github.com/kubeflow/pipelines/pull/6417)

## Documentation Updates

* N/A

# 1.8.4

## Major Features and Improvements

* N/A

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Support artifact types under google namespace [\#6648](https://github.com/kubeflow/pipelines/pull/6648)
* Fix a couple of bugs that affect nested loops and conditions in v2. [\#6643](https://github.com/kubeflow/pipelines/pull/6643)
* Add IfPresentPlaceholder and ConcatPlaceholder for v2 ComponentSpec.[\#6639](https://github.com/kubeflow/pipelines/pull/6639)

## Documentation Updates

* N/A

# 1.8.3

## Major Features and Improvements

* Support URI templates with ComponentStore. [\#6515](https://github.com/kubeflow/pipelines/pull/6515)

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Fix duplicate function for `list_pipeline_versions()`. [\#6594](https://github.com/kubeflow/pipelines/pull/6594)
* Support re-use of PVC with VolumeOp. [\#6582](https://github.com/kubeflow/pipelines/pull/6582)
* When namespace file is missing, remove stack trace so it doesn't look like an error [\#6590](https://github.com/kubeflow/pipelines/pull/6590)
* Local runner supports additional docker options. [\#6599](https://github.com/kubeflow/pipelines/pull/6599)
* Fix the error that kfp v1 and v2 compiler failed to provide unique name for ops of the same component. [\#6600](https://github.com/kubeflow/pipelines/pull/6600)

## Documentation Updates

* N/A

# 1.8.2

## Major Features and Improvements

* N/A

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Fix component decorator could result in invalid component if `install_kfp_package=False`. [\#6527](https://github.com/kubeflow/pipelines/pull/6527))
* v2 compiler to throw no task defined error. [\#6545](https://github.com/kubeflow/pipelines/pull/6545)
* Improve output parameter type checking in V2 SDK. [\#6566](https://github.com/kubeflow/pipelines/pull/6566)
* Use `Annotated` rather than `Union` for `Input` and `Output`. [\#6573](https://github.com/kubeflow/pipelines/pull/6573)
* Depends on `typing-extensions>=3.10.0.2,<4`. [\#6573](https://github.com/kubeflow/pipelines/pull/6573)

## Documentation Updates

* N/A


# 1.8.1

## Major Features and Improvements

* Support container environment variable in v2. [\#6515](https://github.com/kubeflow/pipelines/pull/6515)

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Define PipelineParameterChannel and PipelineArtifactChannel in v2. [\#6470](https://github.com/kubeflow/pipelines/pull/6470)
* Remove dead code on importer check in v1. [\#6508](https://github.com/kubeflow/pipelines/pull/6508)
* Fix issue where dict, list, bool typed input parameters don't accept constant values or pipeline inputs. [\#6523](https://github.com/kubeflow/pipelines/pull/6523)
* Fix passing in "" to a str parameter causes the parameter to receive it as None instead. [\#6533](https://github.com/kubeflow/pipelines/pull/6533)
* Get short name of complex input/output types to ensure we can map to appropriate de|serializer. [\#6504](https://github.com/kubeflow/pipelines/pull/6504)
* Fix Optional type hint causing executor to ignore user inputs for parameters. [\#6541](https://github.com/kubeflow/pipelines/pull/6541)
* Depends on `kfp-pipeline-spec>=0.1.10,<0.2.0` [\#6515](https://github.com/kubeflow/pipelines/pull/6515)
* Depends on `kubernetes>=8.0.0,<19`. [\#6532](https://github.com/kubeflow/pipelines/pull/6532)

## Documentation Updates

* N/A


# 1.8.0

## Major Features and Improvements

* Add "--detail" option to kfp run get. [\#6404](https://github.com/kubeflow/pipelines/pull/6404)
* Support `set_display_name` in v2. [\#6471](https://github.com/kubeflow/pipelines/issues/6471)

## Breaking Changes

* Revert: "Add description to upload_pipeline_version in kfp" [\#6468](https://github.com/kubeflow/pipelines/pull/6468)

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

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

* N/A

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
