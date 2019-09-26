# Deploy Kubeflow Pipelines from Google Cloud Marketplace

You can go to [Google Cloud Marketplace](https://console.cloud.google.com/marketplace) to deploy Kubeflow Pipelines via UI.
To get to Kubeflow Pipelines page in Marketplace, enter "Kubeflow Pipelines" in the search box in Marketplace landing page. 
Alternatively you can access the [Marketplace page for Kubeflow Pipelines](https://console.cloud.google.com/marketplace/details/google-cloud-ai-platform/kubeflow-pipelines) directly.

You can find Kubeflow Pipelines instances deployed via Marketplace in [Pipelines Console](http://console.cloud.google.com/ai-platform/pipelines).

## Cluster

You can choose an existing cluster for Kubeflow Pipelines deployment, or create a new cluster. New cluster created from Marketplace config page cannot be customized.
If you want to customize the cluster, please go to [Google Kubernetes Engine](https://console.cloud.google.com/kubernetes/list) to create one.

Please notice that you can only `install one Kubeflow Pipelines into one cluster`.

## Namespace
Please specify a [Kubenetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

## App instance name
It is possible to install multiple instances of Kubeflow Pipelines in a single cluster.
This name will show up in [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines), so please use a user friendly name.

## GCP Service Account credentials
This credential is used to access GCS and Cloud SQL for managed storage option. 
Also, if you pipeline need to access other GCP services (e.g. Dataflow), you will need to configure this credential.

This deployment requires a service account to use for authentication when calling other GCP services. 
Specify the base64-encoded credentials for the service account you want to use. 
You can get these credentials by running the following command in a terminal window. 

```
$ gcloud iam service-accounts keys create application_default_credentials.json --iam-account [your-service-account] && cat application_default_credentials.json | base64
```

This command will create a new key under the service account. Please note that a single service account can only have 10 keys. Exising key also can be used.
If you are running this command on Linux, please use `base64 -w 0` to disable line wrapping.

## Use managed storage
By selecting this option, your Kubeflow Pipelines deployment will use Google Cloud Storage and Google Cloud SQL for storage.
If you don't use managed storage, it will use Kubernetes [Persisent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and in-cluster MySQL.
Using managed storage will take care of data backup for you. In case you delete the cluster, your data will be preserved.

## Cloud SQL instance connection name
This is a required field if you choose to use managed storage.
Provide the instance connection name for an existing Cloud SQL for MySQL instance.
The instance connection name can be found on the instance detail page in the Cloud SQL console. 
The instance connection name uses the format project:zone:instance-name. Example: myproject:us-central1:myinstance.
For more details on how to create a new instance, see https://cloud.google.com/sql/docs/mysql/quickstart.

## Datatbase username
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
Clicking `Deploy` button, it will start deploy Kubeflow Pipelines into cluster.
It might take few minutes. Please wait. After all done, you can go to [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines).
Pipelines Console helps you easily access the Kubeflow Pipelines instances.

Clicking `Deploy` button will start deploying Kubeflow Pipelines into the cluster you specified.
After clicking the button, you will be taken to GKE console. The deployment might take a few minutes.
After the deployment is complete, you can go to [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines) to view the new Kubeflow Pipelines instance.

## Tips

### Warning message in Pipelines Console
Sometimes Pipelines Console may give you a warning message.

> Sorry, the server was only able to partially fulfill your request. Some data might not be rendered.

The resource would be 
- the cluster is under upgrading
- the new Kubeflow Pipeline instance is under deployment

Wait for a while and then refresh.