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
cd ${REPO_ROOT}
# if you have a sample test at samples/path/to/your/sample_test.py
python -m samples.path.to.your.sample_test --host https://your.KFP.host --output_directory gs://your-bucket/path/to/output/dir
# or to look at command help
python -m samples.path.to.your.sample_test --help
```

## How to add a sample to this sample test?

Edit [samples/test/config.yaml](../../samples/test/config.yaml) and add your own sample.
You can also add other samples not in the `samples/test` folder.

Your sample test needs to conform to the standard interface in
[components/run_sample.yaml](components/run_sample.yaml). You can refer to
existing [sample tests](../../samples/test) for how to implement the interface.

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
