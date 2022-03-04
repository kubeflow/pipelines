# Current Version (Still in Development)

## Major Features and Improvements

* Support passing parameters in v2 using google.protobuf.Value [\#6804](https://github.com/kubeflow/pipelines/pull/6804).
* Implement experimental v2 `@component` component [\#6825](https://github.com/kubeflow/pipelines/pull/6825)
* Add load_component_from_* for v2 [\#6822](https://github.com/kubeflow/pipelines/pull/6822)
* Merge v2 experimental change back to v2 namespace [\#6890](https://github.com/kubeflow/pipelines/pull/6890)
* Add ImporterSpec v2 [\#6917](https://github.com/kubeflow/pipelines/pull/6917)
* Add add set_env_variable for Pipeline task [\#6919](https://github.com/kubeflow/pipelines/pull/6919)
* Add metadata field for importer [\#7112](https://github.com/kubeflow/pipelines/pull/7112)
* Add in filter to list_pipeline_versions SDK method [\#7223](https://github.com/kubeflow/pipelines/pull/7223)
* Add `enable_job` method to client [\#7239](https://github.com/kubeflow/pipelines/pull/7239)
* Support getting pipeline status in exit handler. [\#7309](https://github.com/kubeflow/pipelines/pull/7309)

## Breaking Changes

* Remove sdk/python/kfp/v2/google directory for v2, including google client and custom job [\#6886](https://github.com/kubeflow/pipelines/pull/6886)
* APIs imported from the v1 namespace are no longer supported by the v2 compiler. [\#6890](https://github.com/kubeflow/pipelines/pull/6890)
* Deprecate v2 compatible mode in v1 compiler. [\#6958](https://github.com/kubeflow/pipelines/pull/6958)
* Drop support for python 3.6 [\#7303](https://github.com/kubeflow/pipelines/pull/7303)
* Deprecate v1 code to deprecated folder [\#7291](https://github.com/kubeflow/pipelines/pull/7291)

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Fix importer ignoring reimport setting, and switch to Protobuf.Value for import uri [\#6827](https://github.com/kubeflow/pipelines/pull/6827)
* Fix display name support for groups [\#6832](https://github.com/kubeflow/pipelines/pull/6832)
* Fix regression on optional inputs [\#6905](https://github.com/kubeflow/pipelines/pull/6905) [\#6937](https://github.com/kubeflow/pipelines/pull/6937)
* Depends on `google-auth>=1.6.1,<3` [\#6939](https://github.com/kubeflow/pipelines/pull/6939)
* Change otherwise to else in yaml [\#6952](https://github.com/kubeflow/pipelines/pull/6952)
* Avoid pydantic bug on Union type [\#6957](https://github.com/kubeflow/pipelines/pull/6957)
* Fix bug for if and concat placeholders [\#6978](https://github.com/kubeflow/pipelines/pull/6978)
* Fix bug for resourceSpec [\#6979](https://github.com/kubeflow/pipelines/pull/6979)
* Fix regression on nested loops [\#6990](https://github.com/kubeflow/pipelines/pull/6990)
* Fix bug for input/outputspec and positional arguments [\#6980](https://github.com/kubeflow/pipelines/pull/6980)
* Fix importer not using correct output artifact type [\#7235](https://github.com/kubeflow/pipelines/pull/7235)
* Add verify_ssl for Kubeflow client [\#7174](https://github.com/kubeflow/pipelines/pull/7174)
* Depends on `typing-extensions>=3.7.4,<5; python_version<"3.9"` [\#7288](https://github.com/kubeflow/pipelines/pull/7288)
* Depends on `google-api-core>=1.31.5, >=2.3.2` [\#7377](https://github.com/kubeflow/pipelines/pull/7377)

## Documentation Updates

# 1.8.11

## Major Features and Improvements

* kfp.Client uses namespace from initialization if set for the instance context [\#7056](https://github.com/kubeflow/pipelines/pull/7056)
* Add importer_spec metadata to v1 [\#7180](https://github.com/kubeflow/pipelines/pull/7180)

## Breaking Changes

* Fix breaking change in Argo 3.0, to define TTL for workflows. Makes SDK incompatible with KFP pre-1.7 versions [\#7141](https://github.com/kubeflow/pipelines/pull/7141)

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Remove redundant check in set_gpu_limit [\#6866](https://github.com/kubeflow/pipelines/pull/6866)
* Fix create_runtime_artifact not covering all types. [\#7168](https://github.com/kubeflow/pipelines/pull/7168)
* Depend on `absl-py>=0.9,<2` [\#7172](https://github.com/kubeflow/pipelines/pull/7172)

## Documentation Updates

# 1.8.10

## Major Features and Improvements

* Improve CLI experience for archiving experiments, managing recurring runs and listing resources [\#6934](https://github.com/kubeflow/pipelines/pull/6934)

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Visualizations and metrics do not work with data_passing_methods. [\#6882](https://github.com/kubeflow/pipelines/pull/6882)
* Fix a warning message. [\#6911](https://github.com/kubeflow/pipelines/pull/6911)
* Refresh access token only when it expires. [\#6941](https://github.com/kubeflow/pipelines/pull/6941)
* Fix bug in checking values in _param_values. [\#6965](https://github.com/kubeflow/pipelines/pull/6965)

## Documentation Updates

* N/A

# 1.8.9

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

* Make `Artifact` type be compatible with any sub-artifact types bidirectionally [\#6859](https://github.com/kubeflow/pipelines/pull/6859)

## Documentation Updates

* N/A

# 1.8.8

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

* Fix cloud scheduler's job name [\#6844](https://github.com/kubeflow/pipelines/pull/6844)
* Add deprecation warnings for v2 [\#6851](https://github.com/kubeflow/pipelines/pull/6851)

## Documentation Updates

* N/A

# 1.8.7

## Major Features and Improvements

* Add optional support to specify description for pipeline version [\#6472](https://github.com/kubeflow/pipelines/issues/6472).
* New v2 experimental compiler [\#6803](https://github.com/kubeflow/pipelines/pull/6803).

## Breaking Changes

* N/A

### For Pipeline Authors

* N/A

### For Component Authors

* N/A

## Deprecations

* N/A

## Bug Fixes and Other Changes

* Fix the the specified 'mlpipeline-ui-metadata','mlpipeline-metrics' path is overrided by default value [\#6796](https://github.com/kubeflow/pipelines/pull/6796)
* Fix placeholder mapping error in v2. [\#6794](https://github.com/kubeflow/pipelines/pull/6794)
* Add `OnTransientError` to allowed retry policies [\#6808](https://github.com/kubeflow/pipelines/pull/6808)
* Add optional `filter` argument to list methods of KFP client [\#6748](https://github.com/kubeflow/pipelines/pull/6748)
* Depends on `kfp-pipeline-spec>=0.1.13,<0.2.0` [\#6803](https://github.com/kubeflow/pipelines/pull/6803)

## Documentation Updates

* N/A

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
* Add `load_component_from_spec` for SDK v1 which brings back the ability to build components directly in python, using `ComponentSpec` [\#6690](https://github.com/kubeflow/pipelines/pull/6690)

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

* `kfp.components`no longer imports everything from `kfp.components`. For instance, `load_component_from_*` methods are available only from `kfp.components`, but not from `kfp.components`.
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
