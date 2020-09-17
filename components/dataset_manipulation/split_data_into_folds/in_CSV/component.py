from kfp.components import InputPath, OutputPath, create_component_from_func

def split_table_into_folds(
    table_path: InputPath('CSV'),

    train_1_path: OutputPath('CSV'),
    train_2_path: OutputPath('CSV'),
    train_3_path: OutputPath('CSV'),
    train_4_path: OutputPath('CSV'),
    train_5_path: OutputPath('CSV'),

    test_1_path: OutputPath('CSV'),
    test_2_path: OutputPath('CSV'),
    test_3_path: OutputPath('CSV'),
    test_4_path: OutputPath('CSV'),
    test_5_path: OutputPath('CSV'),

    number_of_folds: int = 5,
    random_seed: int = 0,
):
    """Splits the data table into the specified number of folds.

    The data is split into the specified number of folds k (default: 5).
    Each testing subsample has 1/k fraction of samples. The testing subsamples do not overlap.
    Each training subsample has (k-1)/k fraction of samples.
    The train_i subsample is produced by excluding test_i subsample form all samples.

    Inputs:
        table: The data to split by rows
        number_of_folds: Number of folds to split data into
        random_seed: Random seed for reproducible splitting

    Outputs:
        train_i: The i-th training subsample
        test_i: The i-th testing subsample

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>

    """
    import pandas
    from sklearn import model_selection

    max_number_of_folds = 5

    if number_of_folds < 1 or number_of_folds > max_number_of_folds:
        raise ValueError('Number of folds must be between 1 and {}.'.format(max_number_of_folds))

    df = pandas.read_csv(
        table_path,
    )
    splitter = model_selection.KFold(
        n_splits=number_of_folds,
        shuffle=True,
        random_state=random_seed,
    )
    folds = list(splitter.split(df))
    
    fold_paths = [
        (train_1_path, test_1_path),
        (train_2_path, test_2_path),
        (train_3_path, test_3_path),
        (train_4_path, test_4_path),
        (train_5_path, test_5_path),
    ]

    for i in range(max_number_of_folds):
        (train_path, test_path) = fold_paths[i]
        if i < len(folds):
            (train_indices, test_indices) = folds[i]
            train_fold = df.iloc[train_indices]
            test_fold = df.iloc[test_indices]
        else:
            train_fold = df.iloc[0:0]
            test_fold = df.iloc[0:0]
        train_fold.to_csv(train_path, index=False)
        test_fold.to_csv(test_path, index=False)


if __name__ == '__main__':
    split_table_into_folds_op = create_component_from_func(
        split_table_into_folds,
        base_image='python:3.7',
        packages_to_install=['scikit-learn==0.23.1', 'pandas==1.0.5'],
        output_component_file='component.yaml',
    )
