- Make sure minikube and kubectl are installed
- Make sure Argo is installed and started.
- Git clone the repository under ${GOPATH}/src, as recommended by golang. 
- Point minikube to use docker daemon on the host machine by running 
```
eval $(minikube docker-env)
```
- Build docker image
```
docker build -t pipeline-manager-api-server ${GOPATH}/src/ml/apiserver
```
- Start pipeline manager
```
kubectl create -f ${GOPATH}/src/ml/k8s/pipeline-manager.yaml
```
- Forward the pipeline manager port so you can access it from host machine
```
kubectl get pod
kubectl port-forward [pipeline-manager-pod] 8888:8888
```
-- Get shell for the pod
```
kubectl exec -it pipeline-manager-single -- /bin/bash
```
-- The Logs are located in 
```
\tmp
```
-- The sqlite db is located in 
```
/bin/pipelines.db
```
To dump the db
```
$ sqlite3 /bin/pipelines.db
sqlite3> .dump
```
 