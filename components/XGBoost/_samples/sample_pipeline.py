import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')
xgboost_train_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/eb0e470be494c2f6415d543d7f37c61f1bf768dc/components/XGBoost/Train/component.yaml')
xgboost_predict_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/eb0e470be494c2f6415d543d7f37c61f1bf768dc/components/XGBoost/Predict/component.yaml')


def xgboost_pipeline():
    get_training_data_task = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    )
    
    xgboost_train_task = xgboost_train_op(
        training_data=get_training_data_task.output,
        label_column=0,
        objective='reg:squarederror',
        num_iterations=200,
    )
    
    xgboost_predict_op(
        data=get_training_data_task.output,
        model=xgboost_train_task.outputs['model'],
        label_column=0,
    )


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(xgboost_pipeline, arguments={})
