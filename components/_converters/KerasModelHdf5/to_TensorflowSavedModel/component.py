from kfp.components import create_component_from_func, InputPath, OutputPath

def keras_convert_hdf5_model_to_tf_saved_model(
    model_path: InputPath('KerasModelHdf5'),
    converted_model_path: OutputPath('TensorflowSavedModel'),
):
    '''Converts Keras HDF5 model to Tensorflow SavedModel format.

    Args:
        model_path: Keras model in HDF5 format.
        converted_model_path: Keras model in Tensorflow SavedModel format.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pathlib import Path
    from tensorflow import keras
    
    model = keras.models.load_model(filepath=model_path)
    keras.models.save_model(model=model, filepath=converted_model_path, save_format='tf')


if __name__ == '__main__':
    keras_convert_hdf5_model_to_tf_saved_model_op = create_component_from_func(
        keras_convert_hdf5_model_to_tf_saved_model,
        base_image='tensorflow/tensorflow:2.3.0',
        packages_to_install=['h5py==2.10.0'],
        output_component_file='component.yaml',
    )
