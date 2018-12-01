# Model optimization for tensorflow slim models


This pipeline links generating Tensorflow slim models with OpenVINO model optimization.


## Examples of the parameters

model_name - resnet_v1_50<br />
checkpoint_url - http://download.tensorflow.org/models/resnet_v1_50_2016_08_28.tar.gz<br />
batch_size - 8<br />
num_classes - 1000<br />
saved_model_dir - gs://<bucket>/resnet/1<br />
export_dir - /tmp/export<br />


model_name - inception_v4
checkpoint_url - http://download.tensorflow.org/models/inception_v4_2016_09_09.tar.gz
batch_size - -1
num_classes - 1001
saved_model_dir - gs://<bucket>/inception/1
export_dir - gs://<bucket>/inception_files/1

