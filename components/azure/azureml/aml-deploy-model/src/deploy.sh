#!/bin/sh

# Deploy registered model to Azure Machine Learning
while getopts "n:m:i:d:s:p:u:r:w:t:o:" option;
    do
    case "$option" in
        n ) DEPLOYMENT_NAME=${OPTARG};;
        m ) MODEL_NAME=${OPTARG};;
        i ) INFERENCE_CONFIG=${OPTARG};;
        d ) DEPLOYMENTCONFIG=${OPTARG};;
        s ) SERVICE_PRINCIPAL_ID=${OPTARG};;
        p ) SERVICE_PRINCIPAL_PASSWORD=${OPTARG};;
        u ) SUBSCRIPTION_ID=${OPTARG};;
        r ) RESOURCE_GROUP=${OPTARG};;
        w ) WORKSPACE=${OPTARG};;
        t ) TENANT_ID=${OPTARG};;
        o ) OUTPUT_CONFIG_PATH=${OPTARG};;
    esac
done
az login --service-principal --username ${SERVICE_PRINCIPAL_ID} --password ${SERVICE_PRINCIPAL_PASSWORD} -t ${TENANT_ID}
az ml model deploy -n ${DEPLOYMENT_NAME} -m ${MODEL_NAME} --ic ${INFERENCE_CONFIG} --dc ${DEPLOYMENTCONFIG} -w ${WORKSPACE} -g ${RESOURCE_GROUP} --overwrite -v

# write the web-service description to output folder
parentdir="$(dirname "$OUTPUT_CONFIG_PATH")"
if [ -d "$parentdir" ];
then
    echo Found The directory ${parentdir}.
else
    echo Parent directory did not exist, creating parent directory.
    mkdir -p ${parentdir}
fi
az ml service show -n ${DEPLOYMENT_NAME} --resource-group ${RESOURCE_GROUP} --workspace-name ${WORKSPACE} > ${OUTPUT_CONFIG_PATH}
