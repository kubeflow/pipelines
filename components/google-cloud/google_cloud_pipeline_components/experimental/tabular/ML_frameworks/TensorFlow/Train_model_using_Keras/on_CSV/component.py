from kfp.components import create_component_from_func, InputPath, OutputPath


def train_model_using_Keras_on_CSV(
    training_data_path: InputPath("CSV"),
    model_path: InputPath("TensorflowSavedModel"),
    trained_model_path: OutputPath("TensorflowSavedModel"),
    label_column_name: str,
    loss_function_name: str = "mean_squared_error",
    number_of_epochs: int = 1,
    learning_rate: float = 0.1,
    optimizer_name: str = "Adadelta",
    optimizer_parameters: dict = None,
    batch_size: int = 32,
    metric_names: list = None,
    random_seed: int = 0,
):
    import tensorflow as tf
    tf.random.set_seed(seed=random_seed)

    # Loading model using Keras. Model loaded using TensorFlow does not have .fit.
    #model = tf.saved_model.load(export_dir=model_path)
    keras_model = tf.keras.models.load_model(filepath=model_path)

    optimizer_parameters = optimizer_parameters or {}
    optimizer_parameters["learning_rate"] = learning_rate
    optimizer_config = {
        "class_name": optimizer_name,
        "config": optimizer_parameters,
    }
    optimizer = tf.keras.optimizers.get(optimizer_config)
    loss = tf.keras.losses.get(loss_function_name)

    training_dataset = tf.data.experimental.make_csv_dataset(
        file_pattern=training_data_path,
        batch_size=batch_size,
        label_name=label_column_name,
        header=True,
        # Need to specify num_epochs=1 otherwise the training becomes infinite
        num_epochs=1,
        shuffle=True,
        shuffle_seed=random_seed,
        ignore_errors=True,
    )
    def stack_feature_batches(features_batch, labels_batch):
        # Need to stack individual feature columns to create a single feature tensor
        # Need to cast all column tensor types to float to prevent error:
        # TypeError: Tensors in list passed to 'values' of 'Pack' Op have types [int32, float32, float32, int32, int32] that don't all match.
        list_of_feature_batches = list(tf.cast(x=feature_batch, dtype=tf.float32) for feature_batch in features_batch.values())
        return tf.stack(list_of_feature_batches, axis=-1), labels_batch

    training_dataset = training_dataset.map(stack_feature_batches)

    # Need to compile the model to prevent error:
    # ValueError: No gradients provided for any variable: [..., ...].
    keras_model.compile(
        optimizer=optimizer,
        loss=loss,
        metrics=metric_names,
    )
    keras_model.fit(
        training_dataset,
        epochs=number_of_epochs,
    )

    # Using tf.keras.models.save_model instead of tf.saved_model.save to prevent downstream error:
    #tf.saved_model.save(keras_model, trained_model_path)
    # ValueError: Unable to create a Keras model from this SavedModel.
    # This SavedModel was created with `tf.saved_model.save`, and lacks the Keras metadata.
    # Please save your Keras model by calling `model.save`or `tf.keras.models.save_model`.
    # See https://github.com/keras-team/keras/issues/16451
    tf.keras.models.save_model(keras_model, trained_model_path)


if __name__ == "__main__":
    import os
    import re
    # Fixing google3 paths
    if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
        rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
        os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

    train_model_using_Keras_on_CSV_op = create_component_from_func(
        train_model_using_Keras_on_CSV,
        output_component_file="component.yaml",
        base_image="tensorflow/tensorflow:2.8.0",
        packages_to_install=[],
    )