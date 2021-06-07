from collections import OrderedDict
from kfp import components


split_table_into_folds_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e9b4b29b22a5120daf95b581b0392cd461a906f0/components/dataset_manipulation/split_data_into_folds/in_CSV/component.yaml')
xgboost_train_on_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/567c04c51ff00a1ee525b3458425b17adbe3df61/components/XGBoost/Train/component.yaml')
xgboost_predict_on_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/567c04c51ff00a1ee525b3458425b17adbe3df61/components/XGBoost/Predict/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/6162d55998b176b50267d351241100bb0ee715bc/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')
drop_header_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/02c9638287468c849632cf9f7885b51de4c66f86/components/tables/Remove_header/component.yaml')
calculate_regression_metrics_from_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/7da1ac9464b4b3e7d95919faa2f1107a9635b7e4/components/ml_metrics/Calculate_regression_metrics/from_CSV/component.yaml')
aggregate_regression_metrics_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/7ea9363fe201918d419fecdc00d1275e657ff712/components/ml_metrics/Aggregate_regression_metrics/component.yaml')


def xgboost_5_fold_cross_validation_for_regression(
    data: 'CSV',
    label_column: int = 0,
    objective: str = 'reg:squarederror',
    num_iterations: int = 200,
):
    folds = split_table_into_folds_op(data).outputs

    fold_metrics = {}
    for i in range(1, 6):
        training_data = folds['train_' + str(i)]
        testing_data = folds['test_' + str(i)]
        model = xgboost_train_on_csv_op(
            training_data=training_data,
            label_column=label_column,
            objective=objective,
            num_iterations=num_iterations,
        ).outputs['model']

        predictions = xgboost_predict_on_csv_op(
            data=testing_data,
            model=model,
            label_column=label_column,
        ).output

        true_values_table = pandas_transform_csv_op(
            table=testing_data,
            transform_code='df = df[["tips"]]',
        ).output

        true_values = drop_header_op(true_values_table).output

        metrics = calculate_regression_metrics_from_csv_op(
            true_values=true_values,
            predicted_values=predictions,
        ).outputs['metrics']

        fold_metrics['metrics_' + str(i)] = metrics

    aggregated_metrics_task = aggregate_regression_metrics_op(**fold_metrics)

    return OrderedDict([
        ('mean_absolute_error', aggregated_metrics_task.outputs['mean_absolute_error']),
        ('mean_squared_error', aggregated_metrics_task.outputs['mean_squared_error']),
        ('root_mean_squared_error', aggregated_metrics_task.outputs['root_mean_squared_error']),
        ('metrics', aggregated_metrics_task.outputs['metrics']),
    ])


if __name__ == '__main__':
    xgboost_5_fold_cross_validation_for_regression_op = components.create_graph_component_from_pipeline_func(
        xgboost_5_fold_cross_validation_for_regression,
        output_component_file='component.yaml',
    )
