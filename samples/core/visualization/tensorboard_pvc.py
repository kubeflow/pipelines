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
from kfp import dsl, components
from kfp.components import func_to_container_op

from typing import NamedTuple
def prepare_tensorboard_from_localdir(pvc_name:str = '', tf_image: str = 'gcr.io/deeplearning-platform-release/tf2-cpu.2-3:latest') -> NamedTuple('Outputs', [('mlpipeline_ui_metadata', 'UI_metadata')]):

    prepare_tensorboard = components.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1b107eb4bb2510ecb99fd5f4fb438cbf7c96a87a/components/contrib/tensorflow/tensorboard/prepare_tensorboard/component.yaml'
)
    # log_dir_uri is consisted of volume://<volume.name>/folders
    # the volume.name should be same as ones specified in pod_template.spec.volumes.name
    return prepare_tensorboard(
        log_dir_uri=f'volume://mypvc/logs',
        image=tf_image,
        pod_template_spec=json.dumps({
          "spec": {
            "containers": [
              {
                "volumeMounts": [
                  {
                    "mountPath": "/data",
                    "name": "mypvc"
                  }
                ]
              }
            ],
            "serviceAccountName": "default-editor",
            "volumes": [
              {
                "name": "mypvc",
                "persistentVolumeClaim": {
                  "claimName": pvc_name
                }
              }
            ]
          }
        }),
    )


def train(log_dir: str):
    # Reference: https://www.tensorflow.org/tensorboard/get_started
    import datetime
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

    log_dir_local = os.path.join(log_dir, "logs", datetime.datetime.now().strftime("%Y%m%d-%H%M%S"))
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

@dsl.pipeline(name='pipeline-tensorboard-pvc')
def my_pipeline(
    # Pin to tensorflow 2.3, because in 2.4+ tensorboard cannot load in KFP:
    # refer to https://github.com/kubeflow/pipelines/issues/5521.
    tf_image = 'gcr.io/deeplearning-platform-release/tf2-cpu.2-3:latest',
):

    import time

    log_dir = '/data'
    pvc_name = 'mypvc'

    # we generated an unique as an argument to launch pvc as its default name
    # So we could statically bind the pvc name to tensorboard
    # otherwise, the tensorboard are unlikely to automatically retrieve the pvc name as it is generated during runtime
    unique_pvc_resource_name = 'my-temporal-pvc-name-%d'% int(time.time())
    
    vop = dsl.VolumeOp(
        name=pvc_name,
        resource_name=unique_pvc_resource_name,
        size="1Gi",
        modes=dsl.VOLUME_MODE_RWO,
        generate_unique_name=False,
    )
    train_op = func_to_container_op(
        func=train,
        base_image=tf_image,
    )
    train_task = train_op(log_dir).add_pvolumes({
        log_dir:vop.volume,
    }).set_memory_request('2Gi').set_memory_limit('2Gi')

    tensorboard_task = prepare_tensorboard_from_localdir(unique_pvc_resource_name)
    train_task.after(tensorboard_task)

    # NOTE: need to cleanup pvc to avoid resource leaking

