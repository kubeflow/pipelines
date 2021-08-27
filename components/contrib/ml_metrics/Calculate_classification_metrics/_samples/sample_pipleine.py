from pathlib import Path

import kfp
from kfp.components import ComponentStore, create_component_from_func, InputPath, OutputPath, load_component_from_file

store = ComponentStore.default_store

chicago_taxi_dataset_op = store.load_component('datasets/Chicago_Taxi_Trips')
xgboost_train_on_csv_op = store.load_component('XGBoost/Train')
xgboost_predict_on_csv_op = store.load_component('XGBoost/Predict')
pandas_transform_csv_op = store.load_component('pandas/Transform_DataFrame/in_CSV_format')
drop_header_op = store.load_component('tables/Remove_header')


def convert_values_to_int(text_path: InputPath('Text'),
                          output_path: OutputPath('Text')):
    """Returns the number of values in a CSV column."""
    import numpy as np

    result = np.loadtxt(text_path)

    np.savetxt(output_path, result, fmt='%d')


convert_values_to_int_op = create_component_from_func(
    func=convert_values_to_int,
    base_image='python:3.7',
    packages_to_install=['pandas==1.1'],
)

calculate_classification_metrics_from_csv_op = load_component_from_file(
    str(Path(__file__).parent.parent / 'from_CSV' / 'component.yaml')
)


def classification_metrics_pipeline():
    features = ['trip_seconds', 'trip_miles', 'pickup_community_area', 'dropoff_community_area',
                'fare', 'tolls', 'extras', 'trip_total']
    target = 'company'

    training_data_csv = chicago_taxi_dataset_op(
        select=','.join([target] + features),
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        limit=100,
    ).output

    training_data_transformed_csv = pandas_transform_csv_op(
        table=training_data_csv,
        transform_code=f'''df["{target}"] = df["{target}"].astype('category').cat.codes''',
    ).output

    # Training
    model_trained_on_csv = xgboost_train_on_csv_op(
        training_data=training_data_transformed_csv,
        label_column=0,
        booster_params={'num_class': 13},
        objective='multi:softmax',
        num_iterations=50,
    ).outputs['model']

    # Predicting
    predictions = xgboost_predict_on_csv_op(
        data=training_data_csv,
        model=model_trained_on_csv,
        label_column=0,
    ).output

    predictions_converted = convert_values_to_int_op(
        text=predictions
    ).output

    # Preparing the true values
    true_values_table = pandas_transform_csv_op(
        table=training_data_csv,
        transform_code=f'df["{target}"] = df["{target}"].astype("category").cat.codes\n'
                       f'df = df[["{target}"]]'
    ).output

    true_values = drop_header_op(true_values_table).output

    # Calculating the regression metrics
    calculate_classification_metrics_from_csv_op(
        true_values=true_values,
        predicted_values=predictions_converted,
        average='macro',
    )


if __name__ == '__main__':
    kfp_endpoint = None

    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(classification_metrics_pipeline, arguments={})
