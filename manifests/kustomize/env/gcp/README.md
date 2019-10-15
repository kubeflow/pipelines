# TL;DR
1. To access the GCP services, the application needs a GCP service account token. Download the token to the current folder manifests/kustomize/env/gcp. [Reference](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys)
```
gcloud iam service-accounts keys create application_default_credentials.json \
  --iam-account [SA-NAME]@[PROJECT-ID].iam.gserviceaccount.com
```
2. [Create](https://cloud.google.com/sql/docs/mysql/quickstart) or use an existing CloudSQL instance. The service account should have the access to the CloudSQL instance.
3. Fill in gcp-configurations-patch.yaml with your CloudSQL and GCS configuration.

# Why Cloud SQL and GCS
Kubeflow Pipelines keeps its metadata in mysql database and artifacts in S3 compatible object storage. 
Using CloudSQL and GCS for persisting the data provides better reliability and performance, as well as things like data backups, and usage monitoring.
This is the recommended setup especially for production environments.   
