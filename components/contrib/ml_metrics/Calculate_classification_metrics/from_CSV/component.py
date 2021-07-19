from typing import NamedTuple

from kfp.components import InputPath, create_component_from_func


def calculate_classification_metrics_from_csv(
        true_values_path: InputPath(),
        predicted_values_path: InputPath(),
        sample_weights_path: InputPath() = None,
        average: str = 'binary'
) -> NamedTuple('Outputs', [
    ('f1', float),
    ('precision', float),
    ('recall', float),
    ('accuracy', float),
]):
    """
    Calculates classification metrics.

    Annotations:
        author: Anton Kiselev <akiselev@provectus.com>
    """
    import numpy
    from sklearn.metrics import f1_score, precision_score, recall_score, accuracy_score

    true_values = numpy.loadtxt(true_values_path, dtype=str)
    predicted_values = numpy.loadtxt(predicted_values_path, dtype=str)

    if len(predicted_values.shape) != 1:
        raise NotImplemented('Only single prediction values are supported.')
    if len(true_values.shape) != 1:
        raise NotImplemented('Only single true values are supported.')

    if predicted_values.shape != true_values.shape:
        raise ValueError(f'Input shapes are different: {predicted_values.shape} != {true_values.shape}')

    sample_weights = None
    if sample_weights_path is not None:
        sample_weights = numpy.loadtxt(sample_weights_path, dtype=float)

        if len(sample_weights.shape) != 1:
            raise NotImplemented('Only single sample weights are supported.')

        if sample_weights.shape != predicted_values.shape:
            raise ValueError(f'Input shapes of sample weights and predictions are different: '
                             f'{sample_weights.shape} != {predicted_values.shape}')

    f1 = f1_score(true_values, predicted_values, average=average, sample_weight=sample_weights)
    precision = precision_score(true_values, predicted_values, average=average, sample_weight=sample_weights)
    recall = recall_score(true_values, predicted_values, average=average, sample_weight=sample_weights)
    accuracy = accuracy_score(true_values, predicted_values, normalize=average, sample_weight=sample_weights)

    metrics = dict(
        f1=f1,
        precision=precision,
        recall=recall,
        accuracy=accuracy
    )

    return (
        f1,
        precision,
        recall,
        accuracy,
        metrics,
    )


if __name__ == '__main__':
    calculate_regression_metrics_from_csv_op = create_component_from_func(
        calculate_classification_metrics_from_csv,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['numpy==1.19.0', 'scikit-learn==0.23.2']
    )
