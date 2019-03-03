Kubeflow Pipelines CLI
----------------------

## Installation

This assumes that Go is installed on your machine.
```
export GO111MODULE=on
go get -u github.com/kubeflow/pipelines/backend/src/cmd/ml
```

## Usage

Ensure kubectl is pointed to the right cluster and context.
```
$ ml
This is the command line interface to manipulate pipelines.

Usage:
  ml [command]

Available Commands:
  experiment  Manages experiments
  help        Help about any command
  job         Manages jobs
  pipeline    Manage pipelines
  run         Manage runs
```

### List Pipelines in the cluster

```
$ ml pipeline list -n kubeflow
```