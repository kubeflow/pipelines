Overview
========

What is Kubeflow Pipelines?
----------------------------

Kubeflow Pipelines (KFP) is a platform for building and deploying portable and scalable machine learning (ML) workflows using containers on Kubernetes-based systems.
With KFP you can author :ref:`components <what-is-a-component>` and :ref:`pipelines <what-is-a-pipeline>` using the :ref:`KFP Python SDK <kfp-python-sdk>`, compile pipelines 
to an :ref:`intermediate representation YAML <what-is-a-compiled-pipeline>`, and submit the pipeline to run on a KFP-conformant backend such as the :ref:`open source KFP backend <open-source-deployment>`, `Google Cloud Vertex AI Pipelines <https://cloud.google.com/vertex-ai/docs/pipelines/introduction>`_, or KFP local.

The open source KFP backend is available as a core component of Kubeflow or as a standalone installation. 

Why Kubeflow Pipelines?
-----------------------

KFP enables data scientists and machine learning engineers to:

* Author end-to-end ML workflows natively in Python
* Create fully custom ML components or leverage an ecosystem of existing components
* Easily pass parameters and ML artifacts between pipeline components
* Easily manage, track, and visualize pipeline definitions, runs, experiments, and ML artifacts
* Efficiently use compute resources through parallel task execution and through caching to eliminate redundant executions
* Keep experimentation and iteration light and Python-centric, minimizing the need to (re)build and maintain containers
* Maintain cross-platform pipeline portability through a platform-neutral IR YAML pipeline definition
* Abstract Kubernetes complexity while running pipelines on your organization's existing infrastructure investments (on-prem, cloud, or hybrid)

.. _what-is-a-pipeline:

What is a pipeline?
-------------------

A `pipeline` is a definition of a workflow that composes one or more `components` together to form a computational directed acyclic graph (DAG). At runtime, each component execution corresponds to a single container execution, which may create ML artifacts. Pipelines may also feature `control flow`.

.. _what-is-a-component:

What is a component?
--------------------
Components are the building blocks of KFP pipelines. A component is a remote function definition; it specifies inputs, has user-defined logic in its body, and can create outputs. When the component template is instantiated with input parameters, we call it a task.

KFP provides two high-level ways to author components: Python Components and Container Components.

Python Components are a convenient way to author components implemented in pure Python. There are two specific types of Python components: Lightweight Python Components and Containerized Python Components.

Container Components expose a more flexible, advanced authoring approach by allowing you to define a component using an arbitrary container definition. This is the recommended approach for components that are not implemented in pure Python.

Importer Components are a special "pre-baked" component provided by KFP which allows you to import an artifact into your pipeline when that artifact was not created by tasks within the pipeline.

.. _what-is-a-compiled-pipeline:

What is a compiled pipeline?
----------------------------
A compiled pipeline, often referred to as an IR YAML, is an intermediate representation (IR) of a compiled pipeline or component. The IR YAML is not intended to be written directly.

While IR YAML is not intended to be easily human-readable, you can still inspect it if you know a bit about its contents:

.. _pipelines: #what-is-a-pipeline
.. _components: #what-is-a-component
.. _compiled-pipeline: #what-is-a-compiled-pipeline
