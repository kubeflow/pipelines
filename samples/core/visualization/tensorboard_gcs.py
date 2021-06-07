# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from kfp import dsl, components
from kfp.components import create_component_from_func

prepare_tensorboard = components.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1.5.0/components/tensorflow/tensorboard/prepare_tensorboard/component.yaml'
)


def train(log_dir: 'URI'):
    # Reference: https://www.tensorflow.org/tensorboard/get_started
    import tensorflow as tf

    mnist = tf.keras.datasets.mnist

    (x_train, y_train), (x_test, y_test) = mnist.load_data()
    x_train, x_test = x_train / 255.0, x_test / 255.0

    def create_model():
        return tf.keras.models.Sequential([
            tf.keras.layers.Flatten(input_shape=(28, 28)),
            tf.keras.layers.Dense(512, activation='relu'),
            tf.keras.layers.Dropout(0.2),
            tf.keras.layers.Dense(10, activation='softmax')
        ])

    model = create_model()
    model.compile(
        optimizer='adam',
        loss='sparse_categorical_crossentropy',
        metrics=['accuracy']
    )

    tensorboard_callback = tf.keras.callbacks.TensorBoard(
        log_dir=log_dir, histogram_freq=1
    )

    model.fit(
        x=x_train,
        y=y_train,
        epochs=5,
        validation_data=(x_test, y_test),
        callbacks=[tensorboard_callback]
    )


# Be careful when choosing a tensorboard image:
# * tensorflow/tensorflow may fail with image pull backoff, because of dockerhub rate limiting.
# * tensorboard in tensorflow 2.3+ does not work with KFP, refer to https://github.com/kubeflow/pipelines/issues/5521.
train_op = create_component_from_func(
    train, base_image='gcr.io/deeplearning-platform-release/tf2-cpu.2-4'
)


@dsl.pipeline(name='pipeline-tensorboard-gcs')
def my_pipeline(
    log_dir=f'gs://{{kfp-default-bucket}}/tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}'
):
    prepare_tb_task = prepare_tensorboard(log_dir_uri=log_dir)
    tensorboard_task = train_op(log_dir=prepare_tb_task.outputs['log_dir_uri'],)
