from kfp.components import InputPath, OutputPath, create_component_from_func

def catboost_train_regression(
    training_data_path: InputPath('CSV'),
    model_path: OutputPath('CatBoostModel'),
    starting_model_path: InputPath('CatBoostModel') = None,
    label_column: int = 0,

    loss_function: str = 'RMSE',
    num_iterations: int = 500,
    learning_rate: float = None,
    depth: int = 6,
    random_seed: int = 0,

    cat_features: list = None,

    additional_training_options: dict = {},
):
    '''Train a CatBoost classifier model.

    Args:
        training_data_path: Path for the training data in CSV format.
        model_path: Output path for the trained model in binary CatBoostModel format.
        starting_model_path: Path for the existing trained model to start from.
        label_column: Column containing the label data.

        loss_function: The metric to use in training and also selector of the machine learning
            problem to solve. Default = 'RMSE'. Possible values:
            'RMSE', 'MAE', 'Quantile:alpha=value', 'LogLinQuantile:alpha=value', 'Poisson', 'MAPE', 'Lq:q=value'
        num_iterations: Number of trees to add to the ensemble.
        learning_rate: Step size shrinkage used in update to prevents overfitting.
            Default value is selected automatically for binary classification with other parameters set to default.
            In all other cases default is 0.03.
        depth: Depth of a tree. All trees are the same depth. Default = 6
        random_seed: Random number seed. Default = 0

        cat_features: A list of Categorical features (indices or names).
        additional_training_options: A dictionary with additional options to pass to CatBoostRegressor

    Outputs:
        model: Trained model in binary CatBoostModel format.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    import tempfile
    from pathlib import Path

    from catboost import CatBoostRegressor, Pool

    column_descriptions = {label_column: 'Label'}
    column_description_path = tempfile.NamedTemporaryFile(delete=False).name
    with open(column_description_path, 'w') as column_description_file:
        for idx, kind in column_descriptions.items():
            column_description_file.write('{}\t{}\n'.format(idx, kind))

    train_data = Pool(
        training_data_path,
        column_description=column_description_path,
        has_header=True,
        delimiter=',',
    )

    model = CatBoostRegressor(
        iterations=num_iterations,
        depth=depth,
        learning_rate=learning_rate,
        loss_function=loss_function,
        random_seed=random_seed,
        verbose=True,
        **additional_training_options,
    )

    model.fit(
        train_data,
        cat_features=cat_features,
        init_model=starting_model_path,
        #verbose=False,
        #plot=True,
    )
    Path(model_path).parent.mkdir(parents=True, exist_ok=True)
    model.save_model(model_path)


if __name__ == '__main__':
    catboost_train_regression_op = create_component_from_func(
        catboost_train_regression,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['catboost==0.23']
    )
