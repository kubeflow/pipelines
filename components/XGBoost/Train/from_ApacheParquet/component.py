from kfp.components import InputPath, OutputPath, create_component_from_func

def xgboost_train(
    training_data_path: InputPath('ApacheParquet'),
    model_path: OutputPath('XGBoostModel'),
    model_config_path: OutputPath('XGBoostModelConfig'),
    label_column_name: str,

    starting_model_path: InputPath('XGBoostModel') = None,

    num_iterations: int = 10,
    booster_params: dict = None,

    # Booster parameters
    objective: str = 'reg:squarederror',
    booster: str = 'gbtree',
    learning_rate: float = 0.3,
    min_split_loss: float = 0,
    max_depth: int = 6,
):
    '''Train an XGBoost model.

    Args:
        training_data_path: Path for the training data in Apache Parquet format.
        model_path: Output path for the trained model in binary XGBoost format.
        model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.
        starting_model_path: Path for the existing trained model to start from.
        label_column_name: Name of the column containing the label data.
        num_boost_rounds: Number of boosting iterations.
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

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    import pandas
    import xgboost

    # Loading data
    df = pandas.read_parquet(training_data_path)
    training_data = xgboost.DMatrix(
        data=df.drop(columns=[label_column_name]),
        label=df[[label_column_name]],
    )
    # Training
    booster_params = booster_params or {}
    booster_params.setdefault('objective', objective)
    booster_params.setdefault('booster', booster)
    booster_params.setdefault('learning_rate', learning_rate)
    booster_params.setdefault('min_split_loss', min_split_loss)
    booster_params.setdefault('max_depth', max_depth)

    starting_model = None
    if starting_model_path:
        starting_model = xgboost.Booster(model_file=starting_model_path)

    model = xgboost.train(
        params=booster_params,
        dtrain=training_data,
        num_boost_round=num_iterations,
        xgb_model=starting_model
    )

    # Saving the model in binary format
    model.save_model(model_path)

    model_config_str = model.save_config()
    with open(model_config_path, 'w') as model_config_file:
        model_config_file.write(model_config_str)


if __name__ == '__main__':
    create_component_from_func(
        xgboost_train,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=[
            'xgboost==1.1.1',
            'pandas==1.0.5',
            'pyarrow==0.17.1',
        ]
    )
