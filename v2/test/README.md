# Kubeflow Pipelines Sample Test Infra V2

Note, the sample test only runs on Google Cloud at the moment. Welcome
contribution if you want to adapt it to other platforms.

## How to access the KFP UI running these tests?

kubeflow-pipelines-sample-v2 test pipeline runs on `kfp-standalone-1` cluster,
`kfp-ci` project, `kubeflow.org` organization.

The test script prints KFP host URL in logs. You need to have permission to
access it.

You need to join [Kubeflow ci-team google group](https://github.com/kubeflow/internal-acls/blob/master/google_groups/groups/ci-team.yaml) to get edit access to the project. The group
has very wide permissions to test infra, so access will only be granted to core
developers.

Currently, it's not possible to grant KFP UI only permission, but we can grant
such access to [Kubeflow ci-viewer google group](https://github.com/kubeflow/internal-acls/blob/master/google_groups/groups/ci-viewer.yaml).
Contact @Bobgy if you have such a need.

## How to run sample test in your own KFP?

You need to create an `.env` file in this folder and add the following config to
it:

```env
GCS_ROOT=gs://path/to/sample/test/workingdir
GCR_ROOT=gcr.io/path/to/sample/test/container/registry
HOST=https://<your KFP host>
```

You need to login locally to allow uploading source folder to the GCS_ROOT:

```bash
gcloud auth application-default login
```

Your KFP cluster should have permission the GCS_ROOT and the GCR_ROOT.

Run sample test by:

```bash
make sample-test
```

## How to add a sample to this test infra?

Edit [samples_config.yaml](./samples_config.yaml) and add your own sample.
You can also add samples not in the `v2` folder.

Your sample needs to conform to the standard interface in
[components/run_sample.yaml](components/run_sample.yaml). You can refer to
existing [samples](samples) for how to implement the interface.

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

## How is test infra set up?

* [Prow test config for this presubmit test](https://github.com/GoogleCloudPlatform/oss-test-infra/blob/a4dda24bcc0afc811c4cda9671c2bc48da499cef/prow/prowjobs/kubeflow/pipelines/kubeflow-pipelines-presubmits.yaml#L184-L192).
* [Infra as Code configuration for kfp-ci project](https://github.com/kubeflow/testing/tree/master/test-infra/kfp).
