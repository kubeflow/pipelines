#!/usr/bin/env python
# coding: utf-8

# In[2]:


with open("requirements.txt", "w") as f:
    f.write("kfp==1.8.9\n")
    
get_ipython().system('pip install -r requirements.txt  --upgrade --user')


# In[1]:


from typing import NamedTuple

import kfp
from kfp import dsl
from kfp.components import func_to_container_op, InputPath, OutputPath

from typing import NamedTuple
def train(log_folder:str) -> NamedTuple('Outputs', [('logdir', str)]):
    
    print('mnist_func:', log_folder)
    import tensorflow as tf
    import json
    mnist = tf.keras.datasets.mnist
    (x_train,y_train), (x_test, y_test) = mnist.load_data()
    x_train, x_test = x_train/255.0, x_test/255.0

    def create_model():
        return tf.keras.models.Sequential([
            tf.keras.layers.Flatten(input_shape = (28,28)),
            tf.keras.layers.Dense(512, activation = 'relu'),
            tf.keras.layers.Dropout(0.2),
            tf.keras.layers.Dense(10, activation = 'softmax')
        ])
    model = create_model()
    model.compile(optimizer='adam',
                  loss='sparse_categorical_crossentropy',
                  metrics=['accuracy'])
    import datetime
    import os
    
    ### add tensorboard logout callback
    log_dir = os.path.join(log_folder, "logs", datetime.datetime.now().strftime("%Y%m%d-%H%M%S"))
    tensorboard_callback = tf.keras.callbacks.TensorBoard(log_dir=log_dir, histogram_freq=1)
    ######
    
    model.fit(x=x_train, 
              y=y_train, 
              epochs=5, 
              validation_data=(x_test, y_test), 
              callbacks=[tensorboard_callback])

    print('At least tensorboard callbacks are correct')
    print('logdir:', log_dir)
    return ([log_dir])

def prepare_tensorboard_from_localdir(pvc_name:str) -> NamedTuple('Outputs', [('mlpipeline_ui_metadata', 'UI_metadata')]):
    import json
    import kfp.components as components
    prepare_tensorboard = components.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1b107eb4bb2510ecb99fd5f4fb438cbf7c96a87a/components/contrib/tensorflow/tensorboard/prepare_tensorboard/component.yaml'
)
    # log_dir_uri is consisted of volume:///folders
    # the volume.name should be same as ones specified in pod_template.spec.volumes.name
    
    return prepare_tensorboard(
        log_dir_uri=f'volume://mypvc/logs', 
        image="footprintai/tensorboard:2.7.0",      
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

def model(text1):
  return dsl.ContainerOp(
      name='model',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text1])


# In[2]:


import kfp.dsl as dsl
import kfp.components as components
import time


@dsl.pipeline(
   name='tf pipeline',
   description='A pipeline to train a model on tf dataset and start a tensorboard.'
)
def tf_pipeline(text1='message 1'):

    log_folder = '/data'
    pvc_name = 'input'
    unique_pvc_resource_name = 'my-awesome-kf-workshop-%d'% int(time.time())
        
    vop = dsl.VolumeOp(
        name=pvc_name,
        resource_name=unique_pvc_resource_name,
        size="1Gi",
        modes=dsl.VOLUME_MODE_RWO,
        generate_unique_name=False,
    )
    tf_op = func_to_container_op(
        func=train,
        base_image="tensorflow/tensorflow:2.0.0-py3",
    )
    tensorboard_task = prepare_tensorboard_from_localdir(unique_pvc_resource_name)
    
    tf_task = tf_op(log_folder).add_pvolumes({
        log_folder:vop.volume,
    })

        
    
    step1_task = model(text1)
#   step1_task.after(tensorboard_task)    
    tensorboard_task.after(tf_task)
    tf_task.after(step1_task)


# In[4]:


kfp.compiler.Compiler().compile(tf_pipeline, 'helloworld.zip')


# In[ ]:




