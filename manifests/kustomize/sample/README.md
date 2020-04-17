# Sample installation

1. Prepare a cluster and setup kubectl context
Do whatever you want to customize your cluster. You can use existing cluster
or create a new one.
- **ML Usage** GPU normally is required for deep learning task.
You may consider create **zero-sized GPU node-pool with autoscaling**.
Please reference [GPU Tutorial](/samples/tutorials/gpu/).
- **Security** You may consider use **Workload Identity** in GCP cluster.

Here for simplicity we create a small cluster with **--scopes=cloud-platform** 
to save credentail configure efforts.

```
gcloud container clusters create mycluster \
  --zone us-central1-a \
  --machine-type n1-standard-2 \
  --scopes cloud-platform \
  --enable-autoscaling \
  --min-nodes 1 \
  --max-nodes 5 \
  --num-nodes 3
```

2. Prepare CloudSQL

Create CloudSQL instance. [Console](https://console.cloud.google.com/sql/instances).

Here is a sample for demo. You may use **Private IP** in certain VPC network.
```
gcloud beta sql instances create mycloudsqlname \
  --database-version=MYSQL_5_7 \
  --tier=db-n1-standard-1 \
  --region=us-central1 \
  --root-password=password123
```

3. Prepare GCS Bucket

Create Cloud Storage bucket. [Console](https://console.cloud.google.com/storage).

```
gsutil mb -p myProjectId gs://myBucketName/
```

4. Customize your values
- Edit **params.env** to your values
- Edit kustomization.yaml to set your namespace, e.x. "kubeflow"

5. Install

```
kubectl apply -k cluster-scoped-resources/

kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s

kubectl apply -k sample/
# If upper one action got failed, e.x. you used wrong value, try delete, fix and apply again
# kubectl delete -k sample/

kubectl wait applications/mypipeline -n mykubeflow --for condition=Ready --timeout=1800s

kubectl describe configmap inverse-proxy-config -n kubeflow | grep googleusercontent.com
```

6. Post-installation configures

It depends on how you create the cluster, 
- if the cluster is created with **--scopes=cloud-platform**, no actions required
- if the cluster is on Workload Identity, please run **gcp-workload-identity-setup.sh**
  - make sure the Google Service Account (GSA) can access the CloudSQL instance and GCS bucket
  - if your workload calls other GCP APIs, make sure the GSA can access them
