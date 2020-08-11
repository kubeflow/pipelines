from typing import NamedTuple
from kfp.components import create_component_from_func, InputPath, OutputPath

def keras_train_classifier_from_csv(
    training_features_path: InputPath('CSV'),
    training_labels_path: InputPath('CSV'),
    network_json_path: InputPath('KerasModelJson'),
    model_path: OutputPath('KerasModelHdf5'),
    loss_name: str = 'categorical_crossentropy',
    num_classes: int = None,
    optimizer: str = 'rmsprop',
    optimizer_config: dict = None,
    learning_rate: float = 0.01,
    num_epochs: int = 100,
    batch_size: int = 32,
    metrics: list = ['accuracy'],
    random_seed: int = 0,
) -> NamedTuple('Outputs', [
    ('final_loss', float),
    ('final_metrics', dict),
    ('metrics_history', dict),
]):
    '''Trains classifier model using Keras.

    Annotations:
        author: Alexey Volkov <alexey.volkov@ark-kun.com>
    '''
    from pathlib import Path

    import keras
    import numpy
    import pandas
    import tensorflow

    tensorflow.random.set_seed(random_seed)
    numpy.random.seed(random_seed)
    
    training_features_df = pandas.read_csv(training_features_path)
    training_labels_df = pandas.read_csv(training_labels_path)

    x_train = training_features_df.to_numpy()
    y_train_labels = training_labels_df.to_numpy()
    print('Training features shape:', x_train.shape)
    print('Numer of training samples:', x_train.shape[0])

    # Convert class vectors to binary class matrices.
    y_train_one_hot = keras.utils.to_categorical(y_train_labels, num_classes)

    model_json_str = Path(network_json_path).read_text()
    model = keras.models.model_from_json(model_json_str)

    model.add(keras.layers.Activation('softmax'))

    # Initializing the optimizer
    optimizer_config = optimizer_config or {}
    optimizer_config['learning_rate'] = learning_rate
    optimizer = keras.optimizers.deserialize({
        'class_name': optimizer,
        'config': optimizer_config,
    })

    model.compile(
        loss=loss_name,
        optimizer=optimizer,
        metrics=metrics,
    )

    history = model.fit(
        x_train,
        y_train_one_hot,
        batch_size=batch_size,
        epochs=num_epochs,
        shuffle=True
    )

    model.save(model_path)

    metrics_history = {name: [float(value) for value in values] for name, values in history.history.items()}
    final_metrics = {name: values[-1] for name, values in metrics_history.items()}
    final_loss = final_metrics['loss']
    return (final_loss, final_metrics, metrics_history)


if __name__ == '__main__':
    keras_train_classifier_from_csv_op = create_component_from_func(
        keras_train_classifier_from_csv,
        base_image='tensorflow/tensorflow:2.2.0',
        packages_to_install=['keras==2.3.1', 'pandas==1.0.5'],
        output_component_file='component.yaml',
    )
