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

## API Server Development

### Run Locally With a Kind Cluster

This deploys a local Kubernetes cluster leveraging [kind](https://kind.sigs.k8s.io/), with all the components required
to run the Kubeflow Pipelines API server. Note that the `ml-pipeline` `Deployment` (API server) has its replicas set to
0 so that the API server can be run locally for debugging and faster development. The local API server is available by
pods on the cluster using the `ml-pipeline` `Service`.

#### Prerequisites

* The [kind CLI](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) is installed.
* The following ports are available on your localhost: 3000, 3306, 8080, 9000, and 8889. If these are unavailable,
  modify [kind-config.yaml](../tools/kind/kind-config.yaml) and configure the API server with alternative ports when
  running locally.
* If using a Mac, you will need to modify the
  [Endpoints](../manifests/kustomize/env/dev-kind/forward-local-api-endpoint.yaml) manifest to leverage the bridge
  network interface through Docker/Podman Desktop. See
  [kind #1200](https://github.com/kubernetes-sigs/kind/issues/1200#issuecomment-1304855791) for an example manifest.
* Optional: VSCode is installed to leverage a sample `launch.json` file.

#### Provisioning the Cluster

To provision the kind cluster, run the following from the Git repository's root directory,:

```bash
make -C backend dev-kind-cluster
```

This may take several minutes since there are many pods. Note that many pods will be in "CrashLoopBackOff" status until
all the pods have started.

#### Deleting the Cluster

Run the following to delete the cluster:

```bash
kind delete clusters dev-pipelines-api
```

#### Launch the API Server With VSCode

After the cluster is provisioned, you may leverage the following sample `.vscode/launch.json` file to run the API
server locally:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API Server (Kind)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/src/apiserver",
      "env": {
        "POD_NAMESPACE": "kubeflow",
        "DBCONFIG_MYSQLCONFIG_HOST": "localhost",
        "MINIO_SERVICE_SERVICE_HOST": "localhost",
        "MINIO_SERVICE_SERVICE_PORT": "9000",
        "METADATA_GRPC_SERVICE_SERVICE_HOST": "localhost",
        "METADATA_GRPC_SERVICE_SERVICE_PORT": "8080",
        "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST": "localhost",
        "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT": "8888"
      },
      "args": [
        "--config",
        "${workspaceFolder}/backend/src/apiserver/config",
        "-logtostderr=true"
      ]
    }
  ]
}
```

#### Using the Environment

Once the cluster is provisioned and the API server is running, you can access the API server at
[http://localhost:8888](http://localhost:8888)
(e.g. [http://localhost:8888/apis/v2beta1/pipelines](http://localhost:8888/apis/v2beta1/pipelines)).

You can also access the Kubeflow Pipelines web interface at [http://localhost:3000](http://localhost:3000).

You can also directly connect to the MariaDB database server with:

```bash
mysql -h 127.0.0.1 -u root
```
