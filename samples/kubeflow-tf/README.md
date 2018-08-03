## The requirements:
Preprocessing uses Google Cloud DataFlow. So [DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project. 

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
Kubeflow-training-classification requires two argument:

```
project: MY_GCP_PROJECT
output: gs://[YOUR_GCS_BUCKET]
```

**Note that chicago taxi fare prediction training pipeline 
[sample](https://github.com/googleprivate/ml/blob/master/samples/kubeflow-tf/kubeflow-training-regression.yaml) is under testing.**

<!---
#TODO: since this is not tested, it is commented out for now.

To run a chicago taxi fare prediction training pipeline sample:

argo submit kubeflow-training-regression.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/taximodel" \
     -p schema=gs://ml-pipeline-playground/taxi/schema.json \
     -p train=gs://ml-pipeline-playground/taxi/train.csv \
     -p eval=gs://ml-pipeline-playground/taxi/eval.csv \
     -p target=fare \
     -p analyze-slice-column=company \
     -p hidden-layer-size="1500" \
     -p steps=3000 \
     -p learning-rate=0.1 \
     --entrypoint kubeflow-training
--->
