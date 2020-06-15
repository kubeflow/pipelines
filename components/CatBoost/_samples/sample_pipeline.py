import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/datasets/Chicago_Taxi_Trips/component.yaml')
catboost_train_classifier_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/Train_classifier/from_CSV/component.yaml')
catboost_train_regression_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/Train_regression/from_CSV/component.yaml')
catboost_predict_classes_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/Predict_classes/from_CSV/component.yaml')
catboost_predict_values_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/Predict_values/from_CSV/component.yaml')
catboost_predict_class_probabilities_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/Predict_class_probabilities/from_CSV/component.yaml')
catboost_to_apple_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/convert_CatBoostModel_to_AppleCoreMLModel/component.yaml')
catboost_to_onnx_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/470a5ba3/components/CatBoost/convert_CatBoostModel_to_ONNX/component.yaml')


def catboost_pipeline():
    get_training_data_task = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    )
    
    catboost_train_regression_task = catboost_train_regression_op(
        training_data=get_training_data_task.output,
        label_column=0,
        num_iterations=200,
    )
    
    catboost_predict_values_op(
        data=get_training_data_task.output,
        model=catboost_train_classifier_task.outputs['model'],
        label_column=0,
    )

    catboost_to_apple_op(catboost_train_regression_task.outputs['model'])
    catboost_to_onnx_op(catboost_train_regression_task.outputs['model'])

    catboost_train_classifier_task = catboost_train_classifier_op(
        training_data=get_training_data_task.output,
        label_column=0,
        num_iterations=200,
    )
    
    catboost_predict_classes_op(
        data=get_training_data_task.output,
        model=catboost_train_task.outputs['model'],
        label_column=0,
    )
    
    catboost_predict_class_probabilities_op(
        data=get_training_data_task.output,
        model=catboost_train_task.outputs['model'],
        label_column=0,
    )


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(catboost_pipeline, arguments={})
