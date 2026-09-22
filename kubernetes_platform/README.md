# Kubernetes platform-specific feature

Contains protos, non-proto Python library code, and tools for generating Go and Python proto library code.

## Dependencies
You need to have `protoc` installed. You can find the releases [here](https://github.com/protocolbuffers/protobuf/releases).

If you get an error `ImportError: cannot import name 'runtime_version' from 'google.protobuf'` when trying to import kfp-kubernetes that you've built with protoc, you're using a protoc that's too new. Version 21.12 is known to work for compiling the python protos. Version 27.1 is known to produce this error.

## Generate Python proto code (alongside non-proto library code)
Python proto code should be updated prior to releasing the package. For this reason, bumping the version number is included in the instructions.

Python proto code *should not* be checked into source control.

1. Update the version in `python/kfp/kubernetes/__init__.py` and the matching SDK dependency pins in the workspace `pyproject.toml` files if applicable, then run `uv lock` from the repository root.
2. `make clean-python python`

The Python package is built with `uv build` using the Hatch metadata in `python/pyproject.toml`.

## Generate Go proto code
Go proto code should be updated when the `kubernetes_executor_config.proto` file is updated.

Go proto code *should* be checked into source control.

```bash
make clean-go golang
```
