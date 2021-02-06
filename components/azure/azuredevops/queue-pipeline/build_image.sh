#!/bin/bash -e
while getopts "r:" option;
    do
    case "$option" in
        r ) REGISTRY_NAME=${OPTARG};;
    esac
done
image_name=${REGISTRY_NAME}.azurecr.io/myrepo/queue_pipeline:latest # Specify the image name here
image_tag=latest
full_image_name=${image_name}:${image_tag}

cd "$(dirname "$0")" 
docker build -t "${full_image_name}" .
docker push "$full_image_name"

# Output the strict image name (which contains the sha256 image digest)
docker inspect --format="{{index .RepoDigests 0}}" "${full_image_name}"