- Make sure minikube and kubectl are installed
- Run 
```
kubectl create -f ./ml-pipeline-e2e.yaml
```
- Forward the front end port, and open http://localhost:8080
```
kubectl port-forward $(kubectl get pods -l app=ml-pipeline-frontend -o jsonpath='{.items[0].metadata.name}') 8080:3000
```
- To clean up 
 ```
kubectl delete -f ./ml-pipeline-e2e.yaml
 ```