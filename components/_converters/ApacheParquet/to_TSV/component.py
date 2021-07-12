from kfp.components import InputPath, OutputPath, create_component_from_func

def convert_apache_parquet_to_tsv(
    data_path: InputPath('ApacheParquet'),
    output_data_path: OutputPath('TSV'),
):
    '''Converts Apache Parquet to TSV.

    [Apache Parquet](https://parquet.apache.org/)

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pyarrow import parquet

    data_frame = parquet.read_pandas(data_path).to_pandas()
    data_frame.to_csv(
        output_data_path,
        index=False,
        sep='\t',
    )


if __name__ == '__main__':
    convert_apache_parquet_to_tsv_op = create_component_from_func(
        convert_apache_parquet_to_tsv,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['pyarrow==0.17.1', 'pandas==1.0.3']
    )
