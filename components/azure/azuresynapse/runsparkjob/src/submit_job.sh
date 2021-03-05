#!/bin/bash

# Deploy registered model to Azure Machine Learning
while getopts "s:e:c:d:n:p:w:i:r:u:g:t:a:o:l:f:q:x:y:h:" option;
    do
    case "$option" in
        s ) EXECUTOR_SIZE=${OPTARG};;
        e ) EXECUTORS=${OPTARG};;
        c ) MAIN_CLASS_NAME=${OPTARG};;
        d ) MAIN_DEFINITION_FILE=${OPTARG};;
        n ) NAME=${OPTARG};;
        p ) SPARK_POOL_NAME=${OPTARG};;
        i ) SERVICE_PRINCIPAL_ID=${OPTARG};;
        r ) SERVICE_PRINCIPAL_PASSWORD=${OPTARG};;
        u ) SUBSCRIPTION_ID=${OPTARG};;
        g ) RESOURCE_GROUP=${OPTARG};;
        w ) WORKSPACE_NAME=${OPTARG};;
        t ) TENANT_ID=${OPTARG};;
        a ) COMMAND_LINE_ARGUMENTS=${OPTARG};;
        o ) CONFIGURATION=${OPTARG};;
        l ) LANGUAGE=${OPTARG};;
        f ) REFERENCE_FILE=${OPTARG};;
        q ) TAGS=${OPTARG};;
        x ) SPARK_POOL_CONFIG_FILE=${OPTARG};;
        y ) WAIT_UNTIL_JOB_FINISHED=${OPTARG};;
        h ) WAITING_TIMEOUT_IN_SECONDS=${OPTARG};;
    esac
done

echo "Scheduling spark job in Synapse workspace ${WORKSPACE_NAME} and spark pool ${SPARK_POOL_NAME}."

az login --service-principal --username ${SERVICE_PRINCIPAL_ID} --password ${SERVICE_PRINCIPAL_PASSWORD} -t ${TENANT_ID}
az account set --subscription ${SUBSCRIPTION_ID}

OUTPUT=$(az synapse workspace show --name ${WORKSPACE_NAME} \
                           --resource-group ${RESOURCE_GROUP} 2>&1)

if [[ $OUTPUT == *"ResourceNotFoundError"* ]]; then
    echo "The workspace doesn't exist. Cannot schedule the job."
    exit 1
fi

# Check if the spark pool exists
OUTPUT=$(az synapse spark pool show --name ${SPARK_POOL_NAME} \
                           --resource-group ${RESOURCE_GROUP} \
                           --workspace-name ${WORKSPACE_NAME} 2>&1)

