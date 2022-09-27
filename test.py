from typing import List

from kfp import dsl


@dsl.component
def train_model(model_type: str, epochs: List[int]) -> int:
    ...


@dsl.component
def select_best(score: List[int]) -> int:
    ...


# 1a: standard control flow
# N / A


# 1b: independent condition if-else
@dsl.pipeline
def my_pipeline(model_type: str):
    with dsl.Condition(model_type == 'tf'):
        tf_task = train_tf()

    with dsl.Condition(model_type == 'pt'):
        pt_task = train_pt()

    select_best(scores=(tf_task.output, pt_task.output))


# 2a: independent ParallelFor fan-in


@dsl.pipeline
def my_pipeline(epochs: List[int]):
    with dsl.ParallelFor(epochs) as e:
        task = train_model(epochs=e)

    select_best(scores=task.aggregated_parallel_for.output)


# 2b: simple


# 3b: top-level condition w inner ParallelFor
@dsl.pipeline
def my_pipeline(epochs: List[int], model_type: str):
    with dsl.Condition(model_type == 'tf'):
        with dsl.ParallelFor(epochs) as e:
            tf_task = train_tf(epochs=e)

    with dsl.Condition(model_type == 'pt'):
        with dsl.ParallelFor(epochs) as e:
            pt_task = train_pt(epochs=e)

    select_best(
        scores=(tf_task.aggregated_parallel_for.output,
                pt_task.aggregated_parallel_for.output))


@dsl.pipeline
def my_pipeline(epochs: List[int], model_type: str):
    with dsl.Condition(model_type == 'tf'):
        with dsl.ParallelFor(epochs) as e:
            tf_task = train_tf(epochs=e)

        tf_best = select_best(scores=tf_task.aggregated_parallel_for.output)

    with dsl.Condition(model_type == 'pt'):
        with dsl.ParallelFor(epochs) as e:
            pt_task = train_pt(epochs=e)

        pt_best = select_best(scores=pt_task.aggregated_parallel_for.output)

    another_component(scores=(tf_best.output, pt_best.output))


# 3c: top-level ParallelFor with condition
@dsl.pipeline
def my_pipeline(epochs: List[int], model_type: str):
    with dsl.ParallelFor(epochs) as e:
        with dsl.Condition(model_type == 'tf'):
            tf_task = train_tf(epochs=e)

        with dsl.Condition(model_type == 'pt'):
            pt_task = train_pt(epochs=e)

    select_best(
        scores=(tf_task.aggregated_parallel_for.output,
                pt_task.aggregated_parallel_for.output))
