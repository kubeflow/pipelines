from kfp.components import InputPath, OutputPath, create_component_from_func

def convert_apache_arrow_feather_to_apache_parquet(
    data_path: InputPath('ApacheArrowFeather'),
    output_data_path: OutputPath('ApacheParquet'),
):
    '''Converts Apache Arrow Feather to Apache Parquet.

    [Apache Arrow Feather](https://arrow.apache.org/docs/python/feather.html)
    [Apache Parquet](https://parquet.apache.org/)

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pyarrow import feather, parquet

    table = feather.read_table(data_path)
    parquet.write_table(table, output_data_path)


if __name__ == '__main__':
    create_component_from_func(
        convert_apache_arrow_feather_to_apache_parquet,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['pyarrow==0.17.1'],
        annotations={
            "author": "Alexey Volkov <alexey.volkov@ark-kun.com>",
            "canonical_location": "https://raw.githubusercontent.com/Ark-kun/pipeline_components/master/components/_converters/ApacheParquet/from_ApacheArrowFeather/component.yaml",
        },
    )
