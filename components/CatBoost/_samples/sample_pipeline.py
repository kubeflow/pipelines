import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')
catboost_train_classifier_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/0394a738bcc760ca4b8016c91aa8a0548ba76894/components/CatBoost/Train_classifier/component.yaml')
catboost_predict_class_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/0394a738bcc760ca4b8016c91aa8a0548ba76894/components/CatBoost/Train_classifier/component.yaml')


def catboost_pipeline():
    get_training_data_task = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    )
    
    catboost_train_task = catboost_train_classifier_op(
        training_data=get_training_data_task.output,
        label_column=0,
        num_iterations=200,
    )
    
    catboost_predict_class_op(
        data=get_training_data_task.output,
        model=catboost_train_task.outputs['model'],
        label_column=0,
    )


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(catboost_pipeline, arguments={})
