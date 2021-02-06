import keras
import kfp
from kfp import components


chicago_taxi_dataset_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/e3337b8bdcd63636934954e592d4b32c95b49129/components/datasets/Chicago%20Taxi/component.yaml')
pandas_transform_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/6162d55998b176b50267d351241100bb0ee715bc/components/pandas/Transform_DataFrame/in_CSV_format/component.yaml')
keras_train_classifier_from_csv_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/f6aabf7f10b1f545f1fd5079aa8071845224f8e7/components/keras/Train_classifier/from_CSV/component.yaml')
keras_convert_hdf5_model_to_tf_saved_model_op = components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/51e49282d9511e4b72736c12dc66e37486849c6e/components/_converters/KerasModelHdf5/to_TensorflowSavedModel/component.yaml')


number_of_classes = 2

# Creating the network
dense_network_with_sigmoid = keras.Sequential(layers=[
    keras.layers.Dense(10, activation=keras.activations.sigmoid),
    keras.layers.Dense(number_of_classes, activation=keras.activations.sigmoid),
])


def keras_classifier_pipeline():
    training_data_in_csv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=1000,
    ).output

    training_data_for_classification_in_csv = pandas_transform_csv_op(
        table=training_data_in_csv,
        transform_code='''df.insert(0, "was_tipped", df["tips"] > 0); del df["tips"]; df = df.fillna(0)''',
    ).output

    features_in_csv = pandas_transform_csv_op(
        table=training_data_for_classification_in_csv,
        transform_code='''df = df.drop(columns=["was_tipped"])''',
    ).output

    labels_in_csv = pandas_transform_csv_op(
        table=training_data_for_classification_in_csv,
        transform_code='''df = df["was_tipped"] * 1''',
    ).output

    keras_model_in_hdf5 = keras_train_classifier_from_csv_op(
        training_features=features_in_csv,
        training_labels=labels_in_csv,
        network_json=dense_network_with_sigmoid.to_json(),
        learning_rate=0.1,
        num_epochs=100,
    ).outputs['model']

    keras_model_in_tf_format = keras_convert_hdf5_model_to_tf_saved_model_op(
        model=keras_model_in_hdf5,
    ).output
    
if __name__ == '__main__':
    kfp_endpoint = None
    kfp.Client(host=kfp_endpoint).create_run_from_pipeline_func(keras_classifier_pipeline, arguments={})
