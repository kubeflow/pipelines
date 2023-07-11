#!/bin/bash

# replace with your own project
# do not set to ml-pipeline
PROJECT_ID=<my-project-name>

curdir=$(pwd)
g4d

alias copybara='/google/bin/releases/copybara/public/copybara/copybara'
COPYBARA_OUTPUT_DIR=$(mktemp -d)
copybara third_party/py/google_cloud_pipeline_components/google/copy.bara.sky folder_to_folder .. --folder-dir=$COPYBARA_OUTPUT_DIR --ignore-noop
cd $COPYBARA_OUTPUT_DIR/google_cloud_pipeline_components/container
gcloud --project $PROJECT_ID builds submit --config cloudbuild.yaml

cd $curdir
