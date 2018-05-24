The requirements:

* requires a GKE cluster with [argo](https://github.com/argoproj/argo) and
  [kubeflow](https://github.com/kubeflow/kubeflow) installed.
  The GKE cluster needs cloud-platform scope. For example:

  ```gcloud container clusters create [your-gke-cluster-name] --zone us-central1-a --scopes cloud-platform```

* Preprocessing uses Google Cloud DataFlow. So DataFlow API needs to be enabled for given project.

The sample 

To run an image classification training pipeline sample:

argo submit kubeflow-training-classification.yaml \
     -p project=MY_GCP_PROJECT \
     -p output="gs://my-bucket/flowermodel" \
     -p schema=gs://ml-pipeline-playground/flower/schema.json \
     -p train=gs://ml-pipeline-playground/flower/train200.csv \
     -p eval=gs://ml-pipeline-playground/flower/eval100.csv \
     -p target=label \
     -p hidden-layer-size="100,50" \
     -p steps=2000 \
     -p learning-rate=0.1 \
     --entrypoint kubeflow-training


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




