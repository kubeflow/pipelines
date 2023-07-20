#!/bin/bash
# read the current version from environment variable
GCPC_VERSION=$1
SCRIPT_DIR=$(dirname "$0")

# check if jq is installed
if ! command -v jq &> /dev/null
then
    echo "jq could not be found"
    echo "Please install jq using the following command:"
    echo "sudo apt-get install jq"
    exit
fi

# create a new JSON object
new_version=$(cat <<EOF
{
  "version": "https://google-cloud-pipeline-components.readthedocs.io/en/google-cloud-pipeline-components-${GCPC_VERSION}",
  "title": "${GCPC_VERSION}",
  "aliases": []
}
EOF
)

# append the new version to the existing JSON file
jq ". = [$new_version] + ." $SCRIPT_DIR/source/versions.json > $SCRIPT_DIR/temp.json && mv $SCRIPT_DIR/temp.json $SCRIPT_DIR/source/versions.json
