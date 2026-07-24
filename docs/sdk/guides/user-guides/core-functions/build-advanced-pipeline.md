# Build a More Advanced ML Pipeline

This step demonstrates how to build a more advanced machine learning (ML) pipeline that leverages additional KFP pipeline composition features.

The following ML pipeline creates a dataset, normalizes the features of the dataset as a preprocessing step, and trains a simple ML model on the data using different hyperparameters:

```python
from typing import List

from kfp import client
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Model
from kfp.dsl import Output


@dsl.component(packages_to_install=['pandas==1.3.5'])
def create_dataset(iris_dataset: Output[Dataset]):
    import pandas as pd

    csv_url = 'https://archive.ics.uci.edu/ml/machine-learning-databases/iris/iris.data'
    col_names = [
        'Sepal_Length', 'Sepal_Width', 'Petal_Length', 'Petal_Width', 'Labels'
    ]
    df = pd.read_csv(csv_url, names=col_names)

    with open(iris_dataset.path, 'w') as f:
        df.to_csv(f)


@dsl.component(packages_to_install=['pandas==1.3.5', 'scikit-learn==1.0.2'])
def normalize_dataset(
    input_iris_dataset: Input[Dataset],
    normalized_iris_dataset: Output[Dataset],
    standard_scaler: bool,
    min_max_scaler: bool,
):
    if standard_scaler is min_max_scaler:
        raise ValueError(
            'Exactly one of standard_scaler or min_max_scaler must be True.')

    import pandas as pd
    from sklearn.preprocessing import MinMaxScaler
    from sklearn.preprocessing import StandardScaler

    with open(input_iris_dataset.path) as f:
        df = pd.read_csv(f)
    labels = df.pop('Labels')

    if standard_scaler:
        scaler = StandardScaler()
    if min_max_scaler:
        scaler = MinMaxScaler()

    df = pd.DataFrame(scaler.fit_transform(df))
    df['Labels'] = labels
    with open(normalized_iris_dataset.path, 'w') as f:
        df.to_csv(f)


@dsl.component(packages_to_install=['pandas==1.3.5', 'scikit-learn==1.0.2'])
def train_model(
    normalized_iris_dataset: Input[Dataset],
    model: Output[Model],
    n_neighbors: int,
):
    import pickle

    import pandas as pd
    from sklearn.model_selection import train_test_split
    from sklearn.neighbors import KNeighborsClassifier

    with open(normalized_iris_dataset.path) as f:
        df = pd.read_csv(f)

    y = df.pop('Labels')
    X = df

    X_train, X_test, y_train, y_test = train_test_split(X, y, random_state=0)

    clf = KNeighborsClassifier(n_neighbors=n_neighbors)
    clf.fit(X_train, y_train)
    with open(model.path, 'wb') as f:
        pickle.dump(clf, f)


@dsl.pipeline(name='iris-training-pipeline')
def my_pipeline(
    standard_scaler: bool,
    min_max_scaler: bool,
    neighbors: List[int],
):
    create_dataset_task = create_dataset()

    normalize_dataset_task = normalize_dataset(
        input_iris_dataset=create_dataset_task.outputs['iris_dataset'],
        standard_scaler=True,
        min_max_scaler=False)

    with dsl.ParallelFor(neighbors) as n_neighbors:
        train_model(
            normalized_iris_dataset=normalize_dataset_task
            .outputs['normalized_iris_dataset'],
            n_neighbors=n_neighbors)


endpoint = '<KFP_UI_URL>'
kfp_client = client.Client(host=endpoint)
run = kfp_client.create_run_from_pipeline_func(
    my_pipeline,
    arguments={
        'min_max_scaler': True,
        'standard_scaler': False,
        'neighbors': [3, 6, 9]
    },
)
url = f'{endpoint}/#/runs/details/{run.run_id}'
print(url)
```

This example introduces the following new features in the pipeline:

* Some Python **packages to install** are added at component runtime, using the `packages_to_install` argument on the `@dsl.component` decorator, as follows:

    `@dsl.component(packages_to_install=['pandas==1.3.5'])`

    To use a library after installing it, you must include its import statements within the scope of the component function, so that the library is imported at component runtime.

* **Input and output artifacts** of types `Dataset` and `Model` are introduced in the component signature to describe the input and output artifacts of the components. This is done using the type annotation generics `Input[]` and `Output[]` for input and output artifacts respectively.

  Within the scope of a component, artifacts can be read (for inputs) and written (for outputs) via the `.path` attribute. The KFP backend ensures that *input* artifact files are copied *to* the executing pod's local file system from the remote storage at runtime, so that the component function can read input artifacts from the local file system. By comparison, *output* artifact files are copied *from* the local file system of the pod to remote storage, when the component finishes running. This way, the output artifacts persist outside the pod. In both cases, the component author needs to interact with the local file system only to create persistent artifacts.

  The arguments for the parameters annotated with `Output[]` are not passed to components by the pipeline author. The KFP backend passes this artifact during component runtime, so that component authors don't need to be concerned about the path to which the output artifacts are written. After an output artifact is written, the backend executing the component recognizes the KFP artifact types (`Dataset` or `Model`), and organizes them on the Dashboard.

  An output artifact can be passed as an input to a downstream component using the `.outputs` attribute of the source task and the output artifact parameter name, as follows:

  `create_dataset_task.outputs['iris_dataset']`

* One of the **DSL control flow features**, `dsl.ParallelFor`, is used. It is a context manager that lets pipeline authors create tasks. These tasks execute in parallel in a loop. Using `dsl.ParallelFor` to iterate over the `neighbors` pipeline argument lets you execute the  `train_model` component with different arguments and test multiple hyperparameters in one pipeline run. Other control flow features include `dsl.Condition` and `dsl.ExitHandler`.

Congratulations! You now have a KFP deployment, an end-to-end ML pipeline, and an introduction to the UI. That's just the beginning of KFP pipeline and Dashboard features.
