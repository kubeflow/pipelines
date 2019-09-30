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

## GCP Service Account credentials
This deployment requires a [GCP service account](https://cloud.google.com/iam/docs/service-accounts) to use for authentication when calling other GCP services. This includes Cloud Storage and Cloud SQL if you are using managed storage, as well as other services your pipeline might need, for example Dataflow. Specify the base64-encoded credentials for the service account you want to use.

You can get these credentials by running the following command in a terminal window. This command will create a new key under the service account. Please note that a single service account can only have 10 keys. 

```
$ gcloud iam service-accounts keys create application_default_credentials.json --iam-account [your-service-account] && cat application_default_credentials.json | base64
```

Existing key also can be used.

```
cat existing_credentials.json | base64
```

If you are running this command on Linux, please use `base64 -w 0` to disable line wrapping.

## Use managed storage
Select this option if you want your Kubeflow Pipelines deployment to use Cloud Storage and Cloud SQL for storage. Managed storage takes care of data backup for you, so that your data will be preserved in the case that your cluster is accidentally deleted.

If you don't select this option, your Kubeflow Pipelines deployment will use Kubernetes [Persisent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and in-cluster MySQL.

## Cloud SQL instance connection name
This is a required field if you choose to use managed storage.
Provide the instance connection name for an existing Cloud SQL for MySQL instance.
The instance connection name can be found on the instance detail page in the Cloud SQL console. 
The instance connection name uses the format `project:zone:instance-name`. Example: myproject:us-central1:myinstance.
For more details on how to create a new instance, see https://cloud.google.com/sql/docs/mysql/quickstart.

## Database username
The database username to use when connecting to the Cloud SQL instance. 
If you leave this field empty, the deployment will use the default 'root' user account to connect. 
For more details about MySQL users, see https://cloud.google.com/sql/docs/mysql/users.

## Database password
The database password to use when connecting to the Cloud SQL instance.
If you leave this field empty, the deployment will try to connect to the instance without providing a password.
This will fail if a password is required for the username you provided.

## Database name prefix
The prefix of the database name. Kubeflow Pipelines will create two databases, `prefix_pipeline` and `prefix_metadata`.
Use lowercase letters, numbers, and hyphens. The name must start with a letter.
If you are reusing a prefix from a previous deployment, your new deployment will recover the data from that deployment.
If the prefix is not specified, the app instance name will be used.

## Deploy
Click `Deploy` to start deploying Kubeflow Pipelines into the cluster you specified.
Deployment might take few minutes, so please be patient. After deployment is complete, go to the [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines) to access the Kubeflow Pipelines instance.

## Tips

### Warning message in Pipelines Console
If you see the following warning message in the Pipeline console:

> Sorry, the server was only able to partially fulfill your request. Some data might not be rendered.

Possible reasons are:
- the cluster is under upgrading
- the new Kubeflow Pipeline instance is under deployment

Wait for a while and then refresh.