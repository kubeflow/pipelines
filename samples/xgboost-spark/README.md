## Disclaimer: 
**XGBoost DSL is under testing.**

## The requirements:
* Preprocessing uses Google Cloud DataFlow. So [DataFlow API](https://cloud.google.com/endpoints/docs/openapi/enable-api) needs to be enabled for the given project.

## Compile
Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and 
compile your sample python into workflow yaml.

## Deploy
To run a classification training pipeline sample with SFPD data:
* Prepare output directory  
Create a GCS bucket to store the generated model. Make sure it's in the same project as the ML pipeline deployed above.

```bash
gsutil mb gs://[YOUR_GCS_BUCKET]
```

* Deploy  
Open the ML pipeline UI.  
Kubeflow-training-classification requires two argument:

```
project: MY_GCP_PROJECT
output: gs://[YOUR_GCS_BUCKET]
train: gs://ml-pipeline-playground/sfpd/train.csv"
eval: gs://ml-pipeline-playground/sfpd/eval.csv"
schema: gs://ml-pipeline-playground/sfpd/schema.json"
```

<!---
#TODO: this will be added to the readme after testing. argo commands would mislead users from the web console.

To run a classification training pipeline sample with SFPD data:

argo submit xgboost-training-roc.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/sfpdmodel" \
     -p train="gs://ml-pipeline-playground/sfpd/train.csv" \
     -p eval="gs://ml-pipeline-playground/sfpd/eval.csv" \
     -p schema="gs://ml-pipeline-playground/sfpd/schema.json" \
     -p target=resolution \
     -p trueclass=ACTION \
     -p workers=2 \
     -p rounds=100 \
     -p conf=gs://ml-pipeline-playground/trainconfcla.json \
     --entrypoint xgboost-training

To run a classification training pipeline sample with 20NewsGroups data:

argo submit xgboost-training-cm.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/newsmodel" \
     -p train="gs://ml-pipeline-playground/newsgroup/train.csv" \
     -p eval="gs://ml-pipeline-playground/newsgroup/eval.csv" \
     -p schema="gs://ml-pipeline-playground/newsgroup/schema.json" \
     -p target=news_label \
     -p workers=2 \
     -p rounds=200 \
     -p conf=gs://ml-pipeline-playground/trainconfcla.json \
     --entrypoint xgboost-training


To run an evaluation pipeline sample with 20NewsGroups model & testdata:

argo submit xgboost-evaluation.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/newsmodel" \
     -p eval="gs://ml-pipeline-playground/newsgroup/eval.csv" \
     -p model="gs://ml-pipeline-playground/newsgroup/model/model" \
     -p target=news_label \
     -p trueclass=talk.politics.mideast \
     -p analysis=gs://ml-pipeline-playground/newsgroup/analysis/ \
     --entrypoint xgboost-evaluation
--->


