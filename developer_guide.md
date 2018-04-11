# ML Pipeline Development Guideline

This document describes the development guideline to contribute to ML pipeline project. Please check the [main page](https://github.com/googleprivate/ml/blob/master/README.md) for instruction on how to deploy a ML pipeline system.

## Deploy with your own image

By default the deployment uses the official images stored in [GCR](https://cloud.google.com/container-registry/). 

You can use the deployment script, and provide your own UI or API server image. See next section on how to build your own image.
```
NAMESPACE=[namespace]
deploy/deploy.sh --namespace ${NAMESPACE} --apiserver [api-image] --ui [ui-image]
```
Note Ksonnet creates an [app](https://ksonnet.io/docs/concepts#application) folder under the current path. If you want to update or delete the K8s resource created by the deployment, run
```
# Update
cd ml-pipeline && ks apply default 
# Delete
cd ml-pipeline && ks delete default 
```

## Build Image
 
### GKE
To be able to use GKE, the Docker images need to be uploaded to a public Docker repository, such as [GCR](https://cloud.google.com/container-registry/)

Here is an example to build API server image and upload to GCR.  
````
# Run under src/ directory 
docker build -t gcr.io/your-gcr/api-server:latest src/apiserver
# Push to GCR
gcloud docker -- push gcr.io/your-gcr/api-server:latest
````

### Minikube
Minikube can pick your local Docker image so you don't need to upload to remote repository.

For example, to build API server image  
```
docker build -t ml-pipeline-api-server src
```

## Unit test

### API server
Run unit test for the API server
```
cd src/ && go test ./...
```
### Frontend
TODO: add instruction

## Integration test
TODO: Add instruction for for frontend and backend

## E2E test
TODO: Add instruction

## Publish [optional]
**[Note]** The published images are used as private release for the ML pipeline team. Please have things tested before publishing. 

To publish an updated version of the image to GCR 
```
docker build -t gcr.io/ml-pipeline/api-server:v1alpha1.x src/apiserver
gcloud docker -- push gcr.io/ml-pipeline/api-server:v1alpha1.x
```

## Troubleshooting

**Q: How to access to the database directly?**

You can inspect mysql database directly by running:
```
kubectl run -it --rm --image=mysql:5.6 --restart=Never mysql-client -- mysql -h mysql
mysql> use mlpipeline;
mysql> select * from jobs;
```

**Q: How to inspect object store directly?**

Minio provides its own UI to inspect the object store directly:
```
kubectl port-forward -n ${NAMESPACE} $(kubectl get pods -l app=minio -o jsonpath='{.items[0].metadata.name}' -n ${NAMESPACE}) 9000:9000
Access Key:minio
Secret Key:minio123
```

**Q: I see an error of exceeding Github rate limit when deploying the system. What can I do?**

See Ksonnet troubleshooting page [page](https://github.com/ksonnet/ksonnet/blob/master/docs/troubleshooting.md#github-rate-limiting-errors)

**Q: How do I check my API server log?**

API server logs are located at /tmp directory of the pod. To SSH into the pod, run:
```
kubectl exec -it -n ${NAMESPACE} $(kubectl get pods -l app=ml-pipeline -o jsonpath='{.items[0].metadata.name}' -n ${NAMESPACE}) -- /bin/bash
```

**Q: How to check my cluster status if I am using Minikube?**  

Minikube provides dashboard for deployment
```
minikube dashboard
```
