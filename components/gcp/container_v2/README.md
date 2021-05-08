 docker build -t gcr.io/managed-pipeline-test/launcher:v1 .


JOB_NAME="single_node_python_package_$(date +%Y%m%d%H%M)"
PROJECT_ID=managed-pipeline-test
REGION=us-central1
ENDPOINT=${REGION}-aiplatform.googleapis.com
docker run \
 -e GCLOUD_AUTH_TOKEN=$(gcloud auth print-access-token) \
 -e PROJECT_ID=$PROJECT_ID \
 -e REGION=$REGION \
 -e ENDPOINT=$ENDPOINT \
 -e JOB_NAME=$JOB_NAME \
 -it gcr.io/managed-pipeline-test/launcher:v1 /launch.sh '{"displayName":"'"${JOB_NAME}"'","trainingTaskDefinition":"gs://google-cloud-aiplatform/schema/trainingjob/definition/custom_task_1.0.0.yaml","trainingTaskInputs":{"workerPoolSpecs":[{"replicaCount":1,"machineSpec":{"machineType":"n1-standard-8"},"pythonPackageSpec":{"executorImageUri":"gcr.io/cloud-aiplatform/training/training-tf-cpu.2-1:latest","packageUris":["gs://'${PROJECT_ID}'_'${REGION}'/pythonpackages/tf2_trainer.tar.gz"],"pythonModule":"trainer.task","args":["--model-dir=$(AIP_MODEL_DIR)"],}}],"baseOutputDirectory":{"outputUriPrefix":"gs://'"${PROJECT_ID}"'_'"${REGION}"'/'"${JOB_NAME}"'"}},"modelToUpload":{"displayName":'"\"Custommodel'$(date +%Y%m%d%H%M)'\""',"containerSpec":{"imageUri":"gcr.io/cloud-aiplatform/prediction/tf-cpu.1-15:latest"}}}'








