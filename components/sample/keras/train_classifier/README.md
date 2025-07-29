# Keras - Train classifier
### Trains classifier using Keras sequential model

## Inputs
|Name|Type|Default|Description|
|---|---|---|---|
|training_set_features_path|GcsPath: {data_type: TSV}||Local or GCS path to the training set features table.|
|training_set_labels_path|GcsPath: {data_type: TSV}||Local or GCS path to the training set labels (each label is a class index from 0 to num-classes - 1).|
|output_model_uri|GcsPath: {data_type: Keras model}||Local or GCS path specifying where to save the trained model. The model (topology + weights + optimizer state) is saved in HDF5 format and can be loaded back by calling keras.models.load_model|
|model_config|GcsPath: {data_type: Keras model config json}||JSON string containing the serialized model structure. Can be obtained by calling model.to_json() on a Keras model.|
|number_of_classes|Integer||Number of classifier classes.|
|number_of_epochs|Integer|100|Number of epochs to train the model. An epoch is an iteration over the entire `x` and `y` data provided.|
|batch_size|Integer|32|Number of samples per gradient update|

## Outputs
|Name|Type|Default|Description|
|---|---|---|---|
|output_model_uri|GcsPath: {data_type: Keras model}||GCS path where the trained model has been saved. The model (topology + weights + optimizer state) is saved in HDF5 format and can be loaded back by calling keras.models.load_model|

## Container image
gcr.io/ml-pipeline/components/sample/keras/train_classifier

## Usage:

```python
import os
from pathlib import Path
import requests

import kfp

component_url_prefix = 'https://raw.githubusercontent.com/kubeflow/pipelines/master/components/sample/keras/train_classifier/'
test_data_url_prefix = component_url_prefix + 'tests/testdata/'

#Prepare input/output paths and data
input_data_gcs_dir = 'gs://<my bucket>/<path>/'
output_data_gcs_dir = 'gs://<my bucket>/<path>/'

#Downloading the training set (to upload to GCS later)
training_set_features_local_path = os.path.join('.', 'training_set_features.tsv')
training_set_labels_local_path = os.path.join('.', 'training_set_labels.tsv')

training_set_features_url = test_data_url_prefix + '/training_set_features.tsv'
training_set_labels_url = test_data_url_prefix + '/training_set_labels.tsv'

Path(training_set_features_local_path).write_bytes(requests.get(training_set_features_url).content)
Path(training_set_labels_local_path).write_bytes(requests.get(training_set_labels_url).content)

#Uploading the data to GCS where it can be read by the trainer
training_set_features_gcs_path = os.path.join(input_data_gcs_dir, 'training_set_features.tsv')
training_set_labels_gcs_path = os.path.join(input_data_gcs_dir, 'training_set_labels.tsv')

gfile.Copy(training_set_features_local_path, training_set_features_gcs_path)
gfile.Copy(training_set_labels_local_path, training_set_labels_gcs_path)

output_model_uri_template = os.path.join(output_data_gcs_dir, kfp.dsl.EXECUTION_ID_PLACEHOLDER, 'output_model_uri', 'data')

xor_model_config = requests.get(test_data_url_prefix + 'model_config.json').content


#Load the component
train_op = kfp.components.load_component_from_url(component_url_prefix + 'component.yaml')

#Use the component as part of the pipeline
@kfp.dsl.pipeline(name='Test keras/train_classifier', description='Pipeline to test keras/train_classifier component')
def pipeline_to_test_keras_train_classifier():
    train_task = train_op(
        training_set_features_path=training_set_features_gcs_path,
        training_set_labels_path=training_set_labels_gcs_path,
        output_model_uri=output_model_uri_template,
        model_config=xor_model_config,
        number_of_classes=2,
        number_of_epochs=10,
        batch_size=32,
    )
    #Use train_task.outputs['output_model_uri'] to obtain the reference to the trained model URI that can be a passed to other pipeline tasks (e.g. for prediction or analysis)
```
