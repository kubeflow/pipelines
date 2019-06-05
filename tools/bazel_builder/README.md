This directory contains Dockerfile and Bazel configuration to be used by the
Remote Build Execution service for running Bazel workloads in GCP.

The build is run using a custom image that contains both Bazel as well as
`cmake` (which is required by the ML Metadata dependency used in API Server).

The custom image is pushed to
`gcr.io/ml-pipeline-test/bazel-builder@sha256:<IMAGE_SHA>"`

To build a new image, use the `Dockerfile` in this folder: `docker build -f
Dockerfile .`

Take note of the SHA id of the image, and update the value of
`CONTAINER_VERSION` in the `BUILD` file in this directory.

For more information about RBE, see their
[documentation](https://cloud.google.com/sdk/gcloud/reference/alpha/remote-build-execution/).
