# Model optimization for tensorflow slim models


This pipeline links 3 components:

- generating Tensorflow slim models
- model optimization with OpenVINO model optimizer
- running the inference evaluation based on input and label data in numpy files

## Examples of the parameters

model-name - resnet_v1_50<br />
model-name - 1000
checkpoint-url - http://download.tensorflow.org/models/resnet_v1_50_2016_08_28.tar.gz
batch-size - 1
tf-export-dir = /tmp/export
generated-model-dir = gs://your-bucket/folder
mo_options - saved_model_dir .
input_numpy_file - gs://intelai_public_models/images/imgs.npy
label-numpy-file - gs://intelai_public_models/images/lbs.npy



