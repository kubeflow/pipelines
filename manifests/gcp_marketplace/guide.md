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
After deployment, you can grant KFP proper permission by specifying its service account and binding
proper role to it.

Usually a functional KFP pipeline requires a [GCP service account](https://cloud.google.com/iam/docs/service-accounts) to use for 
authentication when calling other GCP services. This includes Cloud Storage as well as other services your pipeline might need, 
for example Dataflow, Dataproc. Specify the base64-encoded credentials for the service account you want to use.

This can be done through command line using `kubectl`.
```
export CLUSTER=<cluster-where-kfp-was-installed>
export ZONE=<zone-where-kfp-was-installed>
# Configure kubectl to connect with the cluster
gcloud container clusters get-credentials "$CLUSTER" --zone "$ZONE"
```
Then you can create and inject service account credential.
```
export PROJECT=<my-project>
export SA_NAME=<my-account>
# Create service account
gcloud iam service-accounts create $SA_NAME --display-name $SA_NAME
gcloud projects add-iam-policy-binding $PROJECT --member=serviceAccount:my-account@$PROJECT.iam.gserviceaccount.com --role=roles/storage.admin
# Also do this binding for other roles you need. For example, dataproc.admin and dataflow.admin
gcloud iam service-accounts keys create application_default_credentials.json --iam-account $SA_NAME@$PROJECT.iam.gserviceaccount.com
export SERVICE_ACCOUNT_TOKEN="$(cat application_default_credentials.json | base64 -w 0)"
echo -e "apiVersion: v1\nkind: Secret\nmetadata:\n  name: \"user-gcp-sa\"\n  namespace: \"${NAMESPACE}\"\n  labels:\n    app: gcp-sa\n    app.kubernetes.io/name: \"${APP_INSTANCE_NAME}\"\ntype: Opaque\ndata:\n  application_default_credentials.json: ${SERVICE_ACCOUNT_TOKEN}\n  user-gcp-sa.json: $SERVICE_ACCOUNT_TOKEN" > secret.yaml
kubectl apply -f secret.yaml
# Remove secret files
rm application_default_credentials.json test.yaml
```

Note that the above commands use `base64 -w 0` to disable line wrapping, this could be slightly different
across platforms.

## Tips

### Warning message in Pipelines Console
If you see the following warning message in the Pipeline console:

> Sorry, the server was only able to partially fulfill your request. Some data might not be rendered.

Possible reasons are:
- the cluster is under upgrading
- the new Kubeflow Pipeline instance is under deployment

Wait for a while and then refresh.