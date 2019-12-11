#! /bin/bash

#
# Argument parsing and validation
#
KUBEFLOW_HOST=""
PIPELINE_NAME=""
PIPELINE_PARAMS="[]"

function show_usage() {
    echo "compile_upload_run.sh"
    echo
    echo -e "\t--kubeflow-host\t(Required) Kubeflow server address (e.g. http://localhost:8080)"
    echo -e "\t--pipeline-name\t(Required) Name of the pipeline (e.g. databricks_operator_run_pipeline)"
    echo -e "\t--pipeline-name\t(Optional) Pipeline parameters (e.g. '[{\"name\":\"run_name\",\"value\":\"test-run\"},{\"name\":\"parameter\",\"value\":\"10\"}]')"
}

while [[ $# -gt 0 ]]
do
    case "$1" in 
        --kubeflow-host)
            KUBEFLOW_HOST="$2"
            shift 2
            ;;
        --pipeline-name)
            PIPELINE_NAME="$2"
            shift 2
            ;;
        --pipeline-params)
            PIPELINE_PARAMS="$2"
            shift 2
            ;;
        *)
            echo "Unexpected '$1'"
            show_usage
            exit 1
            ;;
    esac
done

if [ -z $KUBEFLOW_HOST ]; then
    echo "kubeflow-host not specified"
    echo
    show_usage
    exit 1
fi

if [ -z $PIPELINE_NAME ]; then
    echo "pipeline-name not specified"
    echo
    show_usage
    exit 1
fi

#
# Main script start
#

echo "Compiling pipeline '$PIPELINE_NAME.py'..."
# To test with the local package: pip install --upgrade ../kfp-azure-databricks
pip install -e "git+https://github.com/kubeflow/pipelines#egg=kfp-azure-databricks&subdirectory=samples/contrib/azure-samples/kfp-azure-databricks" --upgrade
python ./$PIPELINE_NAME.py
PIPELINE_FILE="$PIPELINE_NAME.py.tar.gz"

echo "Uploading pipeline '$PIPELINE_FILE' to '$KUBEFLOW_HOST'..."
UPLOAD_RESPONSE=`curl -s -X POST -F uploadfile=@$PIPELINE_FILE $KUBEFLOW_HOST/pipeline/apis/v1beta1/pipelines/upload?name=$PIPELINE_NAME`
PIPELINE_ID=$(echo $UPLOAD_RESPONSE | jq .id -r 2> /dev/null)

if [ -z $PIPELINE_ID ]; then
    echo -e "\nFailed:\n $UPLOAD_RESPONSE\n"
    exit 1
else
    echo "Uploaded with ID '$PIPELINE_ID'." 
fi

echo "Running pipeline..."
PIPELINE_RUN_NAME=`date +%s`
PIPELINE_DESCRIPTION='{"description":"","name":"'$PIPELINE_RUN_NAME'","pipeline_spec":{"parameters":'$PIPELINE_PARAMS',"pipeline_id":"'$PIPELINE_ID'"},"resource_references":[]}'
echo $PIPELINE_DESCRIPTION
RUN_RESPONSE=`curl -s $KUBEFLOW_HOST/pipeline/apis/v1beta1/runs --data "$PIPELINE_DESCRIPTION"`
PIPELINE_RUN_ID=$(echo $RUN_RESPONSE | jq .run.id -r)

if [ $PIPELINE_RUN_ID = "null" ]; then
    echo -e "\nFailed:\n$RUN_RESPONSE\n"
else
    echo "View run '$PIPELINE_RUN_NAME': $KUBEFLOW_HOST/pipeline/#/runs/details/$PIPELINE_RUN_ID"
fi

read -p "Press any key to delete the pipeline:"

echo "Deleting pipeline with ID '$PIPELINE_ID'..."

DELETE_RESPONSE=`curl -s -X DELETE $KUBEFLOW_HOST/pipeline/apis/v1beta1/pipelines/$PIPELINE_ID`
if [ "$DELETE_RESPONSE" = "{}" ]; then
    echo "We are done!"
else
    echo -e "\nFailed:\n $DELETE_RESPONSE\n"
    exit 1
fi
