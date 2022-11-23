from kfp import dsl


@dsl.component
def return_str() -> str:
    return ''


@dsl.component
def consumer(string: str):
    pass


@dsl.pipeline
def my_pipeline(x: int):
    with dsl.Condition(x == 1):
        t1 = return_str()
    with dsl.Condition(x == 2):
        t2 = return_str()

    consumer(string=(t1.output, t2.output))


from typing import List

from kfp.dsl import Model


@dsl.pipeline
def train_models(epochs: List[int]) -> List[Model]:
    with dsl.ParallelFor(epochs) as e:
        task = train_model(epochs=e)
    return dsl.FanIn(task.output['model'])


@dsl.pipeline
def my_pipeline(epochs: List[int]):
    with dsl.ParallelFor(epochs) as e:
        train_task = train_model(epochs=e)

    select_best(scores=dsl.FanIn(train_task.output))
