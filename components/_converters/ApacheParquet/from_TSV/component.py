from kfp.components import InputPath, OutputPath, create_component_from_func

def convert_tsv_to_apache_parquet(
    data_path: InputPath('TSV'),
    output_data_path: OutputPath('ApacheParquet'),
):
    '''Converts TSV table to Apache Parquet.

    [Apache Parquet](https://parquet.apache.org/)

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pyarrow import csv, parquet

    table = csv.read_csv(data_path, parse_options=csv.ParseOptions(delimiter='\t'))
    parquet.write_table(table, output_data_path)


if __name__ == '__main__':
    create_component_from_func(
        convert_tsv_to_apache_parquet,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['pyarrow==0.17.1']
    )
