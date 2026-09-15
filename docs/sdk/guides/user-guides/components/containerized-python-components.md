# Containerized Python Components

The following assumes a basic familiarity with [Lightweight Python Components][lightweight-python-components].

Containerized Python Components extend [Lightweight Python Components][lightweight-python-components] by relaxing the constraint that Lightweight Python Components be hermetic (i.e., fully self-contained). This means Containerized Python Component functions can depend on symbols defined outside of the function, imports outside of the function, code in adjacent Python modules, etc. To achieve this, the KFP SDK provides a convenient way to package your Python code into a container.

As a production software best practice, component authors should prefer Containerized Python Components to [Lightweight Python Components][lightweight-python-components] when their component specifies [`packages_to_install`][packages-to-install], since the KFP SDK will install these dependencies into the component's image when it is built, rather than at task runtime.

The following shows how to use Containerized Python Components by modifying the `add` component from the [Lightweight Python Components][lightweight-python-components] example:

```python
from kfp import dsl

@dsl.component
def add(a: int, b: int) -> int:
    return a + b
```

## 1. Source code setup
Start by creating an empty `src/` directory to contain your source code:

```text
src/
```

Next, add the following simple module, `src/math_utils.py`, with one helper function:

```python
# src/math_utils.py
def add_numbers(num1, num2):
    return num1 + num2
```

Lastly, move your component to `src/my_component.py` and modify it to use the helper function:

```python
# src/my_component.py
from kfp import dsl

@dsl.component
def add(a: int, b: int) -> int:
    from math_utils import add_numbers
    return add_numbers(a, b)
```

`src` now looks like this:

```text
src/
├── my_component.py
└── math_utils.py
```

### 2. Modify the dsl.component decorator

In this step, you'll provide `base_image` and `target_image` arguments to the `@dsl.component` decorator of your component in `src/my_component.py`:

```python
@dsl.component(base_image='python:3.11',
               target_image='gcr.io/my-project/my-component:v1')
def add(a: int, b: int) -> int:
    from math_utils import add_numbers
    return add_numbers(a, b)
```

Setting `target_image` both (a) specifies the [tag][image-tag] for the image you'll build in Step 3, and (b) instructs KFP to run the decorated Python function in a container that uses the image with that tag.

In a Containerized Python Component, `base_image` specifies the base image that KFP will use when building your new container image. Specifically, KFP uses the `base_image` argument for the [`FROM`][docker-from] instruction in the Dockerfile used to build your image.

The previous example includes `base_image` for clarity, but this is not necessary as `base_image` will default to `'python:3.11'` if omitted.

### 3. Build the component
Now that your code is in a standalone directory and you've specified a target image, you can conveniently build an image using the [`kfp component build`][kfp-component-build] CLI command:

```sh
kfp component build src/ --component-filepattern my_component.py --no-push-image
```

If you have a [configured Docker to use a private image registry](https://docs.docker.com/engine/reference/commandline/login/), you can replace the `--no-push-image` flag with `--push-image` to automatically push the image after building.

### 4. Use the component in a pipeline

Finally, you can use the component in a pipeline:

```python
# pipeline.py
from kfp import compiler, dsl
from src.my_component import add

@dsl.pipeline
def addition_pipeline(x: int, y: int) -> int:
    task1 = add(a=x, b=y)
    task2 = add(a=task1.output, b=x)
    return task2.output

compiler.Compiler().compile(addition_pipeline, 'pipeline.yaml')
```

Your directory now looks like this:

```text
pipeline.py
src/
├── my_component.py
└── math_utils.py
```

Since `add`'s `target_image` uses [Google Cloud Artifact Registry][artifact-registry] (indicated by the `gcr.io` URI), the pipeline shown here assumes you have pushed your image to Google Cloud Artifact Registry, you are running your pipeline on [Google Cloud Vertex AI Pipelines][vertex-pipelines], and you have configured [IAM permissions][iam] so that Vertex AI Pipelines can pull images from Artifact Registry.


[kfp-component-build]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/cli.html#kfp-component-build
[lightweight-python-components]: lightweight-python-components.md
[image-tag]: https://docs.docker.com/engine/reference/commandline/tag/
[docker-from]: https://docs.docker.com/engine/reference/builder/#from
[artifact-registry]: https://cloud.google.com/artifact-registry/docs/docker/authentication
[vertex-pipelines]: https://cloud.google.com/vertex-ai/docs/pipelines/introduction
[iam]: https://cloud.google.com/iam
[packages-to-install]: lightweight-python-components.md#packages_to_install
