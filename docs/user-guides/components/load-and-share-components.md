# Load and Share Components

This section describes how to load and use existing components. In this section, "components" refers to both single-step components and pipelines, which can also be [used as components][pipeline-as-component].

IR YAML serves as a portable, sharable computational template. This allows you compile and share your components with others, as well as leverage an ecosystem of existing components.

To use an existing component, you can load it using the [`components`][components-module] module and use it with other components in a pipeline:

```python
from kfp import components

loaded_comp = components.load_component_from_file('component.yaml')

@dsl.pipeline
def my_pipeline():
    loaded_comp()
```

You can also load a component directly from a URL, such as a GitHub URL:

```python
loaded_comp = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/2.0.0/sdk/python/test_data/components/add_numbers.yaml')
```

Lastly, you can load a component from a string using [`components.load_component_from_text`][components-load-component-from-text]:

```python
with open('component.yaml') as f:
    component_str = f.read()

loaded_comp = components.load_component_from_text(component_str)
```

As components and pipelines are persisted in the same format (IR YAML), you can also load a pipeline from a local file, URL, or string, just like you load components. Once loaded, a pipeline can be used in another pipeline:

```python
from kfp import components

loaded_pipeline = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/2.0.0/sdk/python/test_data/pipelines/pipeline_in_pipeline_complex.yaml')

@dsl.pipeline
def my_pipeline():
    loaded_pipeline(msg='Hello KFP v2!')
```

Some libraries, such as [Google Cloud Pipeline Components][gcpc] package and provide reusable components in a pip-installable [Python package][gcpc-pypi].

[pipeline-as-component]: compose-components-into-pipelines.md#pipelines-as-components
[gcpc]: https://cloud.google.com/vertex-ai/docs/pipelines/components-introduction
[gcpc-pypi]: https://pypi.org/project/google-cloud-pipeline-components/
[components-module]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/components.html
[components-load-component-from-text]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/components.html#kfp.components.load_component_from_text
