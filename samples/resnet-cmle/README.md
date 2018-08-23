## The requirements:
Preprocessing uses Google Cloud DataFlow. So [DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project.

Training and serving uses Google Cloud Machine Learning Engine. So [Cloud Machine Learning Engine API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project. 

## Compile
Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and 
compile your python sample into workflow yaml.

## Deploy
To run an image classification training pipeline sample:
### Prepare output directory  
Create a GCS bucket to store the generated model. Make sure it's in the same project as the ML pipeline deployed above.

```bash
gsutil mb gs://[YOUR_GCS_BUCKET]
```

### Deploy  
Open the ML pipeline UI.  
Kubeflow-training-classification requires several arguments:

```
project-id: MY_GCP_PROJECT_ID
bucket: MY_GCS_BUCKET
region: MY_GCP_REGION
model: MY_MODEL_NAME
version: MY_MODEL_VERSION
model-dir: gs://[PATH_TO_STORE_MODEL]
tf-version: MY_TFVERSION
train-csv : gs://[PATH_TO_TRAIN_CSV]
validation-csv: gs://[PATH_TO_EVAL_CSV]
labels: gs://[PATH_TO_LABELS_TXT]
depth: DEPTH_OF_RESNET
train-batch-size: TRAIN_BATCH_SIZE
eval-batch-size: EVAL_BATCH_SIZE
steps-per-eval: STEPS_PER_EVAL
train-steps: NUM_OF_TRAIN_STEPS
num-train-images: NUM_OF_TRAIN_IMAGES
num-eval-images: NUM_OF_EVAL_IMAGES
num-label-classes: NUM_OF_CLASSES
```
