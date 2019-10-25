# Deploy Kubeflow Pipelines from Google Cloud Marketplace

Go to [Google Cloud Marketplace](https://console.cloud.google.com/marketplace) to deploy Kubeflow Pipelines by using a graphical interface.
You can go to the [Marketplace page for Kubeflow Pipelines](https://console.cloud.google.com/marketplace/details/google-cloud-ai-platform/kubeflow-pipelines) directly, or search for "Kubeflow Pipelines" from the Marketplace landing page.

Once you have deployed Kubeflow Pipelines instances, you can view and manage them in the [Pipelines Console](http://console.cloud.google.com/ai-platform/pipelines).

## Cluster

You can deploy Kubeflow Pipelines to a new cluster or an existing cluster. New clusters aren't customizable, so if you need that, you should use the [Google Kubernetes Engine](https://console.cloud.google.com/kubernetes/list) to create a cluster that meets your requirements and then deploy to that.

You can only deploy one Kubeflow Pipelines into a given cluster.

## Namespace
Specify a [Kubenetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

## App instance name
Specify an app instance name to help you identify this instance.

## Deploy
Click `Deploy` to start deploying Kubeflow Pipelines into the cluster you specified.
Deployment might take few minutes, so please be patient. After deployment is complete, go to the [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines) to access the Kubeflow Pipelines instance.

## GCP Service Account credentials
If you run pipelines that requires calling any GCP services, such as Cloud Storage, Cloud ML Engine, Dataflow, or Dataproc, you need to set the [application default credential](https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application) to a pipeline step by mounting the proper [GCP service account](https://cloud.google.com/iam/docs/service-accounts) token as a [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/).

First point your `kubectl` current context to your cluster
```
export PROJECT_ID=<my-project-id>
export CLUSTER=<cluster-where-kfp-was-installed>
export ZONE=<zone-where-kfp-was-installed>
# Configure kubectl to connect with the cluster
gcloud container clusters get-credentials "$CLUSTER" --zone "$ZONE" --project "$PROJECT_ID"
```

Then you can create a service account with the necessary IAM permissions
```
export SA_NAME=<my-account>
export NAMESPACE=<namespace-where-kfp-was-installed>
# Create service account
gcloud iam service-accounts create $SA_NAME --display-name $SA_NAME --project "$PROJECT_ID"
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/storage.admin

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:$SA_NAME@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/ml.admin

# More roles can be binded if your pipeline requires it.
# --role=roles/dataproc.admin
# --role=roles/dataflow.admin
```

and store the service account credential as a Kubernetes secret `user-gcp-sa` in the cluster
```
gcloud iam service-accounts keys create application_default_credentials.json --iam-account $SA_NAME@$PROJECT_ID.iam.gserviceaccount.com

# Make sure the secret is created under the correct namespace.
kubectl config set-context --current --namespace=$NAMESPACE

kubectl create secret generic user-gcp-sa \
  --from-file=user-gcp-sa.json=application_default_credentials.json \
  --dry-run -o yaml  |  kubectl apply -f -
```
Remove the private key file if needed
```
rm application_default_credentials.json
```

## Tips

### Warning message in Pipelines Console
If you see the following warning message in the Pipeline console:

> Sorry, the server was only able to partially fulfill your request. Some data might not be rendered.

Possible reasons are:
- the cluster is under upgrading
- the new Kubeflow Pipeline instance is under deployment

Wait for a while and then refresh.

### Access Kubeflow Pipelines UI got forbidden
It's possible you can access [Console](https://console.cloud.google.com/ai-platform/pipelines/clusters)
but can't `Open Pipelines Dashboard`. It gives you following message:

> forbidden

Reason:
- Others created the cluster and deployed the instances for you.
- You don't have corresponding permission to access it.

Actions:
- Please ask admin to find out the Google Service Account used to create the cluster and then add your account as its `Service Account User` via [Service accounts](https://console.cloud.google.com/iam-admin/serviceaccounts). From the list table, check the
service account, click the `Info Panel`, you will find a button `Add member` and add it
as `Service Account User`. The Google Service Account is [Compute Engine default service account](https://cloud.google.com/compute/docs/access/service-accounts#compute_engine_service_account) if you didn't set it when creating the cluster.
- Please also add your account as `Project Viewer` via [IAM](https://console.cloud.google.com/iam-admin/iam).

For simplicity but not good for security, adding as `Project Editor` also can work.
