# API server integration test

This folder contains integration tests for API server. The integration test are executed by an Argo workflow.

The workflow will build an image, and deploy the ML pipeline using the image. You can run the test either on Minikube or GKE.

## Run test using Minikube

### tl;dr

To run the test locally on Minikube 
- Install [Argo](https://github.com/argoproj/argo/blob/master/demo.md#argo-v20-getting-started) if you haven't done so.
- Create an admin role in the minikube cluster, and bind to default service account, so that integration test can create additional roles as part of the deployment step.
```
kubectl create -f kubetool/cluster_admin.yaml
```
- Modify the bottom of the integration_test_minikube.yaml file to replace **repo-dir** with your local github repo root dir.
TODO(yangpa): Argo will support parameterization on Volume in v2.1 [Bug](https://github.com/argoproj/argo/issues/822). We should switch to use parameter when available.
- Simply run the workflow with a namespace the test to run in. **ml-pipeline-test** by default
```
argo submit integration_test_minikube.yaml
```
- You can check the result by 
```
argo list
```

### What actually happens
The integration test workflow consists of following steps
1. **Create a test namespace.**
The bootstrapper image has kubectl pre-installed, to talk to the minikube master.

2. **Build API server image using your local code.**
The docker image is mounted with your local docker socket so it can access your local docker daemon.

3. **Deploy ML pipeline.**
We use same bootstrapper image to deploy ml pipeline. 

4. **Run test.** 
The test is running against a Go image with your local test code mounted.

5. **Clean up resource.**
The test namespace is deleted in this step.