# Azure Synapse Run Spark Jobs

This component submit a spark job in [Azure Syanpse workspace](https://docs.microsoft.com/en-us/azure/synapse-analytics/). It provides the following functions:

- Submit a spark job in a spark pool in an Azure Synpase workspace
  - If the spark pool doesn't exist, a new pool will be created with the [configuration](./src/spark_pool_config.yaml)
- You can choose to return after the job is scheduled, or wait until the just finished.

## Input Parameters
### Job scheduling parameters
| Name | Type | Required | Description |
| -------------------------- | ------ | -------- | ----------------------------------------|
| executor_size | String | Y | size of executors. Accepted values: Large, Medium, Small |
| executors | Integer | Y | number of executors |
| main_class_name | String | Y | The fully-qualified identifier or the main class that is in the main definition file |
| main_definition_file | String | Y | The main file used for the job |
| name | String | Y | The Spark job name |
| spark_pool_name | String | Y | The Spark pool name |
| workspace_name | String | Y | The Synapse workspace name |
| subscription_id | String | Y | The id of Azure subscription where the Synapse workspace is in |
| resource_group | String | Y | The Azure resource group where the Synapse workspace is in |
| command_line_arguments | String | N | The command line arguments for the job |
| configuration | String | N | The configuration of Spark job |
| language | String | N | The language of Spark job. Accepted values: CSharp, PySpark, Python, Scala, Spark, SparkDotNet |
| reference_files | String | N | Additional files used for reference in the main definition file |
| tags | String | N | Space-separated tags: key[=value] [key[=value] ...] |
| spark_pool_config_file | String | N | Path of the spark pool configuration yaml file. Default value is ./src/spark_pool_config.yaml |
| wait_until_job_finished | Bool | N | Whether wait for the job completion. Default value is True |
| waiting_timeout_in_seconds | Integer | N | The waiting timeout in seconds. Default value is 3600 |

### Authentication parameters
| Name | Type | Required | Description |
| -------------------------- | ------ | -------- | ----------------------------------------|
| service_principal_id | String | Y | The service principal client id |
| service_principal_password | String | Y | The service principal password/secret |
| tenant_id | String | Y | The Azure tenant id for the service principal |

## Prerequisites
- [Create an AKS cluster](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough-portal).
- [Install Kubeflow on AKS](https://www.kubeflow.org/docs/azure/).
- [Create AAD service principal](https://docs.microsoft.com/cli/azure/create-an-azure-service-principal-azure-cli#password-based-authentication).
- Create a Azure Synapse Workspace and grant following permissions to the service principal:
  - Azure Owner or Contributor for the Synapse workspace (Azure RBAC)
  - Synapse Apache Spark Administrator in Synapse RBAC
  - Storage blob data owner for the attached ADSL account (storing job 
  definition file)

  See [Synapse RBAC Roles documentation](https://docs.microsoft.com/en-us/azure/synapse-analytics/security/synapse-workspace-understand-what-role-you-need) for more details about Synapse RBAC. See [here](https://docs.microsoft.com/azure/role-based-access-control/role-assignments-cli#step-4-add-role-assignment) about how to add a role assignment.
- [Create a Azure Container Registry](https://docs.microsoft.com/en-us/azure/container-registry/container-registry-get-started-portal) and grant access to the AKS cluster by running:
```shell
az login
az aks update -n <aks-name> -g <aks-resource-group-name> --attach-acr <acr-name>
```
- Create a secret for service principal authentication in AKS by running:
```shell
kubectl create secret generic azcreds \
  --from-literal=AZ_TENANT_ID='<tenant-id>' \
  --from-literal=AZ_CLIENT_ID='<client-id>' \
  --from-literal=AZ_CLIENT_SECRET='<client-secret>' \
  --from-literal=AZ_SUBSCRIPTION_ID='<azure-subscription-id>' \
  -n kubeflow
```
  See [here](https://docs.microsoft.com/azure/aks/cluster-container-registry-integration) for more details about AKS and ACI integration.

## Usage
### Step 1. Build the docker image and upload to Azure Container Registry
First, login to your Azure container registry by running:
```shell
az login
sudo az acr login -n <acr-name>
``` 
You can run the following command to build and upload the image
```shell
docker build . -t <acr-name>.azurecr.io/deploy/<image-name>:latest
docker push <acr-name>.azurecr.io/deploy/<image-name>:latest
``` 

> **NOTE**: You can also use container registries other than Azure Container Registry. Please follow the instruction from the service provider to configure integration with Kubeflow and push the images.

### Step 2. Update the parameters
In sample.py, we set *main_definition_file* and *command_line_arguments* as pipeline input parameters. You can update the pipeline input parameters and component parameters as needed before building the sample pipeline.

If you need the Azure Synapse Spark job component to create new Spark pool, make sure you review and update the [Spark pool configuration file](./src/spark_pool_config.yaml). 

### Step 3. Build the sample pipeline using sample.py
First install kfp package
```shell
pip install kfp
```

Run the sample.py script to build and compile sample pipeline
```shell
python sample.py --image_name <image-name> --image_repo_name <acr-name>
```

### Step 4. Download the .gz file and upload the pipeline in Kubeflow UI