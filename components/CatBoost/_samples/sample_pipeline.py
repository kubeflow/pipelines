import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')

catboost_train_classifier_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/Train_classifier/from_CSV/component.yaml')
catboost_train_regression_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/Train_regression/from_CSV/component.yaml')
catboost_predict_classes_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/Predict_classes/from_CSV/component.yaml')
catboost_predict_values_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/Predict_values/from_CSV/component.yaml')
catboost_predict_class_probabilities_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/Predict_class_probabilities/from_CSV/component.yaml')
catboost_to_apple_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/convert_CatBoostModel_to_AppleCoreMLModel/component.yaml')
catboost_to_onnx_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f97ad2/components/CatBoost/convert_CatBoostModel_to_ONNX/component.yaml')

convert_csv_to_apache_parquet_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/d737c448723b9f541a3543012b4414c17b2eab5c/components/_converters/ApacheParquet/from_CSV/component.yaml')
pandas_transform_parquet_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e69a6694/components/pandas/Transform_DataFrame/in_ApacheParquet_format/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e69a6694/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')

def catboost_pipeline():
    training_data_in_csv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    ).output

    evaluation_data_in_csv = training_data_in_csv

    catboost_train_regression_task = catboost_train_regression_op(
        training_data=training_data_in_csv,
        loss_function='RMSE',
        label_column=0,
        num_iterations=200,
    )

    catboost_predict_values_op(
        data=evaluation_data_in_csv,
        model=catboost_train_regression_task.outputs['model'],
        label_column=0,
    )

    catboost_to_apple_op(catboost_train_regression_task.outputs['model'])

    catboost_to_onnx_op(catboost_train_regression_task.outputs['model'])

    training_data_in_parquet = convert_csv_to_apache_parquet_op(training_data_in_csv).output

    training_data_for_classification_in_parquet = pandas_transform_parquet_op(
        table=training_data_in_parquet,
        #transform_code='''df["was_tipped"] = df["tips"] > 0''',
        transform_code='''df.insert(0, "was_tipped", df["tips"] > 0); del df["tips"]''',
    ).output

    training_data_for_classification_in_csv = pandas_transform_csv_op(
        table=training_data_in_csv,
        #transform_code='''df["was_tipped"] = df["tips"] > 0''',
        transform_code='''df.insert(0, "was_tipped", df["tips"] > 0); del df["tips"]''',
    ).output
    
    evaluation_data_for_classification_in_csv = training_data_for_classification_in_csv

    catboost_train_classifier_task = catboost_train_classifier_op(
        training_data=evaluation_data_for_classification_in_csv,
        label_column=0,
        num_iterations=200,
    )

    catboost_predict_classes_op(
        data=evaluation_data_in_csv,
        model=catboost_train_classifier_task.outputs['model'],
        label_column=0,
    )

    catboost_predict_class_probabilities_op(
        data=evaluation_data_for_classification_in_csv,
        model=catboost_train_classifier_task.outputs['model'],
        label_column=0,
    )


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(catboost_pipeline, arguments={})
