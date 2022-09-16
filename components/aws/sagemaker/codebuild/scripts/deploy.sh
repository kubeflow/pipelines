#!/usr/bin/env bash
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

REMOTE_REPOSITORY="public.ecr.aws/kubeflow-on-aws/aws-sagemaker-kfp-components"
DRYRUN="true"
FULL_VERSION_TAG=""
BUILD_VERSION=""
DOCKER_CONFIG_PATH=${DOCKER_CONFIG_PATH:-"/root/.docker"}

while getopts ":d:v:b:" opt; do
	case ${opt} in
		d)
			if [[ "${OPTARG}" = "false" ]]; then
				DRYRUN="false"
			else
				DRYRUN="true"
			fi
			;;
		v)
			FULL_VERSION_TAG="${OPTARG}"
			;;
		b)
			BUILD_VERSION="${OPTARG}"
			;;
	esac
done

# Check that build version is not empty
if [ -z "$BUILD_VERSION" ]; then
  >&2 echo "BUILD_VERSION is required, please provide the variable in codebuild or -b <BUILD_VERSION> if running locally"
  exit 1
fi

function docker_tag_exists() {
    curl --silent -f -lSL https://index.docker.io/v1/repositories/$1/tags/$2 > /dev/null 2> /dev/null
}

if [[ ! -z "${FULL_VERSION_TAG}" && ! "${FULL_VERSION_TAG}" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
	>&2 echo "Version tag does not match SEMVER style (X.Y.Z)"
	exit 1
fi

# Check version does not already exist
if [ "${BUILD_VERSION}" == "v2" ]; then
	VERSION_LICENSE_FILE="THIRD-PARTY-LICENSES.v2.txt"
else
	VERSION_LICENSE_FILE="THIRD-PARTY-LICENSES.txt"
fi

if [[ -z "${FULL_VERSION_TAG}" ]]; then
	FULL_VERSION_TAG="$(cat ${VERSION_LICENSE_FILE} | head -n1 | grep -Po '(?<=version )\d.\d.\d')"
fi

if [ -z "$FULL_VERSION_TAG" ]; then
  >&2 echo "Could not find version inside ${VERSION_LICENSE_FILE} file."
  exit 1
fi

echo "Deploying version ${FULL_VERSION_TAG}"

# if docker_tag_exists "$REMOTE_REPOSITORY" "$FULL_VERSION_TAG"; then
#   >&2 echo "Tag ${REMOTE_REPOSITORY}:${FULL_VERSION_TAG} already exists. Cannot overwrite an existing image."
#   exit 1
# fi

# Build the image
FULL_VERSION_IMAGE="${REMOTE_REPOSITORY}:${FULL_VERSION_TAG}"

if [ "${BUILD_VERSION}" == "v2" ]; then
	echo "Building V2 image"
	docker build . -f v2.Dockerfile -t "${FULL_VERSION_IMAGE}"
else
	echo "Building V1 image"
	docker build . -f Dockerfile -t "${FULL_VERSION_IMAGE}"
fi

# Get the minor and major versions
[[ $FULL_VERSION_TAG =~ ^[0-9]+\.[0-9]+ ]] && MINOR_VERSION_IMAGE="${REMOTE_REPOSITORY}:${BASH_REMATCH[0]}"
[[ $FULL_VERSION_TAG =~ ^[0-9]+ ]] && MAJOR_VERSION_IMAGE="${REMOTE_REPOSITORY}:${BASH_REMATCH[0]}"

# Re-tag the image with major and minor versions
docker tag "${FULL_VERSION_IMAGE}" "${MINOR_VERSION_IMAGE}"
echo "Tagged image with ${MINOR_VERSION_IMAGE}"
docker tag "${FULL_VERSION_IMAGE}" "${MAJOR_VERSION_IMAGE}"
echo "Tagged image with ${MAJOR_VERSION_IMAGE}"

# Push to the remote repository
if [ "${DRYRUN}" == "false" ]; then
  docker --config "$DOCKER_CONFIG_PATH" push "${FULL_VERSION_IMAGE}"
  echo "Successfully pushed tag ${FULL_VERSION_IMAGE} to Docker Hub"

	docker --config "$DOCKER_CONFIG_PATH" push "${MINOR_VERSION_IMAGE}"
  echo "Successfully pushed tag ${MINOR_VERSION_IMAGE} to Docker Hub"

	docker --config "$DOCKER_CONFIG_PATH" push "${MAJOR_VERSION_IMAGE}"
  echo "Successfully pushed tag ${MAJOR_VERSION_IMAGE} to Docker Hub"
else
  echo "Dry run detected. Not pushing images."
fi