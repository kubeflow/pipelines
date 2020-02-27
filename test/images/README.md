This image is the base image for Kubeflow Pipelines [Prow job](https://github.com/kubernetes/test-infra/blob/6555278147dfff550706b41c3f69f41ecf5a8c5a/config/jobs/kubeflow/kubeflow-postsubmits.yaml#L245). Including presubmit and postsubmit jobs.

## How to build the image
To build this Docker image, run the following Docker command from the directory containing the image's files:

``` 
docker build -t gcr.io/ml-pipeline/test-worker:latest . 
```
## Where to push the image
This image is stored in the repo ml-pipeline, which is a public GCR repo. To push the image, run the following command:
1. Configure docker to use the gcloud command-line tool as a credential helper:

``` 
gcloud auth configure-docker 
```
2. Push the image to Container Registry:

``` 
docker push gcr.io/ml-pipeline/test-worker:latest 
```
