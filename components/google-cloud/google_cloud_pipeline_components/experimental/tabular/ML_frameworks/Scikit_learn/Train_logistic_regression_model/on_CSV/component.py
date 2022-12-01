from typing import NamedTuple
from kfp.components import InputPath, OutputPath, create_component_from_func


def train_logistic_regression_model_using_scikit_learn_from_CSV(
    dataset_path: InputPath("CSV"),
    model_path: OutputPath("ScikitLearnPickleModel"),
    label_column_name: str,
    penalty: str = "l2", # l1, l2, elasticnet, none
    solver: str = "lbfgs", # newton-cg, lbfgs, liblinear, sag, saga
    max_iterations: int = 100,
    multi_class_mode: str = "auto", # auto, ovr, multinomial
    random_seed: int = 0,
) -> NamedTuple("Outputs", [
    ("model_parameters", dict),
]):
    """Train logistic regression model using Scikit-learn
    See https://scikit-learn.org/stable/modules/generated/sklearn.linear_model.LogisticRegression.html
    """
    import json
    import pandas
    import pickle
    from sklearn import linear_model

    df = pandas.read_csv(dataset_path).convert_dtypes()
    model = linear_model.LogisticRegression(
        penalty=penalty,
        #dual=False,
        #tol=1e-4,
        #C=1.0,
        #fit_intercept=True,
        #intercept_scaling=1,
        #class_weight=None,
        random_state=random_seed,
        solver=solver,
        max_iter=max_iterations,
        multi_class=multi_class_mode,
        #l1_ratio=None,
        verbose=1,
    )

    model_parameters = model.get_params()
    model_parameters_json = json.dumps(model_parameters, indent=2)
    print("Model parameters:")
    print(model_parameters_json)
    print()

    model.fit(
        X=df.drop(columns=label_column_name),
        y=df[label_column_name],
    )

    with open(model_path, "wb") as f:
        pickle.dump(model, f)

    return (model_parameters_json,)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    train_logistic_regression_model_using_scikit_learn_from_CSV_op = create_component_from_func(
        func=train_logistic_regression_model_using_scikit_learn_from_CSV,
        base_image="python:3.9",
        packages_to_install=[
            "scikit-learn==1.0.2",
            "pandas==1.4.3",
        ],
        output_component_file="component.yaml",
    )
