from kfp.components import InputPath, OutputPath, create_component_from_func


def fill_all_missing_values_using_Pandas_on_CSV_data(
    table_path: InputPath("CSV"),
    transformed_table_path: OutputPath("CSV"),
    replacement_value: str = "0",
    column_names: list = None,
):
    import pandas

    df = pandas.read_csv(
        table_path,
        dtype="string",
    )

    for column_name in column_names or df.columns:
        df[column_name] = df[column_name].fillna(value=replacement_value)

    df.to_csv(
        transformed_table_path, index=False,
    )


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    fill_all_missing_values_using_Pandas_on_CSV_data_op = create_component_from_func(
        fill_all_missing_values_using_Pandas_on_CSV_data,
        output_component_file="component.yaml",
        base_image="python:3.9",
        packages_to_install=["pandas==1.4.1",],
    )
