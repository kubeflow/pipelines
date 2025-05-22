Overview
========

What is Kubeflow Pipelines?
----------------------------

Kubeflow Pipelines (KFP) is a platform for building and deploying portable and scalable machine learning (ML) workflows using containers on Kubernetes-based systems.

With KFP you can author `components`_ and `pipelines`_ using the `KFP Python SDK`_, compile pipelines to an `intermediate representation YAML`_, and submit the pipeline to run on a KFP-conformant backend such as the `open source KFP backend`_ or `Google Cloud Vertex AI Pipelines <https://cloud.google.com/vertex-ai/docs/pipelines/introduction>`_.

The open source KFP backend is available as a core component of Kubeflow or as a standalone installation. To use KFP as part of the Kubeflow platform, follow the instructions for `installing Kubeflow`_. To use KFP as a standalone application, follow the `standalone installation`_ instructions. To get started with your first pipeline, follow the `Getting Started`_ instructions.

Why Kubeflow Pipelines?
-----------------------

KFP enables data scientists and machine learning engineers to:

* Author end-to-end ML workflows natively in Python
* Create fully custom ML components or leverage an ecosystem of existing components
* Easily pass parameters and ML artifacts between pipeline components
* Easily manage, track, and visualize pipeline definitions, runs, experiments, and ML artifacts
* Efficiently use compute resources through parallel task execution and through caching to eliminate redundant executions
* Keep experimentation and iteration light and Python-centric, minimizing the need to (re)build and maintain containers
* Maintain cross-platform pipeline portability through a platform-neutral `IR YAML pipeline definition`_

What is a pipeline?
-------------------

A `pipeline`_ is a definition of a workflow that composes one or more `components`_ together to form a computational directed acyclic graph (DAG). At runtime, each component execution corresponds to a single container execution, which may create ML artifacts. Pipelines may also feature `control flow`_.

