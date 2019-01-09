This directory contains code for the components that comprise the Kubeflow
Pipelines backend.

## Building & Testing
All components can be built using [Bazel](https://bazel.build/). To build
everything under bazel, run:
```
bazel build //backend/...
```

To run all tests:
```
bazel test //backend/...
```

## Updating BUILD files
As the backend is written in Go, the BUILD files can be updated automatically
using [Gazelle](https://github.com/bazelbuild/bazel-gazelle). Whenever a Go
file is added or updated, run the following to ensure the BUILD files are
updated as well:
```
bazel run //:gazelle
```

If a new external Go dependency is added, or an existing one has its version
bumped in the `go.mod` file, ensure the BUILD files pick this up by updating
the WORKSPACE go_repository rules using the following command:
```
bazel run //:gazelle -- update-repos --from_file=go.mod
```