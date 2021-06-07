from kfp.components import InputPath, OutputPath, create_component_from_func

def Pandas_Transform_DataFrame_in_ApacheParquet_format(
    table_path: InputPath('ApacheParquet'),
    transformed_table_path: OutputPath('ApacheParquet'),
    transform_code: 'PythonCode',
):
    '''Transform DataFrame loaded from an ApacheParquet file.

    Inputs:
        table: DataFrame to transform.
        transform_code: Transformation code. Code is written in Python and can consist of multiple lines.
            The DataFrame variable is called "df".
            Examples:
            - `df['prod'] = df['X'] * df['Y']`
            - `df = df[['X', 'prod']]`
            - `df.insert(0, "is_positive", df["X"] > 0)`

    Outputs:
        transformed_table: Transformed DataFrame.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    import pandas

    df = pandas.read_parquet(table_path)
    # The namespace is needed so that the code can replace `df`. For example df = df[['X']]
    namespace = locals()
    exec(transform_code, namespace)
    namespace['df'].to_parquet(transformed_table_path)


if __name__ == '__main__':
    Pandas_Transform_DataFrame_in_ApacheParquet_format_op = create_component_from_func(
        Pandas_Transform_DataFrame_in_ApacheParquet_format,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=[
            'pandas==1.0.4',
            'pyarrow==0.14.1',
        ],
    )
