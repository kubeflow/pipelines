#!/bin/bash -e
while getopts "r:" option;
    do
    case "$option" in
        r ) REGISTRY_NAME=${OPTARG};;
    esac
done
image_name=${REGISTRY_NAME}.azurecr.io/myrepo/aml-register-model  # Specify the image name here
image_tag=latest
full_image_name=${image_name}:${image_tag}

cd "$(dirname "$0")" 
docker build -t "${full_image_name}" .
docker push "$full_image_name"