# Deploy Kubeflow Pipelines from Google Cloud Marketplace

You can go to [Google Cloud Marketplace](https://console.cloud.google.com/marketplace) to deploy Kubeflow Pipelines via UI.
You can find the instances in [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines).

## Cluster
You can choose an existing cluster or create a new cluster. Creating a new cluster from Marketplace config page doesn't allow
you to customize the cluster setting. If you want to customize the cluster,
please go to [Google Kubernetes Engine](https://console.cloud.google.com/kubernetes/list) to create one.

Please notice that you only can `install one Kubeflow Pipelines into one cluster`.

## Namespace
Please specify a [Kubenetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/).

## App instance name
Please specify a name for this instance. It's possible you install muliple instances to muliple clusters.
Please specify a friendly name so that you can well manage it in [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines).

## GCP Service Account credentials
It's used to call other GCP services like Google Cloud Storage, Dataflow etc.

This deployment requires a service account to use for authentication when calling other GCP services. 
Specify the base64-encoded credentials for the service account you want to use. 
You can get these credentials by running the following command in a terminal window. 

```
$ gcloud iam service-accounts keys create application_default_credentials.json --iam-account [your-service-account] && cat application_default_credentials.json | base64
```

If you are using Linux, please use `base64 -w 0` to disable line wrapping.

## Use managed storage
This field controls whether use Google Cloud SQL and Google Cloud Storage.
If you don't use managed storage, it will use Kubernetes [Persisent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and in-cluster MySQL.
It means if you don't backup data and delete the cluster, all data will be lost.

## CloudSQL instance connection name
This field must be specified if choose to use managed storage.
Provide the instance connection name for an existing Cloud SQL for MySQL instance.
The instance connection name can be found on the instance detail page in the Cloud SQL console. 
The instance connection name uses the format project:zone:instance-name, for example,myproject:us-central1:myinstance. 
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
The prefix of the database name. Kubeflow Pipelines will create two databases, prefix`_pipeline` and prefix`_metadata`.
Use lowercase letters, numbers, and hyphens. Start with a letter.
If the prefix specified is same as an old deployment in the past,
the deployment will recover from an old deployment.
If this not specified, the app instance name will be used.

## Deploy
Clicking `Deploy` button, it will start deploy Kubeflow Pipelines into cluster.
It might take few minutes. Please wait. After all done, you can go to [Pipelines Console](http://pantheon.corp.google.com/ai-platform/pipelines).
Pipelines Console helps you easily access the Kubeflow Pipelines instances.