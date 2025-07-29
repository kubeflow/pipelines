# Azure DevOps components for Kubeflow Pipelines

## Components

### Queue Pipeline

The Queue Pipeline component enables you to queue an Azure Pipelines pipeline from a Kubeflow pipeline.

## Authentication

The components in this collection authenticate to Azure DevOps by using a [PAT (Personal Access Token)](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops&tabs=preview-page#create-a-pat). The PAT should be mounted to `/app/secrets/azdopat`.

This can be done by storing the PAT in the the Kubernetes secret `azdopat`, then applying `use_secret(secret_name='azdopat', secret_volume_mount_path='/app/secrets')` to the task in the pipeline.
