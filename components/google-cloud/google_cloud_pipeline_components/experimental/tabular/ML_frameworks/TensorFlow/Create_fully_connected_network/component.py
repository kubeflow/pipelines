from kfp.components import create_component_from_func, OutputPath


def create_fully_connected_tensorflow_network(
    input_size: int,
    model_path: OutputPath("TensorflowSavedModel"),
    hidden_layer_sizes: list = [],
    output_size: int = 1,
    activation_name: str = "relu",
    output_activation_name: str = None,
    random_seed: int = 0,
):
    """Creates fully-connected network in Tensorflow SavedModel format"""
    import tensorflow as tf
    tf.random.set_seed(seed=random_seed)

    model = tf.keras.models.Sequential()
    model.add(tf.keras.Input(shape=(input_size,)))
    for layer_size in hidden_layer_sizes:
        model.add(tf.keras.layers.Dense(units=layer_size, activation=activation_name))
    # The last layer is left without activation
    model.add(tf.keras.layers.Dense(units=output_size, activation=output_activation_name))

    print(model.summary())

    # Using tf.keras.models.save_model instead of tf.saved_model.save to prevent downstream error:
    #tf.saved_model.save(model, model_path)
    # ValueError: Unable to create a Keras model from this SavedModel.
    # This SavedModel was created with `tf.saved_model.save`, and lacks the Keras metadata.
    # Please save your Keras model by calling `model.save`or `tf.keras.models.save_model`.
    # See https://github.com/keras-team/keras/issues/16451
    tf.keras.models.save_model(model, model_path)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    create_fully_connected_tensorflow_network_op = create_component_from_func(
        create_fully_connected_tensorflow_network,
        output_component_file="component.yaml",
        base_image="tensorflow/tensorflow:2.7.0",
        packages_to_install=[],
    )