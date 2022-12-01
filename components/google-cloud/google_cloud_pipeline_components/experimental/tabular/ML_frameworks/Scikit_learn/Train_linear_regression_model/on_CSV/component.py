from kfp.components import InputPath, OutputPath, create_component_from_func


def train_linear_regression_model_using_scikit_learn_from_CSV(
    dataset_path: InputPath("CSV"),
    model_path: OutputPath("ScikitLearnPickleModel"),
    label_column_name: str,
):
    import pandas
    import pickle
    from sklearn import linear_model

    df = pandas.read_csv(dataset_path).convert_dtypes()
    model = linear_model.LinearRegression()
    model.fit(
        X=df.drop(columns=label_column_name),
        y=df[label_column_name],
    )

    with open(model_path, "wb") as f:
        pickle.dump(model, f)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    train_linear_regression_model_using_scikit_learn_from_CSV_op = create_component_from_func(
        func=train_linear_regression_model_using_scikit_learn_from_CSV,
        base_image="python:3.9",
        packages_to_install=[
            "scikit-learn==1.0.2",
            "pandas==1.4.3",
        ],
        output_component_file="component.yaml",
    )
