# Kubernetes platform-specific feature

Contains protos, non-proto Python library code, and tools for generating Go and Python proto library code.

## Generate Python proto code (alongside non-proto library code)
Python proto code should be updated prior to releasing the package. For this reason, bumping the version number is included in the instructions.

Python proto code *should not* be checked into source control.

1. Update version in `python/setup.py` if applicable.
2. `make clean-python python`

If you get an error `error: invalid command 'bdist_wheel'`, run `pip install wheel` then try again.

## Generate Go proto code
Go proto code should be updated when the `kubernetes_executor_config.proto` file is updated.

Go proto code *should* be checked into source control.

```bash
make clean-go golang
```
