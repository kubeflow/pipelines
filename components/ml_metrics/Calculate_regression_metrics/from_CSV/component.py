from typing import NamedTuple
from kfp.components import InputPath, OutputPath, create_component_from_func

def calculate_regression_metrics_from_csv(
    true_values_path: InputPath(),
    predicted_values_path: InputPath(),
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
    import numpy

    true_values = numpy.loadtxt(true_values_path, dtype=numpy.float64)
    predicted_values = numpy.loadtxt(predicted_values_path, dtype=numpy.float64)

    if len(predicted_values.shape) != 1:
        raise NotImplemented('Only single prediction values are supported.')
    if len(true_values.shape) != 1:
        raise NotImplemented('Only single true values are supported.')

    if predicted_values.shape != true_values.shape:
        raise ValueError('Input shapes are different: {} != {}'.format(predicted_values.shape, true_values.shape))

    number_of_items = true_values.size
    errors = (true_values - predicted_values)
    abs_errors = numpy.abs(errors)
    squared_errors = errors ** 2
    max_absolute_error = numpy.max(abs_errors)
    mean_absolute_error = numpy.average(abs_errors)
    mean_squared_error = numpy.average(squared_errors)
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
    calculate_regression_metrics_from_csv_op = create_component_from_func(
        calculate_regression_metrics_from_csv,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['numpy==1.19.0']
    )
