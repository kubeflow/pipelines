# Kubeflow Pipelines 2019 Roadmap

## Overview

This document outlines the main directions on the Kubeflow Pipelines (KFP) project in 2019.

## Production Readiness

We will continue developing capabilities for better reliability, scaling, and maintenance of production ML systems built with Kubeflow Pipelines.

* Ability to easily upgrade KFP system components to new versions and apply fixes to a live cluster without losing state
* Ability to externalize the critical metadata state to a data store outside of the cluster lifetime
* Ability to configure a standard cluster-wide persistent storage that all pipelines can share, connected to any cloud or on-prem storage system
* Easy deployment of KFP system services

## Connector Components

To make it easy to use KFP within an ecosystem of other cloud services, and to take advantage of scale and other capabilities of job scheduling services and data processing services - KFP components will build a framework for reliable connections to other services. Google will extend the framework and contribute a few specific connector components:

* Connectors to DataProc (Spark), DataFlow (Beam), BigQuery, Cloud ML Engine


## Metadata Store and API

As a foundational layer in the ML system, KFP will introduce an extensible and scalable metadata store for tracking versioning, dependencies, and provenance of artifacts and executables. The metadata store will be usable from any other KF component to help users easily connect artifacts to their origins, metrics, and effects and consumption points. 

* Metadata Store and API
* Automatic tracking of pipelines, pipeline steps, parameters, and artifacts 
* Extensible Type System and Standard Types for most common ML artifacts (models, datasets, metrics, visualizations)

## Shareable Components and Pipelines Model

To make it easy for users to share and consume KFP components within and outside of an organization, KFP will improve the sharing capabilities in the KFP SDK:

* Component configuration for easy sharing of components through file sharing and source control
* Ability to represent a pipeline as a component for use in other pipelines


## Enhanced UI and Notebooks

KFP UI will continue to improve so that operating KFP clusters and managing KFP resources is more intuitive:

* Metadata UI to provide an exploration and search experience over artifacts and types
* Ability to use Notebooks outside of the K8S cluster to build and control pipeline execution
* Controls for viewing pipeline topology and execution results within Notebooks


## Pipeline execution and debugging

To make it more efficient to run ML experiments, KFP will add features for faster iteration over experiments, better control, and transparency of the execution engine:

* Support for caching of pipeline artifacts and the ability to use the artifacts cache to accelerate pipeline re-execution. This will allow steps that have already executed to be skipped on subsequent runs.
* Ability to stop/restart pipeline jobs
* Ability to track pipeline dependencies and resources created/used by a pipeline job


## Data and Event driven scheduling

Many ML workflows make more sense to be triggered by data availability or by external events rather than scheduled manually. KFP will have native support for data driven and event driven workflows. KFP will provide the ability to configure pipeline execution upon appearance of certain entries in the metadata store, making it easy to create complex CI pipelines orchestrated around key artifacts, such as models. 


## Enhanced DSL workflow control

The KFP SDK for defining the pipeline topologies and component dependencies will add more advanced control operators for organizing workflow loops, parallel for-each, and enhanced conditions support.


<EOD>
