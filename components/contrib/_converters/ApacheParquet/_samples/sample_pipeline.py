import kfp
from kfp import components

component_store = components.ComponentStore(url_search_prefixes=['https://raw.githubusercontent.com/kubeflow/pipelines/af3eaf64e87313795cad1add9bfd9fa1e86af6de/components/'])

chicago_taxi_dataset_op = component_store.load_component(name='datasets/Chicago_Taxi_Trips')
convert_csv_to_apache_parquet_op = component_store.load_component(name='_converters/ApacheParquet/from_CSV')
convert_tsv_to_apache_parquet_op = component_store.load_component(name='_converters/ApacheParquet/from_TSV')
convert_apache_parquet_to_csv_op = component_store.load_component(name='_converters/ApacheParquet/to_CSV')
convert_apache_parquet_to_tsv_op = component_store.load_component(name='_converters/ApacheParquet/to_TSV')
convert_apache_parquet_to_apache_arrow_feather_op = component_store.load_component(name='_converters/ApacheParquet/to_ApacheArrowFeather')
convert_apache_arrow_feather_to_apache_parquet_op = component_store.load_component(name='_converters/ApacheParquet/from_ApacheArrowFeather')


def parquet_pipeline():
    csv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    ).output

    tsv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
        format='tsv',
    ).output
    
    csv_parquet = convert_csv_to_apache_parquet_op(csv).output
    csv_parquet_csv = convert_apache_parquet_to_csv_op(csv_parquet).output
    csv_parquet_feather = convert_apache_parquet_to_apache_arrow_feather_op(csv_parquet).output
    csv_parquet_feather_parquet = convert_apache_arrow_feather_to_apache_parquet_op(csv_parquet_feather).output
    
    tsv_parquet = convert_tsv_to_apache_parquet_op(tsv).output
    tsv_parquet_tsv = convert_apache_parquet_to_tsv_op(tsv_parquet).output
    tsv_parquet_feather = convert_apache_parquet_to_apache_arrow_feather_op(tsv_parquet).output
    tsv_parquet_feather_parquet = convert_apache_arrow_feather_to_apache_parquet_op(tsv_parquet_feather).output

if __name__ == '__main__':
    kfp_endpoint = None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(parquet_pipeline, arguments={})
