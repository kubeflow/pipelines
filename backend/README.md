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

Use [gofmt](https://pkg.go.dev/cmd/gofmt) package to format your .go source files. There is no need to format the swagger generated go clients, so only run the following command in `./backend/src` and `./backend/test` folder.

```
go fmt ./...
```

For more information, see [this blog](https://go.dev/blog/gofmt).

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
