from kfp.components import InputPath, OutputPath, create_component_from_func


def train_XGBoost_model_on_CSV(
    training_data_path: InputPath("CSV"),
    model_path: OutputPath("XGBoostModel"),
    model_config_path: OutputPath("XGBoostModelConfig"),
    label_column_name: str,
    starting_model_path: InputPath("XGBoostModel") = None,
    num_iterations: int = 10,
    # Booster parameters
    objective: str = "reg:squarederror",
    booster: str = "gbtree",
    learning_rate: float = 0.3,
    min_split_loss: float = 0,
    max_depth: int = 6,
    booster_params: dict = None,
):
    """Trains an XGBoost model.
    Args:
        training_data_path: Training data in CSV format.
        model_path: Trained model in the binary XGBoost format.
        model_config_path: The internal parameter configuration of Booster as a JSON string.
        starting_model_path: Existing trained model to start from (in the binary XGBoost format).
        label_column_name: Name of the column containing the label data.
        num_iterations: Number of boosting iterations.
        booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html
        objective: The learning task and the corresponding learning objective.
            See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters
            The most common values are:
            "reg:squarederror" - Regression with squared loss (default).
            "reg:logistic" - Logistic regression.
            "binary:logistic" - Logistic regression for binary classification, output probability.
            "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation
            "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized
            "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized
        booster: The booster to use. Can be `gbtree`, `gblinear` or `dart`; `gbtree` and `dart` use tree based models while `gblinear` uses linear functions.
        learning_rate: Step size shrinkage used in update to prevents overfitting. Range: [0,1].
        min_split_loss: Minimum loss reduction required to make a further partition on a leaf node of the tree.
            The larger `min_split_loss` is, the more conservative the algorithm will be. Range: [0,Inf].
        max_depth: Maximum depth of a tree. Increasing this value will make the model more complex and more likely to overfit.
            0 indicates no limit on depth. Range: [0,Inf].
    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    """
    import pandas
    import xgboost

    df = pandas.read_csv(
        training_data_path,
    ).convert_dtypes()
    print("Training data information:")
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
    print()
    print("Final training data information:")
    df.info(verbose=True)

    training_data = xgboost.DMatrix(
        data=df.drop(columns=[label_column_name]),
        label=df[[label_column_name]],
        enable_categorical=True,
    )

    booster_params = booster_params or {}
    booster_params.setdefault("objective", objective)
    booster_params.setdefault("booster", booster)
    booster_params.setdefault("learning_rate", learning_rate)
    booster_params.setdefault("min_split_loss", min_split_loss)
    booster_params.setdefault("max_depth", max_depth)

    starting_model = None
    if starting_model_path:
        starting_model = xgboost.Booster(model_file=starting_model_path)

    print()
    print("Training the model:")
    model = xgboost.train(
        params=booster_params,
        dtrain=training_data,
        num_boost_round=num_iterations,
        xgb_model=starting_model,
        evals=[(training_data, "training_data")],
    )

    # Saving the model in binary format
    model.save_model(model_path)

    model_config_str = model.save_config()
    with open(model_config_path, "w") as model_config_file:
        model_config_file.write(model_config_str)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    train_XGBoost_model_on_CSV_op = create_component_from_func(
        train_XGBoost_model_on_CSV,
        output_component_file="component.yaml",
        base_image="python:3.10",
        packages_to_install=[
            "xgboost==1.6.1",
            "pandas==1.4.3",
        ],
    )
