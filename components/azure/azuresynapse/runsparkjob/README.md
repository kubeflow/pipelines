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
- Create an AKS cluster
- Install Kubeflow on AKS following [instructions for deploying Kubeflow on Azure](https://www.kubeflow.org/docs/azure/).
- Create a Azure Synapse Workspace and grant permissions (TODO: document the min privilege)
- Create a Azure Container Registry and grant access to the AKS cluster (TODO: document steps)
- Create a secret for service principal authentication (TODO: document details)

## Usage
### Step 1. Build the docker image and upload to Azure Container Registry
You can run the following command to build and upload the image
```shell
docker build . -t {your_ACR_name}.azurecr.io/deploy/{your_image_name}:latest
docker push {your_ACR_name}.azurecr.io/deploy/{your_image_name}:latest
``` 

### Step 2. Build the sample pipeline using sample.py
First install kfp package
```shell
pip install kfp
```

Run the sample.py script to build and compile sample pipeline
```shell
python sample.py --image_name <your-image-name> --image_repo_name <your-acr-name>
```

### Step 3. Download the .gz file and upload the pipeline in Kubeflow UI