from kfp.components import InputPath, OutputPath, create_component_from_func

def convert_apache_parquet_to_apache_arrow_feather(
    data_path: InputPath('ApacheParquet'),
    output_data_path: OutputPath('ApacheArrowFeather'),
):
    '''Converts Apache Parquet to Apache Arrow Feather.

    [Apache Arrow Feather](https://arrow.apache.org/docs/python/feather.html)
    [Apache Parquet](https://parquet.apache.org/)

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pyarrow import feather, parquet

    data_frame = parquet.read_pandas(data_path).to_pandas()
    feather.write_feather(data_frame, output_data_path)


if __name__ == '__main__':
    convert_apache_parquet_to_apache_arrow_feather_op = create_component_from_func(
        convert_apache_parquet_to_apache_arrow_feather,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['pyarrow==0.17.1', 'pandas==1.0.3']
    )
