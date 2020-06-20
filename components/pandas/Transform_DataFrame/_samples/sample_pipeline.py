import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')
convert_csv_to_apache_parquet_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/d737c448723b9f541a3543012b4414c17b2eab5c/components/_converters/ApacheParquet/from_CSV/component.yaml')

pandas_transform_parquet_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e69a6694/components/pandas/Transform_DataFrame/in_ApacheParquet_format/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e69a6694/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')


def pandas_transform_pipeline():
    training_data_in_csv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=1000,
    ).output

    training_data_in_parquet = convert_csv_to_apache_parquet_op(training_data_in_csv).output

    training_data_for_classification_in_parquet = pandas_transform_parquet_op(
        table=training_data_in_parquet,
        transform_code='''df.insert(0, "was_tipped", df["tips"] > 0); del df["tips"]''',
    ).output

    training_data_for_classification_in_csv = pandas_transform_csv_op(
        table=training_data_in_csv,
        transform_code='''df.insert(0, "was_tipped", df["tips"] > 0); del df["tips"]''',
    ).output


if __name__ == '__main__':
    kfp_endpoint=None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(pandas_transform_pipeline, arguments={})
