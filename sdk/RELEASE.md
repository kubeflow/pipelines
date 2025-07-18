# Current Version (in development)

## Features

* feat(sdk): Add support for compiling pipelines to Kubernetes native format in SDK (#12012)

## Breaking changes

## Deprecations
* PipelineTaskFinalStatus field names pipelineJobResourceName and pipelineTaskName are deprecated. Support for these fields will be removed at a later date.

## Bug fixes and other changes

## Documentation updates

# 2.13.0

## Features

* feat(sdk): add upload pipeline and upload pipeline version from pipeline function (#11804)
* fix(sdk): Add SDK support for setting resource limits on older KFP versions (#11839)
* docs: mention that set_container_image works with dynamic images (#11795)
* fix(local): warn about oci:// not supported too (#11794)
* bug(backend,sdk): Use a valid path separator for Modelcar imports (#11767)
* fix(sdk): avoid conflicting component names in DAG when reusing pipelines (#11071)

## Breaking changes

## Deprecations

## Bug fixes and other changes

* Depends on `google-cloud-storage>=2.2.1,<4` [\#11735](https://github.com/kubeflow/pipelines/pull/11735)
* Fixed missing `kfp.__version__` when installing SDK via `pip install -e`. [\#11997](https://github.com/kubeflow/pipelines/pull/11997)

# 2.12.2

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes

# 2.12.1

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes

* Depends on `kfp-server-api>=2.1.0,<2.5.0` [\#11685](https://github.com/kubeflow/pipelines/pull/11685)

## Documentation updates

# 2.12.0

## Features

* Add support for placeholders in resource limits [\#11501](https://github.com/kubeflow/pipelines/pull/11501)
* Introduce cache_key to sdk [\#11466](https://github.com/kubeflow/pipelines/pull/11466)
* Add support for importing models stored in the Modelcar format (sidecar) [\#11606](https://github.com/kubeflow/pipelines/pull/11606)

## Breaking changes

## Deprecations

## Bug fixes and other changes

* dsl.component docstring typo [\#11547](https://github.com/kubeflow/pipelines/pull/11547)
* Update broken api-connect link [\#11521](https://github.com/kubeflow/pipelines/pull/11521)
* Fix kfp-sdk-test for different python versions [\#11559](https://github.com/kubeflow/pipelines/pull/11559)

## Documentation updates

# 2.11.0

## Features
* Expose `--existing-token` flag in `kfp` CLI to allow users to provide an existing token for authentication. [\#11400](https://github.com/kubeflow/pipelines/pull/11400)
* Add the ability to parameterize container images for tasks within pipelines [\#11404](https://github.com/kubeflow/pipelines/pull/11404)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Add error handling for image build/push failures in KFP SDK. [\#11164](https://github.com/kubeflow/pipelines/pull/11356)
* Backport fixes in kubeflow/pipelines#11075. [\#11392])(https://github.com/kubeflow/pipelines/pull/11392)
* Depends on `kfp-pipeline-spec==0.6.0`. [\#11447](https://github.com/kubeflow/pipelines/pull/11447)

## Documentation updates

# 2.10.1

## Features

## Breaking changes

## Deprecations
* Remove `kfp.deprecated` module [\#11366](https://github.com/kubeflow/pipelines/pull/11366)

## Bug fixes and other changes
* Support Python 3.13. [\#11372](https://github.com/kubeflow/pipelines/pull/11372)
* Fix accelerator type setting [\#11373](https://github.com/kubeflow/pipelines/pull/11373)
* Depends on `kfp-pipeline-spec==0.5.0`.

## Documentation updates

# 2.10.0

## Features
* Support dynamic machine type parameters in pipeline task setters. [\#11097](https://github.com/kubeflow/pipelines/pull/11097)
* Add a new `use_venv` field to the component decorator, enabling the component to run inside a virtual environment. [\#11326](https://github.com/kubeflow/pipelines/pull/11326)
* Add PipelineConfig to DSL to re-implement pipeline-level config [\#11112](https://github.com/kubeflow/pipelines/pull/11112)
* Allow disabling default caching via a CLI flag and env var [\#11222](https://github.com/kubeflow/pipelines/pull/11222)

## Breaking changes
* Deprecate the metrics artifact auto-populating feature. [\#11362](https://github.com/kubeflow/pipelines/pull/11362)

## Deprecations
* Set Python 3.9 as the Minimum Supported Version [\#11159](https://github.com/kubeflow/pipelines/pull/11159)

## Bug fixes and other changes
* Fix invalid escape sequences [\#11147](https://github.com/kubeflow/pipelines/pull/11147)
* Fix nested pipeline returns. [\#11196](https://github.com/kubeflow/pipelines/pull/11196)

## Documentation updates

# 2.9.0

## Features
* Kfp support for pip trusted host [#11151](https://github.com/kubeflow/pipelines/pull/11151)

## Breaking changes

* Pin kfp-pipeline-spec==0.4.0, kfp-server-api>=2.1.0,<2.4.0 [#11192](https://github.com/kubeflow/pipelines/pull/11192)

## Deprecations

## Bug fixes and other changes

* Loosening kubernetes dependency constraint [#11079](https://github.com/kubeflow/pipelines/pull/11079)
* Throw 'exit_task cannot depend on any other tasks.' error when an ExitHandler has a parameter dependent on other task [#11005](https://github.com/kubeflow/pipelines/pull/11005)

## Documentation updates

# 2.8.0

## Features
* Support dynamic machine type parameters in CustomTrainingJobOp. [\#10883](https://github.com/kubeflow/pipelines/pull/10883)

## Breaking changes
* Drop support for Python 3.7 since it has reached end-of-life. [\#10750](https://github.com/kubeflow/pipelines/pull/10750)

## Deprecations

## Bug fixes and other changes
* Throw compilation error when trying to iterate over a single parameter with ParallelFor [\#10494](https://github.com/kubeflow/pipelines/pull/10494)
* Add required auth scopes to RegistryClient for GCP service accounts credentials [#10819](https://github.com/kubeflow/pipelines/pull/10819)

## Documentation updates
* Make full version dropdown visible on all KFP SDK docs versions [\#10577](https://github.com/kubeflow/pipelines/pull/10577)

# 2.7.0

## Features
* Support local execution of sequential pipelines [\#10423](https://github.com/kubeflow/pipelines/pull/10423)
* Support local execution of `dsl.importer` components [\#10431](https://github.com/kubeflow/pipelines/pull/10431)
* Support local execution of pipelines in pipelines [\#10440](https://github.com/kubeflow/pipelines/pull/10440)
* Support `dsl.ParallelFor` over list of Artifacts [\#10441](https://github.com/kubeflow/pipelines/pull/10441)
* Fix bug where `dsl.OneOf` with multiple consumers cannot be compiled [\#10452](https://github.com/kubeflow/pipelines/pull/10452)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix the compilation error when trying to iterate over a list of dictionaries with ParallelFor [\#10436](https://github.com/kubeflow/pipelines/pull/10436)
## Documentation updates

# 2.6.0

## Features

## Breaking changes
* Soft breaking change for [Protobuf 3 EOL](https://protobuf.dev/support/version-support/#python). Migrate to `protobuf==4`. Drop support for `protobuf==3`. [\#10307](https://github.com/kubeflow/pipelines/pull/10307)

## Deprecations

## Bug fixes and other changes

## Documentation updates

# 2.5.0

## Features
* Add support for `dsl.PIPELINE_TASK_EXECUTOR_OUTPUT_PATH_PLACEHOLDER` and `dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER` [\#10240](https://github.com/kubeflow/pipelines/pull/10240)
* Add support for local component execution using `local.init()`, `DockerRunner`, and `SubprocessRunner`

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Support `.after()` referencing task in a `dsl.ParallelFor` group  [\#10257](https://github.com/kubeflow/pipelines/pull/10257)
* Support Python 3.12 [\#10271](https://github.com/kubeflow/pipelines/pull/10271)

## Documentation updates

# 2.4.0

## Features
* Add support for a Pythonic artifact authoring style [\#9932](https://github.com/kubeflow/pipelines/pull/9932)
* Support collecting outputs from conditional branches using `dsl.OneOf` [\#10067](https://github.com/kubeflow/pipelines/pull/10067)

## Breaking changes

## Deprecations
* Add notice of Python 3.7 support removal on April 23, 2024 [\#10139](https://github.com/kubeflow/pipelines/pull/10139)

## Bug fixes and other changes
* Fix type on `dsl.ParallelFor` sub-DAG output when a `dsl.Collected` is used. Non-functional fix. [\#10069](https://github.com/kubeflow/pipelines/pull/10069)
* Fix bug when `dsl.importer` argument is provided by a `dsl.ParallelFor` loop variable. [\#10116](https://github.com/kubeflow/pipelines/pull/10116)
* Fix client authentication in notebook and iPython environments [\#10094](https://github.com/kubeflow/pipelines/pull/10094)

## Documentation updates

# 2.3.0
## Features
* Support `PipelineTaskFinalStatus` in tasks that use `.ignore_upstream_failure()` [\#10010](https://github.com/kubeflow/pipelines/pull/10010)


## Breaking changes

## Deprecations

## Bug fixes and other changes

## Documentation updates
# 2.2.0

## Features
* Add support for `dsl.If`, `dsl.Elif`, and `dsl.Else` control flow context managers; deprecate `dsl.Condition` in favor of `dsl.If` [\#9894](https://github.com/kubeflow/pipelines/pull/9894)
* Create "dependency-free" runtime package (only `typing_extensions` required) for Lightweight Python Components to reduce runtime dependency resolution errors [\#9710](https://github.com/kubeflow/pipelines/pull/9710), [\#9886](https://github.com/kubeflow/pipelines/pull/9886)

# 2.0.1

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes

## Documentation updates
* Fix PyPI/README description [\#9668](https://github.com/kubeflow/pipelines/pull/9668)
# 2.0.0
KFP SDK 2.0.0 release notes are distilled to emphasize high-level improvements in major version 2. See preceding 2.x.x pre-release notes for a more comprehensive list.

Also see the KFP SDK v1 to v2 [migration guide](https://www.kubeflow.org/docs/components/pipelines/v2/migration/).
## Features

The KFP SDK 2.0.0 release contains features present in the KFP SDK v1's v2 namespace along with considerable additional functionality. A selection of these features include:
* An improved and unified Python-based authoring experience for [components](https://www.kubeflow.org/docs/components/pipelines/v2/components/) and [pipelines](https://www.kubeflow.org/docs/components/pipelines/v2/pipelines/)
* Support for using [pipelines as components](https://www.kubeflow.org/docs/components/pipelines/v2/pipelines/pipeline-basics/#pipelines-as-components) (pipeline in pipeline)
* Various additional [configurations for tasks](https://www.kubeflow.org/docs/components/pipelines/v2/pipelines/pipeline-basics/#task-configurations)
* Compilation to an Argo-independent [pipeline definition](https://www.kubeflow.org/docs/components/pipelines/v2/compile-a-pipeline/#ir-yaml) that enables pipelines to be compiled once and run anywhere
* Additonal SDK client functionality
* An improved [KFP CLI](https://www.kubeflow.org/docs/components/pipelines/v2/cli/)
* Refreshed [user documentation](https://www.kubeflow.org/docs/components/pipelines/v2/) and [reference documentation](https://kubeflow-pipelines.readthedocs.io/en/sdk-2.0.0/)

Selected contributions from pre-releases:
* Support for `@component` decorator for Python components [\#6825](https://github.com/kubeflow/pipelines/pull/6825)
* Support for loading v1 and v2 components using `load_component_from_*` [\#6822](https://github.com/kubeflow/pipelines/pull/6822)
* Support importer in KFP v2 [\#6917](https://github.com/kubeflow/pipelines/pull/6917)
* Add metadata field for `importer` [\#7112](https://github.com/kubeflow/pipelines/pull/7112)
* Add in filter to `list_pipeline_versions` client method [\#7223](https://github.com/kubeflow/pipelines/pull/7223)
* Support getting pipeline status in exit handler via `PipelineTaskFinalStatus` [\#7309](https://github.com/kubeflow/pipelines/pull/7309)
* Support v2 KFP API in SDK client
* Enable pip installation from custom PyPI repository [\#7453](https://github.com/kubeflow/pipelines/pull/7453), [\#8871](https://github.com/kubeflow/pipelines/pull/8871)
* Add additional methods to `kfp.client.Client` [\#7239](https://github.com/kubeflow/pipelines/pull/7239), [\#7563](https://github.com/kubeflow/pipelines/pull/7563), [\#7562](https://github.com/kubeflow/pipelines/pull/7562), [\#7463](https://github.com/kubeflow/pipelines/pull/7463), [\#7835](https://github.com/kubeflow/pipelines/pull/7835)
* CLI improvements [\#7547](https://github.com/kubeflow/pipelines/pull/7547), [\#7558](https://github.com/kubeflow/pipelines/pull/7558), [\#7559](https://github.com/kubeflow/pipelines/pull/7559), [\#7560](https://github.com/kubeflow/pipelines/pull/7560), , [\#7569](https://github.com/kubeflow/pipelines/pull/7569), [\#7567](https://github.com/kubeflow/pipelines/pull/7567), [\#7603](https://github.com/kubeflow/pipelines/pull/7603), [\#7606](https://github.com/kubeflow/pipelines/pull/7606), [\#7607](https://github.com/kubeflow/pipelines/pull/7607), [\#7628](https://github.com/kubeflow/pipelines/pull/7628), [\#7618](https://github.com/kubeflow/pipelines/pull/7618)
* Add Registry Client [\#7597](https://github.com/kubeflow/pipelines/pull/7597), [\#7763](https://github.com/kubeflow/pipelines/pull/7763)
* Add support for authoring Custom Container Components [\#8066](https://github.com/kubeflow/pipelines/pull/8066)
* Support setting parallelism in `ParallelFor` [\#8146](https://github.com/kubeflow/pipelines/pull/8146)
* Support using pipelines as components (pipeline in pipeline) [\#8179](https://github.com/kubeflow/pipelines/pull/8179), [\#8204](https://github.com/kubeflow/pipelines/pull/8204), [\#8209](https://github.com/kubeflow/pipelines/pull/8209)
* Support using pipeline in exit handlers [\#8220](https://github.com/kubeflow/pipelines/pull/8220)
* Support `google.`-namespaced artifact types [\#8191](https://github.com/kubeflow/pipelines/pull/8191), [\#8232](https://github.com/kubeflow/pipelines/pull/8232), [\#8233](https://github.com/kubeflow/pipelines/pull/8233), [\#8279](https://github.com/kubeflow/pipelines/pull/8279)
* Support passing upstream outputs to `importer` metadata [\#7660](https://github.com/kubeflow/pipelines/pull/7660)
* Add ability to skip building image when using `kfp component build` [\#8387](https://github.com/kubeflow/pipelines/pull/8387)
* Support single element `then` and `else_` arguments to `IfPresentPlaceholder` [\#8414](https://github.com/kubeflow/pipelines/pull/8414)
* Enable use of input and output placeholders in f-strings [\#8494](https://github.com/kubeflow/pipelines/pull/8494)
* Add comments to IR YAML file [\#8467](https://github.com/kubeflow/pipelines/pull/8467)
* Support fanning-in parameters [\#8631](https://github.com/kubeflow/pipelines/pull/8631) and artifacts [\#8808](https://github.com/kubeflow/pipelines/pull/8808) from tasks in a `dsl.ParellelFor` context using `dsl.Collected`
* Support `.ignore_upstream_failure()` on `PipelineTask` [\#8838](https://github.com/kubeflow/pipelines/pull/8838)
* Support setting cpu/memory requests [\#9121](https://github.com/kubeflow/pipelines/pull/9121)
* Support additional pipeline placeholders
* Support for platform-specific features via extension libraries [\#8940](https://github.com/kubeflow/pipelines/pull/8940), [\#9140](https://github.com/kubeflow/pipelines/pull/9140)
* Support `display_name` and `description` in `@dsl.pipeline` decorator [\#9153](https://github.com/kubeflow/pipelines/pull/9153)
* Extract component input and output descriptions from docstring [\#9156](https://github.com/kubeflow/pipelines/pull/9156)
* Allow user to specify platform when building container components [\#9212](https://github.com/kubeflow/pipelines/pull/9212)

## Breaking changes
See the [KFP SDK v1 to v2 migration guide](https://www.kubeflow.org/docs/components/pipelines/v2/migration/).

## Deprecations
* Deprecate compiling pipelines to JSON in favor of compiling to YAML [\#8179](https://github.com/kubeflow/pipelines/pull/8179)
* Deprecate ability to select `--engine` when building components [\#7559](https://github.com/kubeflow/pipelines/pull/7559)
* Deprecate `@dsl.component`'s `output_component_file` parameter in favor of compilation via the main `Compiler` [\#7554](https://github.com/kubeflow/pipelines/pull/7554)
* Deprecate client's `*_job` methods in favor of `*_recurring_run` methods [\#9112](https://github.com/kubeflow/pipelines/pull/9112)
* Deprecate pipeline task `.set_gpu_limit` in favor of `.set_accelerator_limit` [\#8836](https://github.com/kubeflow/pipelines/pull/8836)
* Deprecate `.add_node_selector_constraint` in favor of `.set_accelerator_type` [\#8980](https://github.com/kubeflow/pipelines/pull/8980)


## Bug fixes and other changes
* Various changes to dependency versions relative to v1
* Various dependencies removed relative to v1
* Various bug fixes applied in KFP SDK 2.x.x pre-release versions
* Various bug fixes associated with providing backward compatibility for KFP SDK v1
* Use YAML as default serialization format for pipeline IR [\#7431](https://github.com/kubeflow/pipelines/pull/7431)
* Support Python 3.10 [\#8186](https://github.com/kubeflow/pipelines/pull/8186) and 3.11 [\#8907](https://github.com/kubeflow/pipelines/pull/8907)
* Enable overriding caching options at submission time [\#7912](https://github.com/kubeflow/pipelines/pull/7912)
* Format file when compiling to JSON [\#7712](https://github.com/kubeflow/pipelines/pull/7712)
* Allow artifact inputs in pipeline definition [\#8044](https://github.com/kubeflow/pipelines/pull/8044)
* Support task-level retry policy [\#7867](https://github.com/kubeflow/pipelines/pull/7867)
* Support multiple exit handlers per pipeline [\#8088](https://github.com/kubeflow/pipelines/pull/8088)
* Migrate Out-Of-Band (OOB) authentication flow [\#8262](https://github.com/kubeflow/pipelines/pull/8262)
* CLI `kfp component build` generates runtime-requirements.txt [\#8372](https://github.com/kubeflow/pipelines/pull/8372)
* Throw exception for component parameter named Output [\#8367](https://github.com/kubeflow/pipelines/pull/8367)
* Block illegal `IfPresentPlaceholder` and `ConcatPlaceholder` authoring [\#8414](https://github.com/kubeflow/pipelines/pull/8414)
* Fix boolean default value compilation bug [\#8444](https://github.com/kubeflow/pipelines/pull/8444)
* Fix bug when writing to same file using gcsfuse and distributed training strategy in lightweight/containerized Python components [#8544](https://github.com/kubeflow/pipelines/pull/8544) [alternative fix after [#8455](https://github.com/kubeflow/pipelines/pull/8455) in `kfp==2.0.0b8`], [#8607](https://github.com/kubeflow/pipelines/pull/8607)
* Pipeline compilation is now triggered from `@pipeline` decorator instead of `Compiler.compile()` method.
Technically no breaking changes but compilation error could be exposed in a different (and earlier) stage [\#8179](https://github.com/kubeflow/pipelines/pull/8179)
* Fully support optional parameter inputs by writing `isOptional` field to IR [\#8612](https://github.com/kubeflow/pipelines/pull/8612)
* Add support for optional artifact inputs (toward feature parity with KFP SDK v1) [\#8623](https://github.com/kubeflow/pipelines/pull/8623)
* Fix upload_pipeline method on client when no name is provided [\#8695](https://github.com/kubeflow/pipelines/pull/8695)
* Enables output definitions when compiling components as pipelines [\#8848](https://github.com/kubeflow/pipelines/pull/8848)
* Add experiment_id parameter to create run methods [\#9004](https://github.com/kubeflow/pipelines/pull/9004)


## Documentation updates
* Refresh KFP SDK v2 [user documentation](https://www.kubeflow.org/docs/components/pipelines/v2/)
* Refresh KFP SDK v2 [reference documentation](https://kubeflow-pipelines.readthedocs.io/en/sdk-2.0.0/)


# 2.0.0-rc.2

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix integer value not allowed as float-typed return [\#9481](https://github.com/kubeflow/pipelines/pull/9481)
* Change `kubernetes` version requirement from `kubernetes>=8.0.0,<24` to `kubernetes>=8.0.0,<27`  [\#9545](https://github.com/kubeflow/pipelines/pull/9545)
* Fix bug when iterating over upstream task output in nested `dsl.ParallelFor` loop [\#9580](https://github.com/kubeflow/pipelines/pull/9580)

## Documentation updates

# 2.0.0-rc.1

## Features
* Support compiling primitive components with `dsl.PipelineTaskFinalStatus` input [\#9080](https://github.com/kubeflow/pipelines/pull/9080), [\#9082](https://github.com/kubeflow/pipelines/pull/9082)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix compilation of boolean constant passed to component [\#9390](https://github.com/kubeflow/pipelines/pull/9390)


## Documentation updates

# 2.0.0-beta.17

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Depends on `kfp-server-api==2.0.0b2` [\#9355](https://github.com/kubeflow/pipelines/pull/9355)

## Documentation updates

# 2.0.0-beta.16

## Features
* Allow user to specify platform when building container components [\#9212](https://github.com/kubeflow/pipelines/pull/9212)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix appengine import error [\#9323](https://github.com/kubeflow/pipelines/pull/9323)

## Documentation updates
# 2.0.0-beta.15

## Features
* Support `display_name` and `description` in `@dsl.pipeline` decorator [\#9153](https://github.com/kubeflow/pipelines/pull/9153)
* Extract component input and output descriptions from docstring [\#9156](https://github.com/kubeflow/pipelines/pull/9156)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix module not found error for containerized python components [\#9157](https://github.com/kubeflow/pipelines/pull/9157)

## Documentation updates

# 2.0.0-beta.14

## Features
* `pip_index_urls` is now considered also for containerized python component - the urls will be used for Dockerfile generation [\#8871](https://github.com/kubeflow/pipelines/pull/8871)
* Support direct indexing into top-level of artifact metadata struct in Container Components [\#9131](https://github.com/kubeflow/pipelines/pull/9131)
* Support compiling platform specific features [\#8940](https://github.com/kubeflow/pipelines/pull/8940)
* Support setting cpu/memory requests. [\#9121](https://github.com/kubeflow/pipelines/pull/9121)
* Support PIPELINE_ROOT_PLACEHOLDER [\#9134](https://github.com/kubeflow/pipelines/pull/9134)
* SDK client v2beta1 API integration [\#9112](https://github.com/kubeflow/pipelines/pull/9112)
* Support submitting pipeline with platform config. [\#9140](https://github.com/kubeflow/pipelines/pull/9140)

## Breaking changes
* New SDK client only works with Kubeflow Pipeline v2.0.0-beta.1 and later version [\#9112](https://github.com/kubeflow/pipelines/pull/9112)

## Deprecations
* Deprecate .add_node_selector_constraint in favor of .set_accelerator_type [\#8980](https://github.com/kubeflow/pipelines/pull/8980)

## Bug fixes and other changes
* Support python 3.11 [\#8907](https://github.com/kubeflow/pipelines/pull/8907)
* Fix loading non-canonical generic type strings from v1 component YAML (e.g., `List[str]`, `typing.List[str]`, `Dict[str]`, `typing.Dict[str, str]` [\#9041](https://github.com/kubeflow/pipelines/pull/9041)
* Add experiment_id parameter to create run methods [\#9004](https://github.com/kubeflow/pipelines/pull/9004)
* Support setting task dependencies via kfp.kubernetes.mount_pvc [\#8999](https://github.com/kubeflow/pipelines/pull/8999)
* cpu_limit and memory_limit can be optional [\#8992](https://github.com/kubeflow/pipelines/pull/8992)

## Documentation updates

# 2.0.0-beta.13

## Features
* Support fanning-in artifact outputs from a task in a `dsl.ParellelFor` context using `dsl.Collected` [\#8808](https://github.com/kubeflow/pipelines/pull/8808)
* Introduces a new syntax for pipeline tasks to consume outputs from the upstream task while at the same time ignoring if the upstream tasks succeeds or not. [\#8838](https://github.com/kubeflow/pipelines/pull/8838)

## Breaking changes

## Deprecations
* Deprecate pipeline task `.set_gpu_limit` in favor of `.set_accelerator_limit` [\#8836](https://github.com/kubeflow/pipelines/pull/8836)

## Bug fixes and other changes
* Enables output definitions when compiling components as pipelines. [\#8848](https://github.com/kubeflow/pipelines/pull/8848)
* Fix bug when passing data between tasks using f-strings [\#8879](https://github.com/kubeflow/pipelines/pull/8879)
* Fix environment variable set in component yaml lost during compilation [\#8885](https://github.com/kubeflow/pipelines/pull/8885)
* Fix attribute error when running Containerized Python Components [\#8887](https://github.com/kubeflow/pipelines/pull/8887)

## Documentation updates

# 2.0.0-beta.12

## Features
* Support fanning-in parameter outputs from a task in a `dsl.ParellelFor` context using `dsl.Collected` [\#8631](https://github.com/kubeflow/pipelines/pull/8631)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix upload_pipeline method on client when no name is provided [\#8695](https://github.com/kubeflow/pipelines/pull/8695)

## Documentation updates
# 2.0.0-beta.11

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Accepts `PyYAML<7` in addition to `PyYAML>=5.3,<6` [\#8665](https://github.com/kubeflow/pipelines/pull/8665)
* Remove v1 dependencies from SDK v2 [\#8668](https://github.com/kubeflow/pipelines/pull/8668)

## Documentation updates
# 2.0.0-beta.10

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fully support optional parameter inputs by witing `isOptional` field to IR [\#8612](https://github.com/kubeflow/pipelines/pull/8612)
* Add support for optional artifact inputs (toward feature parity with KFP SDK v1) [\#8623](https://github.com/kubeflow/pipelines/pull/8623)
* Fix bug deserializing v1 component YAML with boolean defaults, struct defaults, and array defaults [\#8639](https://github.com/kubeflow/pipelines/pull/8639)

## Documentation updates
# 2.0.0-beta.9

## Features
* Add comments to IR YAML file [\#8467](https://github.com/kubeflow/pipelines/pull/8467)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Unblock valid topologies [\#8416](https://github.com/kubeflow/pipelines/pull/8416)
* Fix bug when writing to same file using gcsfuse and distributed training strategy in lightweight/containerized Python components [#8544](https://github.com/kubeflow/pipelines/pull/8544) [alternative fix after [#8455](https://github.com/kubeflow/pipelines/pull/8455) in `kfp==2.0.0b8`], [#8607](https://github.com/kubeflow/pipelines/pull/8607)

## Documentation updates
# 2.0.0-beta.8

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Fix client methods [\#8507](https://github.com/kubeflow/pipelines/pull/8507)


## Documentation updates
# 2.0.0-beta.7

## Features
* Add ability to skip building image when using `kfp component build` [\#8387](https://github.com/kubeflow/pipelines/pull/8387)
* Support single element `then` and `else_` arguments to `IfPresentPlaceholder` [\#8414](https://github.com/kubeflow/pipelines/pull/8414)
* Enable use of input and output placeholders in f-strings [\#8494](https://github.com/kubeflow/pipelines/pull/8494)
## Breaking changes

## Deprecations

## Bug fixes and other changes
* Block illegal `IfPresentPlaceholder` and `ConcatPlaceholder` authoring [\#8414](https://github.com/kubeflow/pipelines/pull/8414)
* Fix boolean default value compilation bug [\#8444](https://github.com/kubeflow/pipelines/pull/8444)
* Fix bug when writing to same file using gcsfuse and distributed training strategy in lightweight/containerized Python components [\#8455](https://github.com/kubeflow/pipelines/pull/8455)


## Documentation updates
* Clarify `PipelineTask.set_gpu_limit` reference docs [\#8477](https://github.com/kubeflow/pipelines/pull/8477)

# 2.0.0-beta.6

## Features

## Breaking changes

## Deprecations

## Bug fixes and other changes

* Fix NamedTuple output with Dict/List bug [\#8316](https://github.com/kubeflow/pipelines/pull/8316)
* Fix PyPI typo in cli/component docstring [\#8361](https://github.com/kubeflow/pipelines/pull/8361)
* Fix "No KFP components found in file" error [\#8359](https://github.com/kubeflow/pipelines/pull/8359)
* CLI `kfp component build` generates runtime-requirements.txt [\#8372](https://github.com/kubeflow/pipelines/pull/8372)
* Throw exception for component parameter named Output [\#8367](https://github.com/kubeflow/pipelines/pull/8367)

## Documentation updates

* Improve KFP SDK reference documentation [\#8337](https://github.com/kubeflow/pipelines/pull/8337)

# 2.0.0-beta.5

## Features
* Support `google.`-namespaced artifact types [\#8191](https://github.com/kubeflow/pipelines/pull/8191), [\#8232](https://github.com/kubeflow/pipelines/pull/8232), [\#8233](https://github.com/kubeflow/pipelines/pull/8233), [\#8279](https://github.com/kubeflow/pipelines/pull/8279)
* Support dynamic importer metadata [\#7660](https://github.com/kubeflow/pipelines/pull/7660)

## Breaking changes

## Deprecations

## Bug fixes and other changes
* Migrate Out-Of-Band (OOB) authentication flow [\#8262](https://github.com/kubeflow/pipelines/pull/8262)

## Documentation updates
* Release KFP SDK v2 [user documentation draft](https://www.kubeflow.org/docs/components/pipelines/v2/)

# 2.0.0-beta.4

## Major Features and Improvements
* Support parallelism setting in ParallelFor [\#8146](https://github.com/kubeflow/pipelines/pull/8146)
* Support for Python v3.10 [\#8186](https://github.com/kubeflow/pipelines/pull/8186)
* Support pipeline as a component [\#8179](https://github.com/kubeflow/pipelines/pull/8179), [\#8204](https://github.com/kubeflow/pipelines/pull/8204), [\#8209](https://github.com/kubeflow/pipelines/pull/8209)
* Support using pipeline in exit handlers [\#8220](https://github.com/kubeflow/pipelines/pull/8220)

## Breaking Changes

### For Pipeline Authors
* Pipeline compilation is now triggered from `@pipeline` decorator instead of `Compiler.compile()` method.
Technically no breaking changes but compilation error could be exposed in a different (and earlier) stage. [\#8179](https://github.com/kubeflow/pipelines/pull/8179)

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Extend upper bound for Kubernetes to <24 in KFP SDK [\#8173](https://github.com/kubeflow/pipelines/pull/8173)

## Documentation Updates

# 2.0.0-beta.3

## Major Features and Improvements
* Add support for ConcatPlaceholder and IfPresentPlaceholder in containerized component [\#8145](https://github.com/kubeflow/pipelines/pull/8145)

## Breaking Changes

### For Pipeline Authors

### For Component Authors
## Deprecations

## Bug Fixes and Other Changes

## Documentation Updates

# 2.0.0-beta.2

## Major Features and Improvements

## Breaking Changes

### For Pipeline Authors

### For Component Authors
* Add support for containerized component [\#8066](https://github.com/kubeflow/pipelines/pull/8066)

## Deprecations

## Bug Fixes and Other Changes
* Enable overriding caching options at submission time [\#7912](https://github.com/kubeflow/pipelines/pull/7912)
* Allow artifact inputs in pipeline definition. [\#8044](https://github.com/kubeflow/pipelines/pull/8044)
* Support task-level retry policy [\#7867](https://github.com/kubeflow/pipelines/pull/7867)
* Support multiple exit handlers per pipeline [\#8088](https://github.com/kubeflow/pipelines/pull/8088)

## Documentation Updates

# 2.0.0-beta.1

## Major Features and Improvements

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes
* Include default registry context JSON in package distribution [\#7987](https://github.com/kubeflow/pipelines/pull/7987)

## Documentation Updates

# 2.0.0-beta.0

## Major Features and Improvements

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

## Documentation Updates

# 2.0.0-alpha.5

## Major Features and Improvements
* Implement Registry Client [\#7597](https://github.com/kubeflow/pipelines/pull/7597), [\#7763](https://github.com/kubeflow/pipelines/pull/7763)
* Write compiled JSON with formatting (multiline with indentation) [\#7712](https://github.com/kubeflow/pipelines/pull/7712)
* Add function to sdk client for terminating run [\#7835](https://github.com/kubeflow/pipelines/pull/7835)
* Re-enable component compilation via @component decorator (deprecated) [\#7554](https://github.com/kubeflow/pipelines/pull/7554)

## Breaking Changes
* Make CLI output consistent, readable, and usable [\#7739](https://github.com/kubeflow/pipelines/pull/7739)

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes
* Fix CLI upload pipeline version [\#7722](https://github.com/kubeflow/pipelines/pull/7722)

## Documentation Updates
# 2.0.0-alpha.3

## Major Features and Improvements
* feat(sdk): add `.list_pipeline_versions` and `.unarchive_experiment` methods to Client [\#7563](https://github.com/kubeflow/pipelines/pull/7563)
* Add additional methods to `kfp.client.Client` [\#7562](https://github.com/kubeflow/pipelines/pull/7562), [\#7463](https://github.com/kubeflow/pipelines/pull/7463)
* Migrate V1 CLI to V2, with improvements [\#7547](https://github.com/kubeflow/pipelines/pull/7547), [\#7558](https://github.com/kubeflow/pipelines/pull/7558), [\#7559](https://github.com/kubeflow/pipelines/pull/7559), [\#7560](https://github.com/kubeflow/pipelines/pull/7560), , [\#7569](https://github.com/kubeflow/pipelines/pull/7569), [\#7567](https://github.com/kubeflow/pipelines/pull/7567), [\#7603](https://github.com/kubeflow/pipelines/pull/7603), [\#7606](https://github.com/kubeflow/pipelines/pull/7606), [\#7607](https://github.com/kubeflow/pipelines/pull/7607), [\#7628](https://github.com/kubeflow/pipelines/pull/7628), [\#7618](https://github.com/kubeflow/pipelines/pull/7618)

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes
* Accepts `typing-extensions>=4,<5` in addition to `typing-extensions>=3.7.4,<4` [\#7632](https://github.com/kubeflow/pipelines/pull/7632)
* Remove dependency on `pydantic` [\#7639](https://github.com/kubeflow/pipelines/pull/7639)

## Documentation Updates

# 2.0.0-alpha.2

## Major Features and Improvements

* Enable pip installation from custom PyPI repository [\#7453](https://github.com/kubeflow/pipelines/pull/7453)

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Fix wrong kfp import causes wrong sdk_version being set in pipeline_spec. [\#7433](https://github.com/kubeflow/pipelines/pull/7433)
* Use YAML as default serialization format for package IR [\#7431](https://github.com/kubeflow/pipelines/pull/7431)
* Support submitting pipeline IR in yaml format via `kfp.client`. [\#7458](https://github.com/kubeflow/pipelines/pull/7458)
* Add pipeline_task_name to PipelineTaskFinalStatus [\#7464](https://github.com/kubeflow/pipelines/pull/7464)
* Depends on `kfp-pipeline-spec>=0.1.14,<0.2.0` [\#7464](https://github.com/kubeflow/pipelines/pull/7464)
* Depends on `google-cloud-storage>=2.2.1,<3` [\#7493](https://github.com/kubeflow/pipelines/pull/7493)

## Documentation Updates

# 1.8.12
## Major Features and Improvements

* Enable pip installation from custom PyPI repository [\#7470](https://github.com/kubeflow/pipelines/pull/7470)
* Support getting pipeline status in exit handler. [\#7483](https://github.com/kubeflow/pipelines/pull/7483)
## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes
* No longer require KFP client for kfp components build [\#7410](https://github.com/kubeflow/pipelines/pull/7410)
* Require google-api-core>=1.31.5, >=2.3.2 [#7377](https://github.com/kubeflow/pipelines/pull/7377)

## Documentation Updates

# 2.0.0-alpha.1

## Major Features and Improvements

## Breaking Changes

### For Pipeline Authors

### For Component Authors

## Deprecations

## Bug Fixes and Other Changes

* Depends on `kfp-server-api>=2.0.0a0, <3` [\#7427](https://github.com/kubeflow/pipelines/pull/7427)

## Documentation Updates

# 2.0.0-alpha.0

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
* Support KFP v2 API in kfp.client [\#7411](https://github.com/kubeflow/pipelines/pull/7411)

## Breaking Changes

* Remove sdk/python/kfp/v2/google directory for v2, including google client and custom job [\#6886](https://github.com/kubeflow/pipelines/pull/6886)
* APIs imported from the v1 namespace are no longer supported by the v2 compiler. [\#6890](https://github.com/kubeflow/pipelines/pull/6890)
* Deprecate v2 compatible mode in v1 compiler. [\#6958](https://github.com/kubeflow/pipelines/pull/6958)
* Drop support for python 3.6 [\#7303](https://github.com/kubeflow/pipelines/pull/7303)
* Deprecate v1 code to deprecated folder [\#7291](https://github.com/kubeflow/pipelines/pull/7291)
* Disable output_component_file temporarily for v2 early release [\#7390](https://github.com/kubeflow/pipelines/pull/7390)

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
* Fix bug that required KFP API server for `kfp components build` command to work [\#7430](https://github.com/kubeflow/pipelines/pull/7430)
* Pass default value for inputs and remove deprecated items in v1 [\#7405](https://github.com/kubeflow/pipelines/pull/7405)


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
