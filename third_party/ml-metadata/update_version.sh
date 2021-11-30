#!/bin/bash

set -e

echo "Usage: update kubeflow/pipelines/third_party/ml-metadata/VERSION to new MLMD version tag by"
echo '`echo -n "\$VERSION" > VERSION` first, then run this script.'
echo "Please use the above command to make sure the file doesn't have extra"
echo "line endings."
echo "MLMD version page: https://github.com/google/ml-metadata/releases"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
TAG_NAME="$(cat ${DIR}/VERSION)"
REPO_ROOT="${DIR}/../.."

cd "${REPO_ROOT}"
echo "${PWD}"
echo "${TAG_NAME}"

image_files=( "${REPO_ROOT}/.cloudbuild.yaml" \
              "${REPO_ROOT}/.release.cloudbuild.yaml" \
              "${REPO_ROOT}/manifests/kustomize/base/metadata/base/metadata-grpc-deployment.yaml" \
              "${REPO_ROOT}/test/tag_for_hosted.sh" \
              "${REPO_ROOT}/v2/Makefile" \
              )
for i in "${image_files[@]}"
do
    echo "Replacing ${i}"
    sed -i.bak -r "s/gcr.io\/tfx-oss-public\/ml_metadata_store_server\:[0-9.a-z-]+/gcr.io\/tfx-oss-public\/ml_metadata_store_server\:${TAG_NAME}/g" "${i}"
done

requirement_files=( "${REPO_ROOT}/backend/metadata_writer/requirements.in" \
                    "${REPO_ROOT}/v2/test/requirements.txt"
                    )
for i in "${requirement_files[@]}"
do
    echo "Replacing ${i}"
    sed -i.bak -r "s/ml-metadata==[0-9.a-z-]+/ml-metadata==${TAG_NAME}/g" "${i}"
done
