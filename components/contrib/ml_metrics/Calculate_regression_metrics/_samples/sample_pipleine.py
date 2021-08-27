import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')
xgboost_train_on_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/567c04c51ff00a1ee525b3458425b17adbe3df61/components/XGBoost/Train/component.yaml')
xgboost_predict_on_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/567c04c51ff00a1ee525b3458425b17adbe3df61/components/XGBoost/Predict/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/6162d55998b176b50267d351241100bb0ee715bc/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')
drop_header_op = kfp.components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/02c9638287468c849632cf9f7885b51de4c66f86/components/tables/Remove_header/component.yaml')
calculate_regression_metrics_from_csv_op = kfp.components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/616542ac0f789914f4eb53438da713dd3004fba4/components/ml_metrics/Calculate_regression_metrics/from_CSV/component.yaml')


def regression_metrics_pipeline():
    training_data_csv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    ).output

    # Training
    model_trained_on_csv = xgboost_train_on_csv_op(
        training_data=training_data_csv,
        label_column=0,
        objective='reg:squarederror',
        num_iterations=200,
    ).outputs['model']

    # Predicting
    predictions = xgboost_predict_on_csv_op(
        data=training_data_csv,
        model=model_trained_on_csv,
        label_column=0,
    ).output

    # Preparing the true values
    true_values_table = pandas_transform_csv_op(
        table=training_data_csv,
        transform_code='''df = df[["tips"]]''',
    ).output
    
    true_values = drop_header_op(true_values_table).output

    # Calculating the regression metrics    
    calculate_regression_metrics_from_csv_op(
        true_values=true_values,
        predicted_values=predictions,
    )


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(regression_metrics_pipeline, arguments={})
