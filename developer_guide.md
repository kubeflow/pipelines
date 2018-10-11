# ML Pipeline Development Guideline

This document describes the development guideline to contribute to ML pipeline project. Please check the [main page](https://github.com/googleprivate/ml/blob/master/README.md) for instruction on how to deploy a ML pipeline system.

## ML pipeline deployment

The ML pipeline system uses [Ksonnet](https://ksonnet.io/) as part of the deployment process.
Ksonnet provides the flexibility to generate Kubernetes manifests from parameterized templates and
makes it easy to customize Kubernetes manifests for different use cases.
The Ksonnet is wrapped in a customized bootstrap container so a user don't need to explicitly deal
with Ksonnet to install ML pipeline.

The docker container accepts various parameters to customize your deployment.
- **--namespace** the namespace to deploy to
- **--api_image** the API server image to use
- **--ui_image** the webserver image to use
- **--report_usage** whether to report usage for the deployment
- **--uninstall** to uninstall everything.

See [bootstrapper.yaml](https://github.com/googleprivate/ml/blob/master/bootstrapper.yaml) for examples on how to pass in parameter.

Alternatively, you can use [deploy.sh](https://github.com/googleprivate/ml/blob/master/ml-pipeline/deploy.sh) if you want to interact with Ksonnet directly.
To deploy, run the script locally.
```bash
$ ml-pipeline/deploy.sh
```
And you will se a Ksonnet [APP](https://ksonnet.io/docs/concepts#application) folder generated in your current path. If you want to update or delete the K8s resource created by the deployment, run
```bash
# Update
$ cd ml-pipeline && ks apply default
# Delete
$ cd ml-pipeline && ks delete default
```


## Build Image

### GKE
To be able to use GKE, the Docker images need to be uploaded to a public Docker repository, such as [GCR](https://cloud.google.com/container-registry/)

To build the API server image and upload it to GCR: 
```bash
# Run in the repository root directory 
$ docker build -t gcr.io/<your-gcp-project>/api-server:latest -f backend/Dockerfile ./backend
# Push to GCR
$ gcloud auth configure-docker
$ docker push gcr.io/<your-gcp-project>/api-server:latest
```

To build the scheduled workflow controller image and upload it to GCR: 
```bash
# Run in the repository root directory 
$ docker build -t gcr.io/<your-gcp-project>/scheduledworkflow:latest -f backend/Dockerfile.scheduledworkflow ./backend
# Push to GCR
$ gcloud auth configure-docker
$ docker push gcr.io/<your-gcp-project>/scheduledworkflow:latest
```

To build the persistence agent image and upload it to GCR: 
```bash
# Run in the repository root directory 
$ docker build -t gcr.io/<your-gcp-project>/persistenceagent:latest -f backend/Dockerfile.persistenceagent ./backend
# Push to GCR
$ gcloud auth configure-docker
$ docker push gcr.io/<your-gcp-project>/persistenceagent:latest
```

### Minikube
Minikube can pick your local Docker image so you don't need to upload to remote repository.

For example, to build API server image  
```bash
$ docker build -t ml-pipeline-api-server backend/src
```

### Update deployment image
If your change updates deployment image (e.g. add new service account, change image version etc.),
remember to update the deployment image as well, and use that image to create deployment job.
```bash
$ docker build -t gcr.io/<your-gcp-project>/bootstrapper ml-pipeline/
$ gcloud auth configure-docker
$ docker push gcr.io/<your-gcp-project>/bootstrapper
```

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

## Integration test

### API server
Check [this](https://github.com/googleprivate/ml/blob/master/test/apiserver/README.md) page for more details.

## E2E test
TODO: Add instruction


## Troubleshooting

**Q: How to access to the database directly?**

You can inspect mysql database directly by running:
```bash
kubectl run -it --rm --image=mysql:5.6 --restart=Never mysql-client -- mysql -h mysql
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

See Ksonnet troubleshooting page [page](https://github.com/ksonnet/ksonnet/blob/master/docs/troubleshooting.md#github-rate-limiting-errors)

**Q: How do I check my API server log?**

API server logs are located at /tmp directory of the pod. To SSH into the pod, run:
```bash
kubectl exec -it -n ${NAMESPACE} $(kubectl get pods -l app=ml-pipeline -o jsonpath='{.items[0].metadata.name}' -n ${NAMESPACE}) -- /bin/bash
```

**Q: How to check my cluster status if I am using Minikube?**  

Minikube provides dashboard for deployment
```bash
minikube dashboard
```
