#!/bin/bash

data='{"name":'\""ci-$1"\"', "code_source_url": "https://github.com/kubeflow/pipelines/tree/'"$1"'", "package_url": {"pipeline_url": "https://storage.googleapis.com/'"$3"'/'"$1"'/pipeline.zip"}, 
"resource_references": [{"key": {"id": '\""$2"\"', "type":3}, "relationship":1}]}'

version=$(curl -H "Content-Type: application/json" -X POST -d "$data" "$4"/apis/v1beta1/pipeline_versions | jq -r ".id")

# create run
rundata='{"name":'\""$1-run"\"', 
"resource_references": [{"key": {"id": '\""$version"\"', "type":4}, "relationship":2}],
"pipeline_spec":{"parameters": [{"name": "gcr_address", "value": '\""$5"\"'}]}'
echo "$rundata"
curl -H "Content-Type: application/json" -X POST -d "$rundata" "$4"/apis/v1beta1/runs