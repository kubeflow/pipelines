Getting Started
===============

Create your first pipeline
--------------------------

Install the KFP v2 SDK:

.. code-block:: bash

   pip install kfp

The following simple pipeline prints a greeting:

.. code-block:: python

   from kfp import dsl

   @dsl.component
   def say_hello(name: str) -> str:
       hello_text = f'Hello, {name}!'
       print(hello_text)
       return hello_text

   @dsl.pipeline
   def hello_pipeline(recipient: str) -> str:
       hello_task = say_hello(name=recipient)
       return hello_task.output

The :doc:`../source/dsl` decorators turn type-annotated Python functions into
components and pipelines. Compile the pipeline to a self-contained pipeline
YAML file with the :doc:`../source/compiler`:

.. code-block:: python

   from kfp import compiler

   compiler.Compiler().compile(hello_pipeline, 'pipeline.yaml')

Submit the YAML file to a KFP-conformant backend with the
:doc:`../source/client`. After deploying a KFP backend and obtaining its
endpoint, run the following:

.. code-block:: python

   from kfp.client import Client

   client = Client(host='<MY-KFP-ENDPOINT>')
   run = client.create_run_from_pipeline_package(
       'pipeline.yaml',
       arguments={
           'recipient': 'World',
       },
   )

The client prints a link to the pipeline execution graph and logs in the UI.
This pipeline has one task that prints and returns ``Hello, World!``.

Next steps
----------

Learn more about `connecting the KFP SDK to the API <https://www.kubeflow.org/docs/components/pipelines/user-guides/core-functions/connect-api/>`_.
For additional authoring and execution guides that have not yet migrated here,
see the `Kubeflow Pipelines user guides <https://www.kubeflow.org/docs/components/pipelines/user-guides/>`_.
