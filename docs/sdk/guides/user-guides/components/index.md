# Create components

Components are the building blocks of KFP pipelines. A component is a remote function definition; it specifies inputs, has user-defined logic in its body, and can create outputs. When the component template is instantiated with input parameters, we call it a task.

KFP provides two high-level ways to author components: **Python Components** and **Container Components.**

Python Components are a convenient way to author components implemented in pure Python. There are two specific types of Python components: **Lightweight Python Components** and **Containerized Python Components.**

Container Components expose a more flexible, advanced authoring approach by allowing you to define a component using an arbitrary container definition. This is the recommended approach for components that are not implemented in pure Python.

**Importer Components** are a special "pre-baked" component provided by KFP which allows you to import an artifact into your pipeline when that artifact was not created by tasks within the pipeline.

:::{toctree}
:maxdepth: 2

lightweight-python-components
compose-components-into-pipelines
containerized-python-components
container-components
notebook-component
importer-component
additional-functionality
load-and-share-components
:::
