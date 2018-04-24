# ML pipeline test infrastructure

This folder contains the integration/e2e tests for ML pipeline. We use Argo workflow to run the tests.

At a high level, a typical test workflow will

- build docker images for all components
- create a dedicate test namespace in the cluster 
- deploy ml pipeline using the newly built components
- run the test
- delete the namespace
- delete the images

All these steps will be taken place in the same Kubernetes cluster. 
We support running tests on both GKE and Minikube, targeting on slightly different development scenario:
- You can use GKE to test against the code in a Github Branch. The image will be temporarily stored in the GCR repository in the same project.
- Minikube can test your local code and doesn't require a GCR to host the intermediate image. 
It also mounts your local docker socket so the build cache is persisted. The docker images is faster. 
This is a better option if you use Minikube as your development environment.

Currently we support running the tests manually. In the future, continuous integration using Kubernetes [Prow](https://github.com/kubernetes/test-infra/tree/master/prow) will also be supported. [Issue](https://github.com/googleprivate/ml/issues/302)

## Run tests using GKE

You could run the tests against a Github branch.

### Setup

Here are the one-time steps to prepare for your GKE testing cluster
- Follow the [main page](https://github.com/googleprivate/ml#setup-gke) to create a GKE cluster. 
- Install [Argo](https://github.com/argoproj/argo/blob/master/demo.md#argo-v20-getting-started) in the cluster. If you have Argo CLI installed locally, just run 
  ```
  argo install
  ```
- Create cluster role binding.
  ```
  kubectl create clusterrolebinding default-as-admin --clusterrole=cluster-admin --serviceaccount default:default
  ```
- Follow the [guideline](https://developer.github.com/v3/guides/managing-deploy-keys/) to create a [ssh](https://help.github.com/articles/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent/) deploy key, and store as Kubernetes secret in your cluster, so the job can later access the code. Note it requires admin permission to add a deploy key to github repo. This step is not needed when the project is public.
  ```
  kubectl create secret generic ssh-key-secret --from-file=id_rsa=/path/to/your/id_rsa --from-file=id_rsa.pub=/path/to/your/id_rsa.pub
  ```

### Run tests
Simply submit the test workflow to the GKE cluster, with a parameter specifying the branch you want to test (master branch by default)  
```
argo submit integration_test_gke.yaml -p branch="my-branch"
```
You can check the result by 
```
argo list
```
The workflow will create a temporary namespace with the same name as the Argo workflow. All the images will be stored in
**gcr.io/project_id/workflow_name/branch_name/***. By default when the test is finished, the namespace and images will be deleted.
However you can keep them by providing additional parameter. 
```
argo submit integration_test_gke.yaml -p branch="my-branch" -p cleanup="false"
```

## Run tests using Minikube
You could run the tests against your local code.

### Setup

To run the test locally on Minikube 
- Install Argo (same step as in GKE)
- Create an admin role in the minikube cluster, and bind to default service account, so that integration test can create additional roles as part of the deployment step.
  ```
  kubectl create -f test/cluster_admin.yaml
  ```
- Modify the bottom of the integration_test_minikube.yaml file to replace **repo-dir** with your local github repo root dir.
TODO(yangpa): Argo will support parameterization on Volume in v2.1 [Bug](https://github.com/argoproj/argo/issues/822). We should switch to use parameter when available.

### Run tests
Simply submit the test workflow
```
argo submit integration_test_minikube.yaml
```

## Troubleshooting

**Q: Why is my test taking so long on GKE?**

The cluster downloads a bunch of images during the first time the test runs. It will be faster the second time since the images are cached.
The image building steps are running in parallel and usually takes 2~3 minutes in total. If you are experiencing high latency, it might due to the resource constrains
on your GKE cluster. In that case you need to deploy a larger cluster. 