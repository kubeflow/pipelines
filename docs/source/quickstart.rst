Quickstart
==========

This guide shows how to create, compile, and run a simple pipeline with Kubeflow Pipelines (KFP).

Prerequisites
-------------
- Python 3.9+
- A running Kubeflow Pipelines deployment (local or remote).

Installation
------------
Install the Kubeflow Pipelines SDK:

.. code-block:: bash

   pip install kfp

Local Initialization
--------------------
Use the `SubprocessRunner` for local execution without Docker:

.. code-block:: python

   from kfp import local
   local.init(runner=local.SubprocessRunner())

Writing a Simple Component
--------------------------

Define a lightweight component using the ``@dsl.component`` decorator:

.. code-block:: python

   from kfp import dsl

   @dsl.component
   def say_hello(name: str) -> str:
       message = f"Hello, {name}!"
       print(message)
       return message

You can run this component directly like a Python function:

.. code-block:: python

   task = say_hello(name="World")
   assert task.output == "Hello, World!"

Writing and Running a Pipeline
------------------------------

Define a pipeline using the ``@dsl.pipeline`` decorator:

.. code-block:: python

   @dsl.pipeline
   def hello_pipeline(recipient: str) -> str:
       hello_task = say_hello(name=recipient)
       return hello_task.output

Run the pipeline locally as a regular function:

.. code-block:: python

   pipeline_task = hello_pipeline(recipient="Local Dev")
   assert pipeline_task.output == "Hello, Local Dev!"


The ``@dsl.component`` and ``@dsl.pipeline`` decorators turn type-annotated Python functions into reusable pipeline components and workflows.

Working with Artifacts
----------------------

You can also write artifacts to disk and read them locally:

.. code-block:: python

   from kfp.dsl import Output, Artifact
   import json

   @dsl.component
   def add(a: int, b: int, out_artifact: Output[Artifact]):
       result = a + b
       with open(out_artifact.path, 'w') as f:
           f.write(json.dumps(result))
       out_artifact.metadata['operation'] = 'addition'

   task = add(a=1, b=2)
   with open(task.outputs['out_artifact'].path) as f:
       result = json.loads(f.read())

   assert result == 3
   assert task.outputs['out_artifact'].metadata['operation'] == 'addition'


Running the pipeline
----------------------
You can run the pipeline locally with Python:

.. code-block:: bash

   python my_pipeline.py


Next steps
----------
- Explore the DSL: :doc:`dsl`
- Learn about Components: :doc:`components`
- See the CLI reference: :doc:`cli`
