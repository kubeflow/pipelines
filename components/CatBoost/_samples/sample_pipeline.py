import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/5e3d9aa791fe684962d16be4f0f767feed6222b2/components/datasets/Chicago_Taxi_Trips/component.yaml')
catboost_train_classifier_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/beea69058b71cc44fb491c5728a6efa99a4a5bee/components/CatBoost/Train_classifier/component.yaml')
catboost_predict_class_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/beea69058b71cc44fb491c5728a6efa99a4a5bee/components/CatBoost/Predict_class/component.yaml')


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
