from typing import NamedTuple
from kfp.components import create_component_from_func


def aggregate_regression_metrics(
    metrics_1: dict,
    metrics_2: dict = None,
    metrics_3: dict = None,
    metrics_4: dict = None,
    metrics_5: dict = None,
) -> NamedTuple('Outputs', [
    ('number_of_items', int),
    ('max_absolute_error', float),
    ('mean_absolute_error', float),
    ('mean_squared_error', float),
    ('root_mean_squared_error', float),
    ('metrics', dict),
]):
    '''Calculates regression metrics.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    import math

    metrics_dicts = [d for d in [metrics_1, metrics_2, metrics_3, metrics_4, metrics_5] if d is not None]
    number_of_items = sum(metrics['number_of_items'] for metrics in metrics_dicts)
    max_absolute_error = max(metrics['max_absolute_error'] for metrics in metrics_dicts)
    mean_absolute_error = sum(metrics['mean_absolute_error'] * metrics['number_of_items'] for metrics in metrics_dicts) / number_of_items
    mean_squared_error = sum(metrics['mean_squared_error'] * metrics['number_of_items'] for metrics in metrics_dicts) / number_of_items
    root_mean_squared_error = math.sqrt(mean_squared_error)
    metrics = dict(
        number_of_items=number_of_items,
        max_absolute_error=max_absolute_error,
        mean_absolute_error=mean_absolute_error,
        mean_squared_error=mean_squared_error,
        root_mean_squared_error=root_mean_squared_error,
    )

    return (
        number_of_items,
        max_absolute_error,
        mean_absolute_error,
        mean_squared_error,
        root_mean_squared_error,
        metrics,
    )


if __name__ == '__main__':
    aggregate_regression_metrics_op = create_component_from_func(
        aggregate_regression_metrics,
        output_component_file='component.yaml',
        base_image='python:3.7',
    )
