from kfp.components import InputPath, OutputPath, create_component_from_func

def Pandas_Transform_DataFrame_in_ApacheParquet_format(
    dataframe_path: InputPath('ApacheParquet'),
    output_dataframe_path: OutputPath('ApacheParquet'),
    transform_code: 'PythonCode',
):
    '''Transform DataFrame loaded from an ApacheParquet file.

    Inputs:
        dataframe: DataFrame to transform.
        transform_code: Transformation code. Code is written in Python and can consist of multiple lines.
            The DataFrame variable is called "df".
            Example: `df['prod'] = df['X'] * df['Y']` or `df = df[['X', 'prod']]`

    Outputs:
        output_dataframe: Transformed DataFrame.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    import pandas  # This import can be used in the transformation code
    import pyarrow
    from pyarrow import parquet

    df = parquet.read_pandas(dataframe_path).to_pandas()
    exec(transform_code)
    table = pyarrow.Table.from_pandas(df)
    parquet.write_table(table, output_dataframe_path)


if __name__ == '__main__':
    Pandas_Transform_DataFrame_in_ApacheParquet_format_op = create_component_from_func(
        Pandas_Transform_DataFrame_in_ApacheParquet_format,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=[
            'pyarrow==0.14.1',
            'pandas==1.0.4',
        ],
    )
