# TL;DR
1. Download the GCP service account token to same folder. [Document](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys)
```
gcloud iam service-accounts keys create application_default_credentials.json \
  --iam-account [SA-NAME]@[PROJECT-ID].iam.gserviceaccount.com
```
2. Create or use an existing CloudSQL instance. [Document](https://cloud.google.com/sql/docs/mysql/quickstart). The service account should have access to the CloudSQL instance.
3. Fill in gcp-configurations-patch.yaml with the CloudSQL and GCS information. 

# Why Cloud SQL and GCS
Kubeflow Pipelines keeps its metadata in mysql and artifacts in S3 compatible object storage. 
When deploying on GCP, you could choose to use CloudSQL and GCS for persisting the data. 
This provides better reliability and performance, as well as things like data backups, for production environments.   
