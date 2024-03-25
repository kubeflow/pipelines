from kfp.components import InputPath, OutputPath, create_component_from_func


def select_columns_using_Pandas_on_CSV_data(
    table_path: InputPath("CSV"),
    transformed_table_path: OutputPath("CSV"),
    column_names: list,
):
    import pandas

    df = pandas.read_csv(
        table_path,
        dtype="string",
    )
    df = df[column_names]
    df.to_csv(transformed_table_path, index=False)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    select_columns_using_Pandas_on_CSV_data_op = create_component_from_func(
        select_columns_using_Pandas_on_CSV_data,
        output_component_file="component.yaml",
        base_image="python:3.9",
        packages_to_install=["pandas==1.4.2",],
    )
