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

import os
import json
from kfp.onprem import use_k8s_secret
from kfp import dsl, components
from kfp.components import OutputPath, create_component_from_func

prepare_tensorboard = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        '../../../components/tensorflow/tensorboard/prepare_tensorboard/component.yaml'
    )
)
# New features used locally not released yet, so we can only use local import.
# prepare_tensorboard = components.load_component_from_url(
#     'https://raw.githubusercontent.com/kubeflow/pipelines/1.5.0/components/tensorflow/tensorboard/prepare_tensorboard/component.yaml'
# )


def train(minio_endpoint: 'URI', log_bucket: str, log_dir: 'Path'):
    # Reference: https://www.tensorflow.org/tensorboard/get_started
    import tensorflow as tf
    import datetime

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

    log_dir_local = "logs/fit"
    tensorboard_callback = tf.keras.callbacks.TensorBoard(
        log_dir=log_dir_local, histogram_freq=1
    )

    model.fit(
        x=x_train,
        y=y_train,
        epochs=5,
        validation_data=(x_test, y_test),
        callbacks=[tensorboard_callback]
    )

    # Copy the local logs folder to minio.
    #
    # TODO: we may write a filesystem watch process that continuously copy logs
    # dir to minio, so that we can watch live training logs via tensorboard.
    #
    # Note, although tensorflow supports minio via s3:// protocol. We want to
    # demo how minio can be used instead, e.g. the same approach can be used with
    # frameworks only support local path.
    from minio import Minio
    import os
    minio_access_key = os.getenv('MINIO_ACCESS_KEY')
    minio_secret_key = os.getenv('MINIO_SECRET_KEY')
    if not minio_access_key or not minio_secret_key:
        raise Exception('MINIO_ACCESS_KEY or MINIO_SECRET_KEY env is not set')
    client = Minio(
        minio_endpoint,
        access_key=minio_access_key,
        secret_key=minio_secret_key,
        secure=False
    )
    count = 0
    from pathlib import Path
    for path in Path("logs").rglob("*"):
        if not path.is_dir():
            object_name = os.path.join(
                log_dir, os.path.relpath(start=log_dir_local, path=path)
            )
            client.fput_object(
                bucket_name=log_bucket,
                object_name=object_name,
                file_path=path,
            )
            count = count + 1
            print(f'{path} uploaded to minio://{log_bucket}/{object_name}')
    print(f'{count} log files uploaded to minio://{log_bucket}/{log_dir}')


# tensorflow/tensorflow:2.4 may fail with image pull backoff, because of dockerhub rate limiting.
train_op = create_component_from_func(
    train,
    base_image='gcr.io/deeplearning-platform-release/tf2-cpu.2-3:latest',
    packages_to_install=['minio'],  # TODO: pin minio version
)


@dsl.pipeline(name='pipeline-tensorboard-minio')
def my_pipeline(
    minio_endpoint='minio-service:9000',
    log_bucket='mlpipeline',
    log_dir=f'tensorboard/logs/{dsl.RUN_ID_PLACEHOLDER}',
    # Pin to tensorflow 2.3, because in 2.4+ tensorboard cannot load in KFP:
    # refer to https://github.com/kubeflow/pipelines/issues/5521.
    tf_image='gcr.io/deeplearning-platform-release/tf2-cpu.2-3:latest'
):
    # tensorboard uses s3 protocol to access minio
    prepare_tb_task = prepare_tensorboard(
        log_dir_uri=f's3://{log_bucket}/{log_dir}',
        image=tf_image,
        pod_template_spec=json.dumps({
            'spec': {
                'containers': [{
                    # These env vars make tensorboard access KFP in-cluster minio
                    # using s3 protocol.
                    # Reference: https://blog.min.io/hyper-scale-machine-learning-with-minio-and-tensorflow/
                    'env': [{
                        'name': 'AWS_ACCESS_KEY_ID',
                        'valueFrom': {
                            'secretKeyRef': {
                                'name': 'mlpipeline-minio-artifact',
                                'key': 'accesskey'
                            }
                        }
                    }, {
                        'name': 'AWS_SECRET_ACCESS_KEY',
                        'valueFrom': {
                            'secretKeyRef': {
                                'name': 'mlpipeline-minio-artifact',
                                'key': 'secretkey'
                            }
                        }
                    }, {
                        'name': 'AWS_REGION',
                        'value': 'minio'
                    }, {
                        'name': 'S3_ENDPOINT',
                        'value': f'{minio_endpoint}',
                    }, {
                        'name': 'S3_USE_HTTPS',
                        'value': '0',
                    }, {
                        'name': 'S3_VERIFY_SSL',
                        'value': '0',
                    }]
                }],
            },
        })
    )
    train_task = train_op(
        minio_endpoint=minio_endpoint,
        log_bucket=log_bucket,
        log_dir=log_dir,
    )
    train_task.apply(
        use_k8s_secret(
            secret_name='mlpipeline-minio-artifact',
            k8s_secret_key_to_env={
                'secretkey': 'MINIO_SECRET_KEY',
                'accesskey': 'MINIO_ACCESS_KEY'
            },
        )
    )
    # optional, let training task use the same tensorflow image as specified tensorboard
    train_task.container.image = tf_image
    train_task.after(prepare_tb_task)
