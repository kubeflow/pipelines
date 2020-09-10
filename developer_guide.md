# ML Pipeline Development Guideline

This document describes the development guideline to contribute to ML pipeline project. Please check the [main page](https://github.com/kubeflow/pipelines/blob/master/README.md) for instruction on how to deploy a ML pipeline system.

## ML pipeline deployment

The Pipeline system is included in kubeflow. See [Getting Started Guide](https://www.kubeflow.org/docs/started/getting-started/) for how to deploy with Kubeflow.

## Build Images

The Docker images need to be uploaded to a Docker container repository such as [GCR](https://cloud.google.com/container-registry/) to be used on a kubernetes cluster.

First export the base container repository URL and folder (i.e. project name):

```bash
export URI='gcr.io'
export FOLDER='myproject'
```

Then authenticate to your container repository:

```bash
# Push to GCR
$ gcloud auth configure-docker
```

> **Run all commands in the repository root directory**

To build the API server image and upload it to your container repository on x86_64 machines:

```bash
docker build -t $URI/$FOLDER/api-server:latest -f backend/Dockerfile .
docker push $URI/$FOLDER/api-server:latest
```

To build the scheduled workflow controller image and upload it to your container registry:

```bash
docker build -t $URI/$FOLDER/scheduledworkflow:latest -f backend/Dockerfile.scheduledworkflow .
docker push $URI/$FOLDER/scheduledworkflow:latest
```

To build the viewer CRD controller image and upload it to your container registry:

```bash
docker build -t $URI/$FOLDER/viewer-crd-controller:latest -f backend/Dockerfile.viewercontroller .
docker push $URI/$FOLDER/viewer-crd-controller:latest
```

To build the persistence agent image and upload it to your container registry:

```bash
docker build -t $URI/$FOLDER/persistenceagent:latest -f backend/Dockerfile.persistenceagent .
docker push $URI/$FOLDER/persistenceagent:latest
```

To build the frontend image and upload it to your container registry:

```bash
docker build -t $URI/$FOLDER/frontend:latest -f frontend/Dockerfile .
docker push $URI/$FOLDER/frontend:latest
```

### Minikube

Minikube can pick your local Docker image so you don't need to upload to remote repository.

For example, to build API server image

```bash
docker build -t ml-pipeline-api-server -f backend/Dockerfile .
```

## Python based visualizations

Python based visualizations are a new method to visualize results within the
Kubeflow Pipelines UI. For more information about Python based visualizations
please visit the [documentation page](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations).
To create predefine visualizations please check the [developer guide](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/README.md).

## Unit test

### API server

Run unit test for the API server

```bash
cd backend/src/ && go test ./...
```

### Frontend

TODO: add instruction

### DSL

```bash
pip install ./dsl/ --upgrade && python ./dsl/tests/main.py
pip install ./dsl-compiler/ --upgrade && python ./dsl-compiler/tests/main.py
```

## Integration test & E2E test

Check [this](https://github.com/kubeflow/pipelines/blob/master/test/README.md) page for more details.

## Troubleshooting

**Q: How to access to the database directly?**

You can inspect mysql database directly by running:

```bash
kubectl run -it --rm --image=$URI/ml-pipeline/mysql:5.6 --restart=Never mysql-client -- mysql -h mysql
mysql> use mlpipeline;
mysql> select * from jobs;
```

**Q: How to inspect object store directly?**

Minio provides its own UI to inspect the object store directly:

```bash
kubectl port-forward -n ${NAMESPACE} $(kubectl get pods -l app=minio -o jsonpath='{.items[0].metadata.name}' -n ${NAMESPACE}) 9000:9000
Access Key:minio
Secret Key:minio123
```

**Q: I see an error of exceeding Github rate limit when deploying the system. What can I do?**

See [Ksonnet troubleshooting page](https://github.com/ksonnet/ksonnet/blob/master/docs/troubleshooting.md#github-rate-limiting-errors)

**Q: How do I check my API server log?**

API server logs are located at /tmp directory of the pod. To SSH into the pod, run:

```bash
kubectl exec -it -n ${NAMESPACE} $(kubectl get pods -l app=ml-pipeline -o jsonpath='{.items[0].metadata.name}' -n ${NAMESPACE}) -- /bin/sh
```

or

```bash
kubectl logs -n ${NAMESPACE} $(kubectl get pods -l app=ml-pipeline -o jsonpath='{.items[0].metadata.name}' -n ${NAMESPACE})
```

**Q: How to check my cluster status if I am using Minikube?**

Minikube provides dashboard for deployment

```bash
minikube dashboard
```
