# Kubernetes platform-specific feature

Contains proto sources and tools for generating Go and Python bindings. Python
library code lives in `sdk/python/kfp/kubernetes` and ships with `kfp`.

## Dependencies
You need to have `protoc` installed. You can find the releases [here](https://github.com/protocolbuffers/protobuf/releases).

Use the repository's generator image (or the protoc version pinned in
`.github/actions/protobuf/action.yml`) and the protobuf runtime constrained by
`sdk/python/pyproject.toml`.

## Generate Python proto code (alongside non-proto library code)
Run `make clean-python python` after changing the schema and commit the generated
`sdk/python/kfp/kubernetes/kubernetes_executor_config_pb2.py`. CI regenerates it
and rejects stale output. There is no separate Kubernetes package version.
From the repository root, `make -C sdk python` generates all Python bindings and
builds the unified SDK. Tests remain in `sdk/python/test/kubernetes`.

## Generate Go proto code
Go proto code should be updated when the `kubernetes_executor_config.proto` file is updated.

Go proto code *should* be checked into source control.

```bash
make clean-go golang
```
