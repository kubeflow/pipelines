# Common issues and Troubleshooting


## 1. Kubeflow ambassador gets stuck in a crash loop.
An issue is described at 
https://github.com/kubeflow/kubeflow/issues/344.
One reason might be due to a defect Kubenetes core DNS. Check if K8 DNS service is running properly:

```kubectl get pods --namespace=kube-system -l k8s-app=kube-dns```

Check for common problems at diagnosis at:
https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/

One common solution is to remove any loopback interfaces, such as `127.0.0.1` from 
`/run/systemd/resolve/resolv.conf`.
