# Kubeflow Pipelines Roadmap

## Kubeflow Pipelines 2021 Roadmap (major themes)

### KFP v2 compatible

Quick links:

* Design: [bit.ly/kfp-v2-compatible](https://bit.ly/kfp-v2-compatible)
* [Tracker Project](https://github.com/kubeflow/pipelines/projects/13)
* [Documentation](https://www.kubeflow.org/docs/components/pipelines/sdk/v2/v2-compatibility/)

#### Goals - v2 compatible

* Enables v2 DSL core features for early user feedback, while retaining backward compatibility to most v1 features.
* Stepping stone for v1 to v2 migration.
* Validate v2 system design perf & scalability requirements.

#### Timeline - v2 compatible

* First beta release late May
* Feature complete mid July

#### New Features in v2 compatible

* Improved Artifact Passing

  Improvements/changes below make KFP artifacts easier to integrate with other systems:
  * In components, support consuming input artifacts by URI. This is useful for components that launch external jobs using artifact URIs, but do not need to access the data directly by themselves.
  * A new intermediate artifact repository feature is designed -- pipeline root. It is configurable at:
    * Cluster default
    * Namespace defaults
    * Authoring pipelines
    * Submitting a pipeline
  * Pipeline root supports MinIO, S3, GCS natively using Go CDK.
  * Artifacts are no longer compressed by default.
* Artifacts with metadata
  * Support for components that can consume MLMD metadata.
  * Support for components that can produce/update MLMD-based metadata.
* Visualizations
  * [#5668](https://github.com/kubeflow/pipelines/issues/5668) Visualize v2 metrics -- components can output metrics artifacts that are rendered in UI. [sample pipeline](https://github.com/kubeflow/pipelines/blob/307e91aaae5e9c71dde1fddaffa10ffd751a40e8/samples/test/metrics_visualization_v2.py#L103)
  * [#3970](https://github.com/kubeflow/pipelines/issues/3970) Easier visualizations: HTML, Markdown, etc using artifact type + metadata.
* v2 python components.
  * A convenient component authoring method designed to support new features above natively. (v1 components do not completely support all the features mentioned above)
  * [samples/test/lightweight_python_functions_v2_with_outputs.py](https://github.com/kubeflow/pipelines/blob/master/samples/test/lightweight_python_functions_v2_with_outputs.py)
  * [samples/test/lightweight_python_functions_v2_pipeline.py](https://github.com/kubeflow/pipelines/blob/master/samples/test/lightweight_python_functions_v2_pipeline.py)
* [#5669](https://github.com/kubeflow/pipelines/issues/5669) KFP semantics in MLMD
  * MLMD state with exact KFP semantics (e.g. parameter / artifact / task names are the same as in DSL). This will enable use-cases like: “querying a result artifact from a pipeline run using MLMD API and then use the result in another system or another pipeline”.
  
    [Example pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/test/lightweight_python_functions_v2_pipeline.py) and [corresponding MLMD state](https://github.com/kubeflow/pipelines/blob/master/samples/test/lightweight_python_functions_v2_pipeline_test.py).
* [#5670](https://github.com/kubeflow/pipelines/issues/5670) Revamp KFP UI to show inputs & outputs in KFP semantics for v2 compatible pipelines
* [#5667](https://github.com/kubeflow/pipelines/issues/5667) KFP data model based caching using MLMD.

### KFP v2

Design: [bit.ly/kfp-v2](https://bit.ly/kfp-v2)

#### KFP v2 Goals

* Data Management:
  * build first class support for metadata -- recording, presentation and orchestration.
  * making it easy to keep track of all the data produced by machine learning pipelines and how it was computed.
* KFP native (and argo agnostic) spec and status: define a clear interface for KFP, so that other systems can understand KFP pipeline spec and status in KFP semantics.
* Gaining more control over KFP runtime behavior, so that it sets up a solid foundation for us to add new features to KFP: give the KFP system more control over the exact runtime behavior. This wasn’t a goal initially coming from use cases. However, the more we innovate on Data Management and KFP native spec/status, the clearer that other workflow systems become limitations to how we may implement new KFP features on our own. Therefore, re-architecturing KFP to let us get more control of runtime behavior is ideal for achieving KFP’s long term goals.
* Be backward compatible with KFP v1: for existing features, we want to keep them as backward compatible as possible to ease upgrade.

#### Timeline - v2

* Start work after v2 compatible feature complete
* Alpha release in October
* Beta/Stable release timelines TBD

#### Planned Features in KFP v2

* KFP v2 DSL (TBD)

* Use [the pipeline spec](https://github.com/kubeflow/pipelines/blob/master/api/kfp_pipeline_spec/pipeline_spec.proto) as pipeline description data structure. The new spec is argo workflow agnostic and can be a shared common format for different underlying engines.

* Design and implement a pipeline run status API (also argo agnostic).

* KFP v2 DAG UI
  * KFP semantics.
  * Convenient features: panning, zooming, etc.

* Control flow features
  * Reusable subgraph component.
  * Subgraph component supports return value and aggregating from parallel for.

* Caching improvement: skipped tasks will not execute at all (in both v1 and v2 compatible, skipped tasks will still run a Pod which does not do anything).

* TFX on KFP v2.

### Other Items

* [#3857](https://github.com/kubeflow/pipelines/issues/3857) Set up a Vulnerability Scanning Process.

## Kubeflow Pipelines 2019 Roadmap

### 2019 Overview

This document outlines the main directions on the Kubeflow Pipelines (KFP) project in 2019.

### Production Readiness

We will continue developing capabilities for better reliability, scaling, and maintenance of production ML systems built with Kubeflow Pipelines.

* Ability to easily upgrade KFP system components to new versions and apply fixes to a live cluster without losing state
* Ability to externalize the critical metadata state to a data store outside of the cluster lifetime
* Ability to configure a standard cluster-wide persistent storage that all pipelines can share, connected to any cloud or on-prem storage system
* Easy deployment of KFP system services

### Connector Components

To make it easy to use KFP within an ecosystem of other cloud services, and to take advantage of scale and other capabilities of job scheduling services and data processing services - KFP components will build a framework for reliable connections to other services. Google will extend the framework and contribute a few specific connector components:

* Connectors to DataProc (Spark), DataFlow (Beam), BigQuery, Cloud ML Engine

### Metadata Store and API

As a foundational layer in the ML system, KFP will introduce an extensible and scalable metadata store for tracking versioning, dependencies, and provenance of artifacts and executables. The metadata store will be usable from any other KF component to help users easily connect artifacts to their origins, metrics, and effects and consumption points.

* Metadata Store and API
* Automatic tracking of pipelines, pipeline steps, parameters, and artifacts 
* Extensible Type System and Standard Types for most common ML artifacts (models, datasets, metrics, visualizations)

### Shareable Components and Pipelines Model

To make it easy for users to share and consume KFP components within and outside of an organization, KFP will improve the sharing capabilities in the KFP SDK:

* Component configuration for easy sharing of components through file sharing and source control
* Ability to represent a pipeline as a component for use in other pipelines

### Enhanced UI and Notebooks

KFP UI will continue to improve so that operating KFP clusters and managing KFP resources is more intuitive:

* Metadata UI to provide an exploration and search experience over artifacts and types
* Ability to use Notebooks outside of the K8S cluster to build and control pipeline execution
* Controls for viewing pipeline topology and execution results within Notebooks

### Pipeline execution and debugging

To make it more efficient to run ML experiments, KFP will add features for faster iteration over experiments, better control, and transparency of the execution engine:

* Support for caching of pipeline artifacts and the ability to use the artifacts cache to accelerate pipeline re-execution. This will allow steps that have already executed to be skipped on subsequent runs.
* Ability to stop/restart pipeline jobs
* Ability to track pipeline dependencies and resources created/used by a pipeline job

### Data and Event driven scheduling

Many ML workflows make more sense to be triggered by data availability or by external events rather than scheduled manually. KFP will have native support for data driven and event driven workflows. KFP will provide the ability to configure pipeline execution upon appearance of certain entries in the metadata store, making it easy to create complex CI pipelines orchestrated around key artifacts, such as models. 

### Enhanced DSL workflow control

The KFP SDK for defining the pipeline topologies and component dependencies will add more advanced control operators for organizing workflow loops, parallel for-each, and enhanced conditions support.

<EOD>
