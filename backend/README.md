This directory contains code for the components that comprise the Kubeflow
Pipelines backend.

## Building & Testing

To run all unittests for backend:

```
go test -v -cover ./backend/...
```

The API server itself can be built using:

```
go build -o /tmp/apiserver backend/src/apiserver/*.go
```

## Code Style

Backend codebase follows the [Google's Go Style Guide](https://google.github.io/styleguide/go/). Please, take time to get familiar with the [best practices](https://google.github.io/styleguide/go/best-practices). It is not intended to be exhaustive, but it often helps minimizing guesswork among developers and keep codebase uniform and consistent.

We use [golangci-lint](https://golangci-lint.run/) tool that can catch common mistakes locally (see detailed configuration [here](https://github.com/kubeflow/pipelines/blob/master/.golangci.yaml)). It can be [conveniently integrated](https://golangci-lint.run/usage/integrations/) with multiple popular IDEs such as VS Code or Vim.

Finally, it is advised to install [pre-commit](https://pre-commit.com/) in order to automate linter checks (see configuration [here](https://github.com/kubeflow/pipelines/blob/master/.pre-commit-config.yaml))

## Building APIServer image locally

The API server image can be built from the root folder of the repo using:
```
export API_SERVER_IMAGE=api_server
docker build -f backend/Dockerfile . --tag $API_SERVER_IMAGE
```
## Deploy APIServer with the image you own build

Run
```
kubectl edit deployment.v1.apps/ml-pipeline -n kubeflow
```
You'll see the field reference the api server docker image.
Change it to point to your own build, after saving and closing the file, apiserver will restart with your change.

## Building client library and swagger files

After making changes to proto files, the Go client libraries, Python client libraries and swagger files
need to be regenerated and checked-in. Refer to [backend/api](./api/README.md) for details.

## Updating licenses info

1. [Install go-licenses tool](../hack/install-go-licenses.sh) and refer to [its documentation](https://github.com/google/go-licenses) for how to use it.


2. Run the tool to update all licenses:

    ```bash
    make all
    ```

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `./update_requirements.sh` to update and pin the transitive
dependencies.

# Visualization Server Instructions

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `./update_requirements.sh` to update and pin the transitive
dependencies.


## Building conformance tests (WIP)

Run
```
docker build . -f backend/Dockerfile.conformance -t <tag>
```