# Create spark pool if not exists
if [[ $OUTPUT == *"ResourceNotFoundError"* ]]; then
    echo "The spark pool doesn't exist, creating the spark pool!"
    while read -r line; 
        do arrIN=(${line//:/ }); declare "${arrIN[0]}"="${arrIN[1]}";
    done < ${SPARK_POOL_CONFIG_FILE}
    command="az synapse spark pool create --name ${SPARK_POOL_NAME} \
                                --node-count ${SPARK_POOL_NODE_COUNT} \
                                --node-size ${SPARK_POOL_NODE_SIZE} \
                                --resource-group ${RESOURCE_GROUP} \
                                --workspace-name ${WORKSPACE_NAME} \
                                --spark-version ${SPARK_POOL_SPARK_VERSION} \
                                --enable-auto-pause ${SPARK_POOL_ENABLE_AUTO_PAUSE} \
                                --enable-auto-scale ${SPARK_POOL_ENABLE_AUTO_SCALE} \
                                --delay ${SPARK_POOL_DELAY} \
                                --max-node-count ${SPARK_POOL_MAX_NODE_COUNT} \
                                --min-node-count ${SPARK_POOL_MIN_NODE_COUNT} \
                                --node-size-family ${SPARK_POOL_NODE_SIZE_FAMILY} \
                                --library-requirements-file ${SPARK_POOL_LIBRARY_REQUIREMENTS_FILE} \
                                --default-spark-log-folder ${SPARK_POOL_DEFAULT_SPARK_LOG_FOLDER} \
                                --spark-events-folder ${SPARK_POOL_SPARK_EVENTS_FOLDER} \
                                --tags ${SPARK_POOL_TAGS}"

    OUTPUT=$(eval $command)
    pool_name=$( echo $OUTPUT | jq ".name")

    # If cannot get Livy id from the output, return -1 and the error output
    if [[ "$pool_name" == "" ]]; then
        echo "Failed to create spark cluster. See errors above for more details."
        exit 1
    else
        echo "Created spark pool ${SPARK_POOL_NAME} in workspace ${WORKSPACE_NAME}."
    fi
else
    echo "Found spark pool ${SPARK_POOL_NAME} in workspace ${WORKSPACE_NAME}."
fi

# Run az synapse spark job submit to submit spark job
command="az synapse spark job submit --executor-size ${EXECUTOR_SIZE} \
                            --executors ${EXECUTORS} \
                            --main-definition-file ${MAIN_DEFINITION_FILE} \
                            --name ${NAME} \
                            --spark-pool-name ${SPARK_POOL_NAME} \
                            --workspace-name ${WORKSPACE_NAME}"

if [[ "$MAIN_CLASS_NAME" == "" ]]; then 
    command="${command} --main-class-name \"\""
else
    command="${command} --main-class-name ${MAIN_CLASS_NAME}"
fi

if [[ "$COMMAND_LINE_ARGUMENTS" != "" ]]; then 
    command="${command} --command-line-arguments ${COMMAND_LINE_ARGUMENTS}"
fi

if [[ "$CONFIGURATION" != "" ]]; then 
    command="${command} --configuration ${CONFIGURATION}"
fi

if [[ "$LANGUAGE" != "" ]]; then 
    command="${command} --language ${LANGUAGE}"
fi

if [[ "$REFERENCE_FILE" != "" ]]; then 
    command="${command} --reference-files ${REFERENCE_FILE}"
fi

if [[ "$TAGS" != "" ]]; then 
    command="${command} --tags ${TAGS}"
fi

OUTPUT=$(eval $command)

# Try the get the Livy id
job_id=$( echo $OUTPUT | jq ".id")

# If cannot get Livy id from the output, return -1 and the error output
if [[ "$job_id" == "" ]]; then
    echo "Failed to schedule spark job. See errors above for more details."
    exit 1
fi

echo "Spark job with Livy id ${job_id} is submitted in spark pool ${SPARK_POOL_NAME}."

# Get job result, return if the result is not "Uncertain" or reachs the timeout
if [[ "$WAIT_UNTIL_JOB_FINISHED" == "True" ]]; then
    result="\"Uncertain\""
    iterations=$(expr ${WAITING_TIMEOUT_IN_SECONDS} / 10)
    for i in $(seq 1 $iterations); do
        sleep 10
        command="az synapse spark job show --livy-id ${job_id} --workspace-name ${WORKSPACE_NAME} --spark-pool-name ${SPARK_POOL_NAME}"
        OUTPUT=$(eval $command)
        result=$( echo $OUTPUT | jq ".result")

        if [[ "${result}" != "\"Uncertain\"" ]]; then
            echo "Job finished with status ${result}!";
            echo "Details: ${OUTPUT}"
            exit 0;
        fi
        waiting_time=$(expr ${i} \* 10)
        echo "Job ${job_id} is still running! Total waiting time is ${waiting_time}s.";
    done
    echo "Job ${job_id} is still running! But reached timeout ${WAITING_TIMEOUT_IN_SECONDS}s.";
    echo "Details: ${OUTPUT}"
    exit 0;

fi

echo "Job ${job_id} is submitted!";
echo "Details: ${OUTPUT}"
exit 0;