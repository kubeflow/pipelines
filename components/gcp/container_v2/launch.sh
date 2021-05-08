#!/bin/bash
set -x 


NAME=$(curl \
 -X POST \
 -H "Authorization: Bearer ${GCLOUD_AUTH_TOKEN}" \
 -H "Content-Type: application/json" \
  https://${ENDPOINT}/ui/projects/${PROJECT_ID}/locations/${REGION}/trainingPipelines \
 -d $1 |  jq -r '.name')


get() {
	curl \
	 -H "Authorization: Bearer ${GCLOUD_AUTH_TOKEN}" \
	 -H "Content-Type: application/json" \
	 https://${ENDPOINT}/v1/$NAME
}

STATE=$(get | jq -r '.state')
while [ $STATE != "PIPELINE_STATE_SUCCEEDED" ]; do
	echo "job $NAME running..."
	sleep 5
	STATE=$(get | jq -r '.state')
done
