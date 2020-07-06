from kfp.components import InputPath, OutputPath, create_component_from_func

def xgboost_predict(
    data_path: InputPath('ApacheParquet'),
    model_path: InputPath('XGBoostModel'),
    predictions_path: OutputPath('Text'),
    label_column_name: str = None,
):
    '''Make predictions using a trained XGBoost model.

    Args:
        data_path: Path for the feature data in Apache Parquet format.
        model_path: Path for the trained model in binary XGBoost format.
        predictions_path: Output path for the predictions.
        label_column_name: Optional. Name of the column containing the label data that is excluded during the prediction.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pathlib import Path

    import numpy
    import pandas
    import xgboost

    # Loading data
    df = pandas.read_parquet(data_path)
    if label_column_name:
        df = df.drop(columns=[label_column_name])

    evaluation_data = xgboost.DMatrix(
        data=df,
    )

    # Training
    model = xgboost.Booster(model_file=model_path)

    predictions = model.predict(evaluation_data)

    Path(predictions_path).parent.mkdir(parents=True, exist_ok=True)
    numpy.savetxt(predictions_path, predictions)


if __name__ == '__main__':
    create_component_from_func(
        xgboost_predict,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=[
            'xgboost==1.1.1',
            'pandas==1.0.5',
            'pyarrow==0.17.1',
        ]
    )
