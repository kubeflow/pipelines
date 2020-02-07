This image is the base image for Kubeflow Pipelines [Prow job](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubeflow/kubeflow-postsubmits.yaml#L245). Including presubmit and postsubmit jobs.

## How to build the image
To build this Docker image, run the following Docker command from the directory containing the image's files:
``` docker build -t test-worker . ```
## Where to push the image
This image stores in the repo ml-pipeline-test, which is a public GCR repo. To push the image, run the following command:
1. Configure docker to use the gcloud command-line tool as a credential helper:

``` 
gcloud auth configure-docker 
```
2. Tag the image with a registry name:

``` 
docker tag test-worker gcr.io/ml-pipeline-test/test-worker:latest 
```
3. Push the image to Container Registry:

``` 
docker push gcr.io/ml-pipeline-test/test-worker:latest 
```