# Common issues and Troubleshooting

Kubernetes provides several useful tools for debugging pipelines. To continiously monitor the status of all Kubeflow pods and services, one can do:

`watch kubectl -n kubeflow get all `

```
pod/profiles-6b5f999584-pjj7j                                   1/1     Running            1          86m
pod/pytorch-operator-577469ccd8-p8r4z                           1/1     Running            1          86m
pod/resnet-cifar10-pipeline-2f6v8-4205778421                    0/2     Error              0          40m
pod/resnet-cifar10-pipeline-5pz5w-2553990163                    0/2     Completed          0          4m
pod/resnet-cifar10-pipeline-5pz5w-2832477042                    0/2     Completed          0          29m
pod/resnet-cifar10-pipeline-5pz5w-349525310                     0/2     Completed          0          25m
pod/resnet-cifar10-pipeline-5pz5w-4048043379                    0/2     Completed          0          3m52s
pod/resnet-cifar10-pipeline-t2kv2-4047898736                    0/2     Error              0          82m
```

Pipelines with error status can be inspected with:

`kubectl logs   resnet-cifar10-pipeline-t2kv2-4047898736 --namespace kubeflow  --all-containers`

To continuously monitor a pipeline:

`kubectl logs -f   resnet-cifar10-pipeline-t2kv2-4047898736 --namespace kubeflow  --all-containers`

## 1. Kubeflow ambassador stuck in a crash loop.
An issue is described at 
https://github.com/kubeflow/kubeflow/issues/344.
One reason might be due to a defect Kubenetes core DNS. Check if K8 DNS service is running properly:

```kubectl get pods --namespace=kube-system -l k8s-app=kube-dns```

```
NAME                      READY   STATUS    RESTARTS   AGE                                                                    
coredns-fb8b8dccf-p622f   1/1     Running   19         157m
coredns-fb8b8dccf-zcplw   1/1     Running   19         157m
```

Check for common problems at diagnosis at:
https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/

One common solution is to remove any loopback interfaces, such as `127.0.0.1` from 
`/run/systemd/resolve/resolv.conf`.

## 2. Kubeflow v0.5.1 stuck at startup, no resources found
https://github.com/kubeflow/kubeflow/issues/2863

A temporary solution is by moving the following lines 
```
      set +x
      if [[ "${PLATFORM}" == "minikube" ]] || [[ "${PLATFORM}" == "docker-for-desktop" ]]; then
        if is_kubeflow_ready; then
          mount_local_fs
          setup_tunnels
        else
          echo -e "${RED}Unable to get kubeflow ready${NC}"
        fi
      fi
      set -x
```
from https://github.com/kubeflow/kubeflow/blob/master/scripts/kfctl.sh#L252 to https://github.com/kubeflow/kubeflow/blob/master/scripts/kfctl.sh#L588:

