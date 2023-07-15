# list before
from typing import List

from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Model
from kfp.dsl import Output


@dsl.component
def artifact_component(
    datasets: Input[List[Dataset]],
    models: Output[List[Model]],
):
    for i, dataset in enumerate(datasets):
        model = models.create()
        model.metadata['data'] = dataset.name

        # model training code omitted
        trained_model.save(model.path)


@dsl.pipeline
def artifact_pipeline(datasets: Input[List[Dataset]]) -> List[Model]:
    # see definition of artifact_component above
    return artifact_component(datasets=datasets).outputs['models']


# list -- after

from typing import List

from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Model


@dsl.component
def artifact_component(datasets: List[Dataset]) -> List[Model]:
    models = []
    for i, dataset in enumerate(datasets):
        with open(dataset.path) as f:
            data = f.read()

        # the mechanism for obtaining a system-generated path is discussed below
        uri = dsl.get_uri(suffix=f'model/{i}')
        model = Model(name='my_model', uri=uri, metadata={'data': dataset.name})

        # model training code omitted
        trained_model.save(model.path)

        models.append(model)

    return models


@dsl.pipeline
def artifact_pipeline(datasets: List[Dataset]) -> List[Model]:
    # see definition of artifact_component above
    return artifact_component(datasets=datasets).output


# multiple returns -- BEFORE
import typing

from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Model


@dsl.component
def artifact_component(dataset: Input[Dataset], model: Output[Model]) -> int:
    with open(dataset.path) as f:
        data = f.read()

    # the mechanism for obtaining a system-generated path is discussed below
    uri = ...
    model = Model(name='my_model', uri=uri, metadata={'data': dataset.name})

    # model training code omitted
    trained_model.save(model.path)

    Outputs = typing.NamedTuple('Outputs', model=Model, epochs=int)

    return Outputs(model=model, epochs=10)


Outputs = typing.NamedTuple('Outputs', model=Model, epochs=int)


@dsl.pipeline
def artifact_pipeline(dataset: Dataset) -> Outputs:
    # see definition of artifact_component above
    task = artifact_component(dataset=dataset)
    return Outputs(
        model=task.outputs['model'],
        epochs=task.outputs['epochs'],
    )


# multiple returns -- AFTER
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Model


@dsl.component
def artifact_component(
        dataset: Dataset) -> typing.NamedTuple(
            'Outputs', model=Model, epochs=int):
    with open(dataset.path) as f:
        data = f.read()

    # the mechanism for obtaining a system-generated path is discussed below
    uri = ...
    model = Model(name='my_model', uri=uri, metadata={'data': dataset.name})

    # model training code omitted
    trained_model.save(model.path)

    Outputs = typing.NamedTuple('Outputs', model=Model, epochs=int)

    return Outputs(model=model, epochs=10)


Outputs = typing.NamedTuple('Outputs', model=Model, epochs=int)


@dsl.pipeline
def artifact_pipeline(dataset: Dataset) -> Outputs:
    # see definition of artifact_component above
    task = artifact_component(dataset=dataset)
    return Outputs(
        model=task.outputs['model'],
        epochs=task.outputs['epochs'],
    )
