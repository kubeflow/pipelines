from pathlib import Path

from kfp import compiler, dsl
from kfp.dsl import Dataset, Metrics, Output


@dsl.component(base_image='python:3.11')
def produce(
    seed: str, dataset: Output[Dataset], metrics: Output[Metrics]
) -> str:
    from pathlib import Path

    Path(dataset.path).write_text(seed, encoding='utf-8')
    metrics.log_metric('accuracy', 0.875)
    metrics.log_metric('loss', 0.125)
    return seed + '-produced'


@dsl.component(base_image='python:3.11')
def consume(value: str) -> str:
    return value + '-consumed'


@dsl.component(base_image='python:3.11')
def fail() -> None:
    raise ValueError('intentional fixture failure')


@dsl.pipeline(name='release-218-source-success')
def success_pipeline(seed: str = 'fixture'):
    producer = produce(seed=seed)
    consume(value=producer.outputs['Output'])
    with dsl.If(seed == 'run-optional'):
        consume(value='optional')


@dsl.pipeline(name='release-218-source-failure')
def failure_pipeline():
    fail()


@dsl.pipeline(name='release-218-source-inner')
def inner_pipeline(value: str):
    consume(value=value)


@dsl.pipeline(name='release-218-source-structure')
def structure_pipeline():
    exit_task = consume(value='exit-handler')
    with dsl.ExitHandler(exit_task=exit_task):
        with dsl.ParallelFor(items=['one', 'two']):
            inner_pipeline(value='nested')
        dsl.importer(
            artifact_uri='s3://release-218-fixture/input',
            artifact_class=Dataset,
        )


if __name__ == '__main__':
    output_dir = Path(__file__).resolve().parent
    for pipeline, filename in (
        (success_pipeline, 'success_pipeline.yaml'),
        (failure_pipeline, 'failure_pipeline.yaml'),
        (structure_pipeline, 'structure_pipeline.yaml'),
    ):
        compiler.Compiler().compile(pipeline, str(output_dir / filename))
