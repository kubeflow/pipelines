# ML pipeline test infrastructure

This folder contains the integration/e2e tests for ML pipeline. We use Argo workflow to run the tests.

At a high level, a typical test workflow will

- build docker images for all components
- create a dedicate test namespace in the cluster 
- deploy ml pipeline using the newly built components
- run the test
- delete the namespace
- delete the images

All these steps will be taking place in the same Kubernetes cluster. 
You can use GKE to test against the code in a Github Branch. The image will be temporarily stored in the GCR repository in the same project.

Tests are run automatically on each commit in a Kubernetes cluster using
[Prow](https://github.com/kubernetes/test-infra/tree/master/prow).
Tests can also be run manually, see the next section.

## Run tests using GKE

You could run the tests against a specific commit.

### Setup

Here are the one-time steps to prepare for your GKE testing cluster:
- Follow the [main page](https://github.com/kubeflow/pipelines#setup-gke) to
create a GKE cluster.
- Install [Argo](https://github.com/argoproj/argo/blob/master/demo.md#argo-v20-getting-started)
in the cluster.
- Create cluster role binding.
  ```
  kubectl create clusterrolebinding default-as-admin --clusterrole=cluster-admin --serviceaccount=default:default
  ```
- Follow the
[guideline](https://developer.github.com/v3/guides/managing-deploy-keys/) to
create a
[ssh](https://help.github.com/articles/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent/)
deploy key, and store as Kubernetes secret in your cluster, so the job can
later access the code. Note it requires admin permission to add a deploy key
to github repo. This step is not needed when the project is public.
  ```
  kubectl create secret generic ssh-key-secret
  --from-file=id_rsa=/path/to/your/id_rsa
  --from-file=id_rsa.pub=/path/to/your/id_rsa.pub
  ```

### Run tests
Simply submit the test workflow to the GKE cluster, with a parameter
specifying the commit you want to test (master HEAD by default):
```
argo submit integration_test_gke.yaml -p commit-sha=<commit>
```
You can check the result by doing:
```
argo list
```
The workflow will create a temporary namespace with the same name as the Argo
workflow. All the images will be stored in
**gcr.io/project_id/workflow_name/branch_name/***. By default when the test is
*finished, the namespace and images will be deleted.
However you can keep them by providing additional parameter. 
```
argo submit integration_test_gke.yaml -p branch="my-branch" -p cleanup="false"
```

### Run presubmit-tests-with-pipeline-deployment.sh locally

Run the following commands from root of kubeflow/pipelines repo.
```
# $WORKSPACE are env variables set by Prow
export WORKSPACE=$(pwd) # root of kubeflow/pipelines git repo
export SA_KEY_FILE=PATH/TO/YOUR/GCP/PROJECT/SERVICE/ACCOUNT/KEY
# (optional) uncomment the following to keep reusing the same cluster
# export TEST_CLUSTER=YOUR_PRECONFIGURED_CLUSTER_NAME
# (optional) uncomment the following to disable built image caching
# export DISABLE_IMAGE_CACHING=true

./test/presubmit-tests-with-pipeline-deployment.sh \
  --workflow_file e2e_test_gke_v2.yaml \ # You can specify other workflows you want to test too.
  --test_result_folder ${FOLDER_NAME_TO_HOLD_TEST_RESULT} \
  --test_result_bucket ${YOUR_GCS_TEST_RESULT_BUCKET} \
  --project ${YOUR_GCS_PROJECT}
```

### Run multiuser_tests.sh locally

KFP internal team members can run the test in `ml-pipeline-test` project.
OAuth client, test profiles, etc. have already been prepared in this project.
You can run the following commands from root of kubeflow/pipelines repo.
```
export WORKSPACE=$(pwd) # root of kubeflow/pipelines git repo
export SA_KEY_FILE=PATH/TO/YOUR/GCP/PROJECT/SERVICE/ACCOUNT/KEY

./test/multiuser-tests.sh --workflow_file multiuser_test.yaml
```

For external users, or if you want to run the test in a different project, complete the pre-requisites first:
1. Follow this [instruction](https://www.kubeflow.org/docs/gke/deploy/oauth-setup/) to set up OAuth for Cloud IAP.
1. Modify the template [profile.yaml](./multi-user-test/profile.yaml) file with your choice of namespaces and user emails.
1. Get the refresh tokens for the test user accounts using this [helper method](https://github.com/kubeflow/pipelines/blob/587292fbaedf0a51694770bc167f9f1e7a1e3e82/sdk/python/kfp/_auth.py#L168). An easier alternative is to initiate an SDK client, and then find the refresh token from the [file](https://github.com/kubeflow/pipelines/blob/587292fbaedf0a51694770bc167f9f1e7a1e3e82/sdk/python/kfp/_auth.py#L32).

Once completing the above steps, you can run the following commands to trigger the test.
```
export WORKSPACE=$(pwd) # root of kubeflow/pipelines git repo
export SA_KEY_FILE=PATH/TO/YOUR/GCP/PROJECT/SERVICE/ACCOUNT/KEY

export CLIENT_ID=<OAuth client ID>
export CLIENT_SECRET=<secret for the OAuth client>
export OTHER_CLIENT_ID=<other client ID>
export OTHER_CLIENT_SECRET=<secret for other client ID>
export REFRESH_TOKEN_A=<refresh token for user A>
export USER_NAMESPACE_A=<namespace for user A>
export REFRESH_TOKEN_B=<refresh token for user B>
export USER_NAMESPACE_B=<namespace for user B>

./test/multiuser-tests.sh --workflow_file multiuser_test.yaml --project <your project> --test_result_bucket <your gcs bucket name>
```

## Troubleshooting

**Q: Why is my test taking so long on GKE?**

The cluster downloads a bunch of images during the first time the test runs. It will be faster the second time since the images are cached.
The image building steps are running in parallel and usually takes 2~3 minutes in total. If you are experiencing high latency, it might due to the resource constrains
on your GKE cluster. In that case you need to deploy a larger cluster. 
