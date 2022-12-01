from kfp.components import InputPath, OutputPath, create_component_from_func


def xgboost_predict_on_CSV(
    data_path: InputPath("CSV"),
    model_path: InputPath("XGBoostModel"),
    predictions_path: OutputPath("Text"),
    label_column_name: str = None,
):
    """Makes predictions using a trained XGBoost model.
    Args:
        data_path: Feature data in Apache Parquet format.
        model_path: Trained model in binary XGBoost format.
        predictions_path: Model predictions.
        label_column_name: Optional. Name of the column containing the label data that is excluded during the prediction.
    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    from pathlib import Path

    import numpy
    import pandas
    import xgboost

    df = pandas.read_csv(
        data_path,
    ).convert_dtypes()
    print("Evaluation data information:")
    df.info(verbose=True)
    # Converting column types that XGBoost does not support
    for column_name, dtype in df.dtypes.items():
        if dtype in ["string", "object"]:
            print(f"Treating the {dtype.name} column '{column_name}' as categorical.")
            df[column_name] = df[column_name].astype("category")
            print(f"Inferred {len(df[column_name].cat.categories)} categories for the '{column_name}' column.")
        # Working around the XGBoost issue with nullable floats: https://github.com/dmlc/xgboost/issues/8213
        if pandas.api.types.is_float_dtype(dtype):
            # Converting from "Float64" to "float64"
            df[column_name] = df[column_name].astype(dtype.name.lower())
    print("Final evaluation data information:")
    df.info(verbose=True)

    if label_column_name is not None:
        df = df.drop(columns=[label_column_name])

    testing_data = xgboost.DMatrix(
        data=df,
        enable_categorical=True,
    )

    model = xgboost.Booster(model_file=model_path)

    predictions = model.predict(testing_data)

    Path(predictions_path).parent.mkdir(parents=True, exist_ok=True)
    numpy.savetxt(predictions_path, predictions)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    xgboost_predict_on_CSV_op = create_component_from_func(
        xgboost_predict_on_CSV,
        output_component_file="component.yaml",
        base_image="python:3.10",
        packages_to_install=[
            "xgboost==1.6.1",
            "pandas==1.4.3",
        ],
    )