#!/bin/sh

# Deploy registered model to Azure Machine Learning
while getopts "n:m:i:d:s:p:u:r:w:t:" option;
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
    esac
done
az login --service-principal --username ${SERVICE_PRINCIPAL_ID} --password ${SERVICE_PRINCIPAL_PASSWORD} -t ${TENANT_ID}
az ml model deploy -n ${DEPLOYMENT_NAME} -m ${MODEL_NAME} --ic ${INFERENCE_CONFIG} --dc ${DEPLOYMENTCONFIG} -w ${WORKSPACE} -g ${RESOURCE_GROUP} --overwrite -v