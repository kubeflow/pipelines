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

## Building Go client library and swagger files

After making changes to proto files, the Go client libraries and swagger files
need to be regenerated and checked-in. The `backend/api/generate_api.sh` script
takes care of this. It should be noted that this requires [Bazel](https://bazel.build/), version 0.24.0` 

# Visualization Server Instructions

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `./update_requirements.sh <requirements.in >requirements.txt` to update and pin the transitive
dependencies.
