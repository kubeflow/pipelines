- Make sure minikube and kubectl are installed
- Make sure Argo is installed and started.
- Git clone the repository under ${GOPATH}/src, as recommended by golang. 

- Run bash script
```
./run.sh
```
- Forward the pipeline manager port so you can access it from host machine
```
kubectl get pod
kubectl port-forward [pipeline-manager-pod] 8888:8888
```
- Get shell for the api server pod
```
kubectl exec -it pipeline-manager-single -- /bin/bash
```
- The API server logs are located in 
```
\tmp
```
- You can inspect the package files through Minio UI
```
kubectl port-forward [minio-pod] 9000:9000
``` 
- To clean up 
 ```
 kubectl delete deploy,svc -l app=ml-pipeline-manager
 kubectl delete svc minio-service
 kubectl delete pvc minio-pv-claim
 ```