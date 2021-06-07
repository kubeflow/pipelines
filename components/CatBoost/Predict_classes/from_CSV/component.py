from kfp.components import InputPath, OutputPath, create_component_from_func

def catboost_predict_classes(
    data_path: InputPath('CSV'),
    model_path: InputPath('CatBoostModel'),
    predictions_path: OutputPath(),

    label_column: int = None,
):
    '''Predict classes using the CatBoost classifier model.

    Args:
        data_path: Path for the data in CSV format.
        model_path: Path for the trained model in binary CatBoostModel format.
        label_column: Column containing the label data.
        predictions_path: Output path for the predictions.

    Outputs:
        predictions: Class predictions in text format.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    import tempfile

    from catboost import CatBoostClassifier, Pool
    import numpy

    if label_column:
        column_descriptions = {label_column: 'Label'}
        column_description_path = tempfile.NamedTemporaryFile(delete=False).name
        with open(column_description_path, 'w') as column_description_file:
            for idx, kind in column_descriptions.items():
                column_description_file.write('{}\t{}\n'.format(idx, kind))
    else:
        column_description_path = None

    eval_data = Pool(
        data_path,
        column_description=column_description_path,
        has_header=True,
        delimiter=',',
    )

    model = CatBoostClassifier()
    model.load_model(model_path)

    predictions = model.predict(eval_data)
    numpy.savetxt(predictions_path, predictions, fmt='%s')


if __name__ == '__main__':
    catboost_predict_classes_op = create_component_from_func(
        catboost_predict_classes,
        output_component_file='component.yaml',
        base_image='python:3.7',
        packages_to_install=['catboost==0.22']
    )
