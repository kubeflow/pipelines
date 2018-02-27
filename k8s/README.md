- Make sure minikube and kubectl are installed
- Make sure Argo is installed and started.
- Git clone the repository under ${GOPATH}/src, as recommended by golang. 
- Point minikube to use docker daemon on the host machine by running 
```
eval $(minikube docker-env)
```
- Build docker image
```
docker build -t pipeline-manager ${GOPATH}/src/ml/apiserver
```
- Start pipeline manager
```
kubectl create -f ${GOPATH}/src/ml/k8s/pipeline-manager.yaml
```
- Forward the pipeline manager port so you can access it from host machine
```
kubectl port-forward pipeline-manager 8888:8888
```