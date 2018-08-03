[![Build Status](https://travis-ci.com/googleprivate/ml.svg?token=JjfzFsYGxZwkHvXFCpwt&branch=master)](https://travis-ci.com/googleprivate/ml)

# ML Pipeline User Guideline

This document guides you through the steps to install Machine Learning Pipelines and run your first pipeline sample. 

## Requirements
- Install [gcloud CLI](https://cloud.google.com/sdk/) and [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#download-as-part-of-the-google-cloud-sdk)
- Install [git client](https://git-scm.com/downloads)
- Install [conda](https://conda.io/docs/user-guide/install/download.html)
- Install [docker client](https://www.docker.com/get-docker )
- [Setup a GKE cluster](#setup-gke)

## Install

To install ML pipeline, run following command 
```
$ kubectl create -f https://storage.googleapis.com/ml-pipeline/release/latest/bootstrapper.yaml
```
And you should see a job created that deploys ML pipeline along with all dependencies in the cluster.
```
$ kubectl get job
NAME                      DESIRED   SUCCESSFUL   AGE
deploy-ml-pipeline-wjqwt  1         1            9m
```
You can check the deployment log in case of any failure
```
$ kubectl logs $(kubectl get pods -l job-name=[JOB_NAME] -o jsonpath='{.items[0].metadata.name}')
```

By default, the ML pipeline will deployed with usage collection turned on. 
We use [Spartakus](https://github.com/kubernetes-incubator/spartakus) which does not report any personal identifiable information (PII).

If you want to turn off the usage report, or deploy to a different namespace, you can pass in additional arguments to the deployment job.
See [bootstrapper.yaml](https://github.com/googleprivate/ml/blob/master/bootstrapper.yaml#L57) file for example arguments.


When deployment is successful, forward its port to visit the ML pipeline UI. 
```
$ kubectl port-forward $(kubectl get pods -l app=ml-pipeline-ui -o jsonpath='{.items[0].metadata.name}') 8080:3000
```
You can now access the ML pipeline UI at [localhost:8080](http://localhost:8080).

## Run your first TFJob pipeline
Clone pipelines repo
 
(note that you cannot clone a private repo to a laptop, please remote to your workstation and do it there)
```
git clone https://github.com/googleprivate/ml.git
```

Install DSL compiler by following [README.md](https://github.com/googleprivate/ml/blob/master/samples/basic/README.md). 

Create a GCS bucket to store the generated model. Make sure it's in the same project as the ML Pipeline deployed above.
```
gsutil mb -c nearline gs://[YOUR_GCS_BUCKET]
```

Compile [kubeflow-training-classification.py](https://github.com/googleprivate/ml/blob/master/samples/kubeflow-tf/kubeflow-training-classification.py)
and upload the generated yaml through the Pipeline UI. Fill up the following parameters
```
project: MY_GCP_PROJECT
output: gs://[YOUR_GCS_BUCKET]/flowermodel 
```

Run and check out the current status in the Web UI. 

Checkout [here](https://github.com/googleprivate/ml/blob/master/samples) for more samples. 

## Uninstall
To uninstall ML pipeline, create a job following the same steps as installation, with additional uninstall argument. 
Check [bootstrapper.yaml](https://storage.googleapis.com/ml-pipeline/bootstrapper.yaml) for details.

## Appendix

### Setup GKE

Follow the [instruction](https://cloud.google.com/resource-manager/docs/creating-managing-projects) and create a GCP project. 
Once created, enable the GKE API in this [page](https://console.developers.google.com/apis/enabled). You can also find more details about enabling the [billing](https://cloud.google.com/billing/docs/how-to/modify-project?visit_id=1-636559671979777487-508867449&rd=1#enable-billing), as well as activating [GKE API](https://cloud.google.com/kubernetes-engine/docs/quickstart#before-you-begin).

```
$ gcloud auth login
$ gcloud config set project [your-project-id]
```

The ML pipeline will be running on a GKE cluster. To start a new GKE cluster, first set a default compute zone (us-central1-a in this case):
```
$ gcloud config set compute/zone us-central1-a
```
Then start a GKE cluster. 
```
$ gcloud container clusters create ml-pipeline \
  --zone us-central1-a \
  --scopes cloud-platform \
  --enable-cloud-logging \
  --enable-cloud-monitoring \
  --machine-type n1-standard-2 \
  --num-nodes 4
```
Here we choose cloud-platform scope so it can invoke GCP APIs. You can find all options for creating a cluster in [here](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create). 

ML Pipeline uses Argo as its underlying pipeline orchestration system. When installing ML pipeline, a few new cluster roles will be generated for Argo and Pipeline’s API server, in order to allow them creating and retrieving resources such as pods, Argo workflow CRDs etc. 

On GKE with RBAC enabled, you might need to grant your account the ability to create such new cluster roles.

```
$ kubectl create clusterrolebinding BINDING_NAME --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
```

Now the GKE cluster is ready to use.
