Quickstart
==========

This fork keeps the KFP Python DSL and compiles native Argo resources. Components
run on Kubernetes; calling a component outside a pipeline does not execute it.

Prerequisites
-------------

* Python 3.11 or newer and this checkout's SDK.
* Argo Workflows with an artifact repository and workflow service account.
* A component image containing this checkout's runtime. The executable setup in
  ``samples/argo-native/README.md`` builds ``kfp-argo-demo:dev`` from the SDK wheel.

Write and compile a pipeline
----------------------------

.. code-block:: python

   from kfp import compiler, dsl

   @dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
   def say_hello(name: str) -> str:
       message = f'Hello, {name}!'
       print(message)
       return message

   @dsl.pipeline(name='hello-pipeline')
   def hello_pipeline(recipient: str = 'World') -> str:
       task = say_hello(name=recipient).set_caching_options(False)
       return task.output

   compiler.Compiler().compile(hello_pipeline, 'pipeline.yaml')

The output is a content-addressed ``WorkflowTemplate``. To emit an individual
run, pass ``resource_kind='Workflow'``. A recurring run uses
``resource_kind='CronWorkflow', schedule='0 9 * * *'`` and starts suspended.

.. code-block:: bash

   argo lint --offline pipeline.yaml
   kubectl -n argo apply -f pipeline.yaml
   argo submit -n argo --from workflowtemplate/NAME_FROM_PIPELINE_YAML --watch

Caching requires immutable image identity: pin the image by digest or supply the
image's digest through ``Compiler(image_digests={...})``. A mutable development
image should disable caching unless its exact build digest is supplied.

Typed artifacts
---------------

Components continue to use ``Input[Dataset]``, ``Output[Model]``, ``artifact.path``,
``artifact.uri`` and ``artifact.metadata``. Argo stages and transfers payloads; the
in-container runtime invokes the Python function and records typed output metadata.
See ``samples/argo-native/components.py`` for a complete dataset/model/metrics
pipeline and its end-to-end verification commands.

Next steps
----------

* Explore the DSL: :doc:`dsl`
* Learn about components: :doc:`components`
* See the CLI reference: :doc:`cli`
