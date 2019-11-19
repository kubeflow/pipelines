# Deployer of OpenVINO Model Server

This component triggers deployment of [OpenVINO Model Server](https://github.com/IntelAI/OpenVINO-model-server) in Kubernetes.

It applies the passed component parameters on jinja template and applied deployment and server records.



```bash
./deploy.sh
     --model-export-path
     --cluster-name
     --namespace
     --server-name
     --replicas
     --batch-size
     --model-version-policy
     --log-level
```


## building docker image


```bash
docker build --build-arg http_proxy=$http_proxy --build-arg https_proxy=$https_proxy .
```

## testing the image locally


```