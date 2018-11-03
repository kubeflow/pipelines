[![Build Status](https://travis-ci.com/kubeflow/pipelines.svg?token=JjfzFsYGxZwkHvXFCpwt&branch=master)](https://travis-ci.com/kubeflow/pipelines)

# ML Pipeline Services - Overview

This project is aimed at:

* End to end orchestration: enabling and simplifying the orchestration of end to end machine learning pipelines
* Easy Experimentation: making it easy for users to try numerous ideas and techniques, and manage the various trials/experiments.
* Easy Re-Use: enabling users to re-use components and pipelines to quickly cobble together end to end solutions, without having to re-build each time.

## Introduction

### The Python Code to Represent a Pipeline Workflow Graph

```python

@mlp.pipeline(
  name='XGBoost Trainer',
  description='A trainer that does end-to-end distributed training for XGBoost models.'
)
def xgb_train_pipeline(
    output,
    project,
    region=PipelineParam(value='us-central1'),
    train_data=PipelineParam(value='gs://ml-pipeline-playground/sfpd/train.csv'),
    eval_data=PipelineParam(value='gs://ml-pipeline-playground/sfpd/eval.csv'),
    schema=PipelineParam(value='gs://ml-pipeline-playground/sfpd/schema.json'),
    target=PipelineParam(value='resolution'),
    rounds=PipelineParam(value=200),
    workers=PipelineParam(value=2),
):
  delete_cluster_op = DeleteClusterOp('delete-cluster', project, region)
  with mlp.ExitHandler(exit_op=delete_cluster_op):
    create_cluster_op = CreateClusterOp('create-cluster', project, region, output)

    analyze_op = AnalyzeOp('analyze', project, region, create_cluster_op.output, schema,
                           train_data, '%s/{{workflow.name}}/analysis' % output)

    transform_op = TransformOp('transform', project, region, create_cluster_op.output,
                               train_data, eval_data, target, analyze_op.output,
                               '%s/{{workflow.name}}/transform' % output)

    train_op = TrainerOp('train', project, region, create_cluster_op.output, transform_op.outputs['train'],
                         transform_op.outputs['eval'], target, analyze_op.output, workers,
                         rounds, '%s/{{workflow.name}}/model' % output)

    predict_op = PredictOp('predict', project, region, create_cluster_op.output, transform_op.outputs['eval'],
                           train_op.output, target, analyze_op.output, '%s/{{workflow.name}}/predict' % output)

    confusion_matrix_op = ConfusionMatrixOp('confusion-matrix', predict_op.output,
                                            '%s/{{workflow.name}}/confusionmatrix' % output)

    roc_op = RocOp('roc', predict_op.output, '%s/{{workflow.name}}/roc' % output)

```

### The Above Pipeline After Being Uploaded

<kbd>
  <img src="docs/images/job.png">
</kbd>

### The Run Time Execution Graph of the Pipeline Above

<kbd>
  <img src="docs/images/graph.png">
</kbd>

### Outputs From The Pipeline

<kbd>
  <img src="docs/images/output.png">
</kbd>

# ML Pipeline Services - User Guideline

This document guides you through the steps to deploy Machine Learning Pipeline Services and run your first pipeline sample. 

## Requirements

The guideline assume you have a [GCP Project](https://cloud.google.com/resource-manager/docs/creating-managing-projects) ready. You can use [Cloud Shell](https://cloud.google.com/shell/docs/quickstart) to run all the commands in this guideline. 

Alternatively, if you prefer to install and interact with GKE from your local machine, make sure you have [gcloud CLI](https://cloud.google.com/sdk/) and [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#download-as-part-of-the-google-cloud-sdk) installed locally.
 
## Setup a GKE cluster

Follow the [instruction](https://cloud.google.com/resource-manager/docs/creating-managing-projects) and create a GCP project. 
Once created, enable the GKE API in this [page](https://console.developers.google.com/apis/enabled). You can also find more details about enabling the [billing](https://cloud.google.com/billing/docs/how-to/modify-project?#enable-billing), as well as activating [GKE API](https://cloud.google.com/kubernetes-engine/docs/quickstart#before-you-begin).

We recommend to use CloudShell from the GCP console to run the below commands. CloudShell starts with an environment already logged in to your account and set to the currently selected project. The following two commands are required only in a workstation shell environment, they are not needed in the CloudShell. 

```bash
gcloud auth login
gcloud config set project [your-project-id]
```

The ML Pipeline Services will be running on a GKE cluster. To start a new GKE cluster, first set a default compute zone (us-central1-a in this case):
```bash
gcloud config set compute/zone us-central1-a
```
Then start a GKE cluster. 
```bash
# Specify your cluster name
CLUSTER_NAME=[YOUR-CLUSTER-NAME]
gcloud container clusters create $CLUSTER_NAME \
  --zone us-central1-a \
  --scopes cloud-platform \
  --enable-cloud-logging \
  --enable-cloud-monitoring \
  --machine-type n1-standard-2 \
  --num-nodes 4
```
Here we choose the cloud-platform scope so the cluster can invoke GCP APIs. You can find all the options for creating a cluster in [here](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create). 

Next, grant your user account permission to create new cluster roles. This step is necessary because installing ML Pipelines Services inlcudes installing a few [clusterroles](https://github.com/kubeflow/pipelines/search?utf8=%E2%9C%93&q=clusterrole+path%3Aml-pipeline%2Fml-pipeline&type=). 
```bash
kubectl create clusterrolebinding ml-pipeline-admin-binding --clusterrole=cluster-admin --user=$(gcloud config get-value account)
```
 
## Deploy ML Pipeline Services and Kubeflow 

Go to [release page](https://github.com/kubeflow/pipelines/releases) to find a version of ML Pipeline Services. Deploy the ML Pipeline Services and Kubeflow to your cluster.

For example:
```bash
PIPELINE_VERSION=0.0.26
kubectl create -f https://storage.googleapis.com/ml-pipeline/release/$PIPELINE_VERSION/bootstrapper.yaml
```
And by running `kubectl get job`, you should see a job created that deploys ML pipeline along with all dependencies in the cluster.
```
NAME                      DESIRED   SUCCESSFUL   AGE
deploy-ml-pipeline-wjqwt  1         1            9m
```
You can check the deployment log in case of any failure
```bash
kubectl logs $(kubectl get pods -l job-name=[JOB_NAME] -o jsonpath='{.items[0].metadata.name}')
```

By default, the ML pipeline will deployed with usage collection turned on. 
We use [Spartakus](https://github.com/kubernetes-incubator/spartakus) which does not report any personal identifiable information (PII).

If you want to turn off the usage report, you can download the bootstrapper file and change the arguments to the deployment job.

For example, download bootstrapper
```bash
PIPELINE_VERSION=0.0.26
curl https://storage.googleapis.com/ml-pipeline/release/$PIPELINE_VERSION/bootstrapper.yaml --output bootstrapper.yaml
```
and then update argument in the file
```
        args: [
          ... 
          # uncomment following line
          "--report_usage", "false",
          ...
        ]
```
then create job using the updated YAML by running ```kubectl create -f bootstrapper.yaml```

When deployment is successful, forward its port to visit the ML pipeline UI. 
```bash
kubectl port-forward -n kubeflow `kubectl get pods -n kubeflow --selector=service=ambassador -o jsonpath='{.items[0].metadata.name}'` 8080:80
```
If you are using Cloud Shell, you could view the UI by open the [web preview](https://cloud.google.com/shell/docs/using-web-preview#previewing_the_application) button ![alt text](https://cloud.google.com/shell/docs/images/web-preview-button.png). Make sure the it's preview on port 8080.

If you are using local console instead of Cloud Shell, you can access the ML pipeline UI at [localhost:8080/pipeline](http://localhost:8080/pipeline).

## Run your first TFJob pipeline
See the following authoring guide on how to compile your python pipeline code into workflow tar file. Then, follow [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/kubeflow-tf/README.md) to deploy your first TFJob pipeline.  

## Uninstall
To uninstall ML pipeline, download the bootstrapper file and change the arguments to the deployment job.

For example, download bootstrapper
```bash
PIPELINE_VERSION=0.0.26
curl https://storage.googleapis.com/ml-pipeline/release/$PIPELINE_VERSION/bootstrapper.yaml --output bootstrapper.yaml
```
and then update argument in the file
```
        args: [
          ... 
          # uncomment following line
          "--uninstall",
          ...
        ]
```
then create job using the updated YAML by running ```kubectl create -f bootstrapper.yaml```

# ML Pipeline Services - Authoring Guideline

For more details, see [README.md](https://github.com/kubeflow/pipelines/blob/master/samples/README.md).

## Setup
* Create a python3 environment.
 
* Install a version of pipeline SDK:
```bash
pip install https://storage.googleapis.com/ml-pipeline/release/0.0.26/kfp-0.0.26.tar.gz --upgrade
```

Note: if you prefer adding "--user" in installation of dsl-compiler, please also run "export PATH=~/.local/bin:$PATH".

After successful installation the command "dsl-compile" should be added to your PATH.

## Compile the samples
The pipelines are written in Python, but they must be compiled to an intermediate representation before submitting to the Pipeline system.

The sample pipelines have a self-compilation entry point. To compile the sample pipelines just run them:
```bash
path/sample_pipeline.py
```
This produces the some_pipeline.py.tar.gz file that can be submitted to the Pipeline system web UI.

Another way to compile a pipeline is to run the DSL compiler against it:
```bash
dsl-compile --py [path/to/py/file] --output [path/to/output/tar.gz]
```

For example:

```bash
cd $EXTRACTED_DIRECTORY
dsl-compile --py ./samples/basic/sequential.py --output ./samples/basic/sequential.tar.gz
```

## Deploy the samples
Upload the generated tar file through the ML pipeline UI.
