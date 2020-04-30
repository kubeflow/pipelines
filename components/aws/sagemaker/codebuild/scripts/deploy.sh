#!/usr/bin/env bash

set -e

REMOTE_REPOSITORY="amazon/aws-sagemaker-kfp-components"
DRYRUN=true

while getopts ":d:" opt; do
	case ${opt} in
		d)
			if [[ "${OPTARG}" = "false" ]]; then
				DRYRUN=false
			else
				DRYRUN=true
			fi
			;;
	esac
done

function docker_tag_exists() {
    curl --silent -f -lSL https://index.docker.io/v1/repositories/$1/tags/$2 > /dev/null 2> /dev/null
}

# Check version does not already exist
VERSION_LICENSE_FILE="THIRD-PARTY-LICENSES.txt"
LOCAL_VERSION="$(cat ${VERSION_LICENSE_FILE} | head -n1 | grep -Po '(?<=version )\d.\d.\d')"

if [ -z "$LOCAL_VERSION" ]; then
  >&2 echo "Could not find version inside ${VERSION_LICENSE_FILE} file."
  exit 1
fi

echo "Deploying version ${LOCAL_VERSION}"

if docker_tag_exists "$REMOTE_REPOSITORY" "$LOCAL_VERSION"; then
  >&2 echo "Tag ${REMOTE_REPOSITORY}:${LOCAL_VERSION} already exists. Cannot overwrite an existing image."
  exit 1
fi

# Build the image
DOCKERHUB_TAG="${REMOTE_REPOSITORY}:${LOCAL_VERSION}"
docker build . -f Dockerfile -t "${DOCKERHUB_TAG}"

# Push to the remote repository
if [ "${DRYRUN}" == "false" ]; then
  docker push "${DOCKERHUB_TAG}"
  echo "Successfully pushed tag ${DOCKERHUB_TAG} to Docker Hub"
else
  echo "Dry run detected. Not pushing image."
fi