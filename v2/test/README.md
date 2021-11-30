# Kubeflow Pipelines Sample Test Infra V2

Note, the sample test only runs on Google Cloud at the moment. Welcome
contribution if you want to adapt it to other platforms.

Quick Links:

* [prowjob config](https://github.com/GoogleCloudPlatform/oss-test-infra/blob/48b09567c8df28fab2d3f2fb6df86defa12207fb/prow/prowjobs/kubeflow/pipelines/kubeflow-pipelines-presubmits.yaml#L184-L192)
* [past prow jobs](https://oss-prow.knative.dev/job-history/gs/oss-prow/pr-logs/directory/kubeflow-pipelines-samples-v2)
* [sample test config](../../samples/test/config.yaml)
* [KFP test cluster hostname](https://github.com/kubeflow/testing/blob/master/test-infra/kfp/endpoint)
* [Infra as Code configuration for kfp-ci project](https://github.com/kubeflow/testing/tree/master/test-infra/kfp).

## How to access the KFP UI running these tests?

kubeflow-pipelines-sample-v2 test pipeline runs on [kfp-standalone-1 cluster](https://console.cloud.google.com/kubernetes/clusters/details/us-central1/kfp-standalone-1/details?folder=&organizationId=&project=kfp-ci),
`kfp-ci` project, `kubeflow.org` organization.

The test script prints KFP host URL in logs. You need to have permission to
access it.

You need to join [Kubeflow ci-team google group](https://github.com/kubeflow/internal-acls/blob/master/google_groups/groups/ci-team.yaml) to get edit access to the project. The group
has very wide permissions to test infra, so access will only be granted to core
developers.

Currently, it's not possible to grant KFP UI only permission, but we can grant
such access to [Kubeflow ci-viewer google group](https://github.com/kubeflow/internal-acls/blob/master/google_groups/groups/ci-viewer.yaml).
Contact @Bobgy if you have such a need.

## How to run the entire sample test suite in your own KFP?

You need to create an `.env` file in this folder and add the following config:

```env
GCS_ROOT=gs://path/to/sample/test/workingdir
GCR_ROOT=gcr.io/path/to/sample/test/container/registry
HOST=https://your.kfp.hostname.com
```

You need to login locally to allow uploading source folder to the GCS_ROOT:

```bash
gcloud auth application-default login
# Or use the following to login both gcloud and application default
# at the same time.
gcloud auth login --update-adc
```

Your KFP cluster should have permission for the GCS_ROOT and the GCR_ROOT.

Run sample test by:

```bash
make sample-test
```

Note, there's one caveat, for any files not tracked by git, they will not be uploaded.
So recommend doing a `git add -A` before running this if you added new files. However,
it's OK to have dirty files, the dirty version in your workdir will be uploaded
as expected.

For why the caveat exists, refer to context rule in [Makefile](./Makefile).

## How to develop one single sample?

```bash
# These env vars are loaded by default, recommend configuring them in your
# .bashrc or .zshrc
export KFP_HOST=https://your.KFP.host
export KFP_PIPELINE_ROOT=gs://your-bucket/path/to/output/dir
export METADATA_GRPC_SERVICE_HOST=localhost
export PATH="$HOME/bin:$PATH" # Some CLI tools will be installed to ~/bin.
# optional, when you want to override images to your dev project
# export KFP_LAUNCHER_V2_IMAGE=gcr.io/your-project/dev/kfp-launcher-v2:latest
# export KFP_DRIVER_IMAGE=gcr.io/your-project/kfp-driver:latest

# optional, when you want to override pipeline root
# export KFP_PIPELINE_ROOT="gs://your-bucket/your-folder"

# optional, when you need to override which KFP python package v2 components use:
# export KFP_PACKAGE_PATH=git+https://github.com/kubeflow/pipelines#egg=kfp&subdirectory=sdk/python

cd "${REPO_ROOT}/v2"
# Installs kfp-v2-compiler as a CLI tool to ~/bin
# Note, when you update backend compiler code, you need to run this again!
make install-compiler

# Note, for tests that use metadata grpc api, you should port-forward it locally in a separate terminal by:
cd "${REPO_ROOT}/v2/test"
make mlmd-port-forward

# To run a single sample test:
cd "${REPO_ROOT}"
# if you have a sample test at samples/path/to/your/sample_test.py
python -m samples.path.to.your.sample_test
# or to look at command help
python -m samples.path.to.your.sample_test --help
```

## How to add a sample to this sample test?

Edit [samples/test/config.yaml](../../samples/test/config.yaml) and add your own sample.
You can also add other samples not in the `samples/test` folder.

Your sample test needs to conform to the standard interface in
[components/run_sample.yaml](components/run_sample.yaml). You can refer to
existing [sample tests](../../samples/test) for how to implement the interface.

## How can a sample verify MLMD status of a run?

Refer to [an existing test](../../samples/v2/hello_world_test.py).

## FAQs

1. Q: I'm getting error `main.go:56] Failed to execute component: unable to get pipeline with PipelineName "pipeline-with-lightweight-io" PipelineRunID "pipeline-with-lightweight-io-pmxzr": Failed PutParentContexts(parent_contexts:{child_id:174  parent_id:173}): rpc error: code = Unimplemented desc =`.

   A: You need to upgrade metadata-grpc-service deployment to 1.0.0+. KFP manifest master branch includes the upgrade, but it hasn't been released yet. Therefore, you need to install KFP standalone from master: `kubectl apply -k manifests/kustomize/env/dev`.

## Implementation Details

When kubeflow-pipelines-samples-v2 test is called from presubmit, it goes through
the following steps:

1. configure env
2. package source folder into a tarball and upload it to Cloud Storage as input to the test pipeline
3. use KFP sdk to compile, create and wait for a test orchestration KFP pipeline
4. The test orchestration pipeline
   1. builds needed images
   2. compiles, creates and waits for sub sample KFP pipelines
   3. verifies execution result of sample pipelines
