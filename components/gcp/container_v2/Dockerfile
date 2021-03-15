FROM google/cloud-sdk:slim

RUN apt-get install -y jq

ENV GCLOUD_AUTH_TOKEN=${GCLOUD_AUTH_TOKEN}
ENV PROJECT=${PROJECT_ID}
ENV REGION=${REGION}
ENV ENDPOINT=${REGION}-aiplatform.googleapis.com
ENV JOB_NAME=${JOB_NAME}

COPY . .

CMD bash ./launch.sh 