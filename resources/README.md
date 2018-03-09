# [WIP] Pipeline CRD/controller

This is work in progress.

The following assumes that the current directory is the top level directory of the repository. 

## Running the pipeline controller locally


Install the dependencies: 

```
glide install
```

(Optional) Regenerate the client: 

```
./hack/update-codegen.sh
```

Run the controller locally: 

```
go run ./resources/pipeline/*.go -kubeconfig=$HOME/.kube/config -alsologtostderr=true
```

## Creating and deleting pipelines

Create the pipeline CRD: 

```
kubectl create -f ./resources/pipeline/examples/pipeline-crd.yaml
```

Create a pipeline:

```
kubectl create -f ./resources/pipeline/examples/example-pipeline.yaml
```

Get the list of running pipelines: 

```
kubectl get pipelines
```

Delete the pipeline:

```
kubectl delete -f ./resources/pipeline/examples/pipeline-crd.yaml
```

Delete the pipeline CRD: 

```
kubectl delete -f ./resources/pipeline/examples/pipeline-crd.yaml
```