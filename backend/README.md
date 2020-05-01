This directory contains code for the components that comprise the Kubeflow
Pipelines backend.

## Building & Testing

All components can be built using [Bazel](https://bazel.build/). Note, 
make sure to install version `0.23.0` as newer versions will break. To build
everything under backend, run: 
```bash
bazel build --action_env=PATH --define=grpc_no_ares=true //backend/...
```

To run all tests: 
```bash
bazel test --action_env=PATH --define=grpc_no_ares=true //backend/...
```

The API server itself can only be built/tested using Bazel. The following
commands target building and testing just the API server. 
```bash
bazel build --action_env=PATH --define=grpc_no_ares=true backend/src/apiserver/...
bazel test --action_env=PATH --define=grpc_no_ares=true backend/src/apiserver/...
```

## Building APIServer Image using Remote Build Execution

If you are a dev in the Kubeflow Pipelines team, you can use
[Remote Build Execution Service](https://cloud.google.com/sdk/gcloud/reference/alpha/remote-build-execution/)
to build the API Server image using Bazel with use of a shared cache for
speeding up the build. To do so, execute the following command:

```
./build_api_server.sh -i gcr.io/cloud-ml-pipelines-test/api-server:dev
```

## Building Go client library and swagger files

After making changes to proto files, the Go client libraries and swagger files
need to be regenerated and checked-in. The backend/api/generate_api.sh script
takes care of this.

## Updating BUILD files

As the backend is written in Go, the BUILD files can be updated automatically
using [Gazelle](https://github.com/bazelbuild/bazel-gazelle). Whenever a Go file
is added or updated, run the following to ensure the BUILD files are updated as
well: 
```bash
bazel run //:gazelle
```

If a new external Go dependency is added, or an existing one has its version
bumped in the `go.mod` file, ensure the BUILD files pick this up by updating the
WORKSPACE go_repository rules using the following command: 
```bash
bazel run //:gazelle -- update-repos --from_file=go.mod
```

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `./update_requirements.sh <requirements.in >requirements.txt` to update and pin the transitive
dependencies.

