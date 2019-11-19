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
# 

###############################################################################
#
# This script performs the following steps:
# 1. Uploads local files to the cluster host (i.e. '--master').
#    The files it uploads are specified in the following parameters:
#       --files
#       --jars
#       --py-files
#       The application JAR file or python file
#    If you want to use files already on the spark cluster, you can disable
#    the uploading of files by setting operating system environment variables
#    described below. Uploaded files will be placed on the cluster at
#    <TENANT_ID>/data/<generated-UUID>/
# 2. Re-writes paths for files uploaded to the cluster.  The re-written paths
#    are used when calling submit REST API.
# 3. Gets returning spark-submit's submission ID and periodically polls for status
#    of the job using the submission ID.
# 4. When the job is FINISHED, downloads 'stdout' and 'stderr' from the
#    cluster.
# 5. Delete the job workspace folder <TENANT_ID>/data/<generated-UUID>/ on the cluster
#
# Before running this script, operating system variables must be set.
# Optional:
#    SS_APP_MAIN_UPLOAD=<true|false> # Default: 'true' application jar file is uploaded.
#    SS_FILES_UPLOAD=<true|false> # Default: 'true'. '--files' and "--py-files" files are uploaded.
#    SS_JARS_UPLOAD=<true|false> # Default: 'true'. '--jars' files are uploaded.
#    SS_LOG_ENABLE=<true|false> # Default: 'true'. Execution log is created.
#
# VCAP information needs to be made available to this program in the '--vcap'
# parameter.  The VCAP information is obtained from your BlueMix application.
# Here is one way to create a file from your VCAP:
# cat <<EOT > ~/vcap.json
# {
#    "credentials": {
#              "tenant_id": "xxxxxx",
#              "tenant_id_full": "xxxxxx",
#              "cluster_master_url": "https://x.x.x.x",
#              "instance_id": "xxxxxx",
#              "tenant_secret": "xxxxx",
#              "plan": "ibm.SparkService.PayGoPersonal"
#          }
#    }
# }
# EOT
#
# Example command to run:
#
# ./spark-submit.sh \
#    --vcap ~/vcap.json \
#    --deploy-mode cluster \
#    --class com.ibm.sparkservice.App \
#    --master https://x.x.x.x\
#    --jars /path/to/mock-library-1.0.jar,/path/to/mock-utils-1.0.jar \
#    ~/mock-app-1.0.jar
#
#
###############################################################################

invokeCommand="$(basename $0) $@"

# -- User-modifiable variables ------------------------------------------------
# To modify,  set the operating system environment variable to the desired value.

if [ -z ${SS_LOG_ENABLE} ];      then SS_LOG_ENABLE=true;  fi # Enable detailed logging
if [ -z ${SS_APP_MAIN_UPLOAD} ]; then SS_APP_MAIN_UPLOAD=true; fi # If true, copy the local application JAR or python file to the spark cluster
if [ -z ${SS_JARS_UPLOAD} ];     then SS_JARS_UPLOAD=true; fi # If true, copy the local JAR files listed in "--jars" to the spark cluster.
if [ -z ${SS_FILES_UPLOAD} ];    then SS_FILES_UPLOAD=true; fi # If true, copy the local files listed in "--files" and "--py-files" to the spark cluster.
if [ -z ${SS_POLL_INTERVAL} ];   then SS_POLL_INTERVAL=10; fi # Number of seconds until script polls spark cluster again.
if [ -z ${SS_SPARK_WORK_DIR} ];  then SS_SPARK_WORK_DIR="workdir"; fi # Work directory on spark cluster
if [ -z ${SS_DEBUG} ];           then SS_DEBUG=false; fi # Detailed debugging

# -- Set working environment variables ----------------------------------------

if [ "${SS_DEBUG}" = "true" ]
then
    set -x
fi

EXECUTION_TIMESTAMP="$(date +'%s%N')"
APP_MAIN=
app_parms=
FILES=
JARS=
PY_FILES=
CLASS=
APP_NAME=
DEPLOY_MODE=
LOG_FILE=spark-submit_${EXECUTION_TIMESTAMP}.log
MASTER=
INSTANCE_ID=
TENANT_ID=
TENANT_SECRET=
CLUSTER_MASTER_URL=
SPARK_VERSION=
submissionId=
declare -a CONF_KEY
declare -a CONF_VAL
confI=0
CHECK_STATUS=false
KILL_JOB=false
PY_APP=false
IS_JOB_ERROR=false
HEADER_REQUESTED_WITH=spark-submit
VERSION="1.0.11"

# Determine which sha command to use for UUID calculation
SHASUM_CMD=""
if hash shasum 2>/dev/null; then
    SHASUM_CMD="shasum -a 1"
elif hash sha1sum 2>/dev/null; then
    SHASUM_CMD="sha1sum"
else
    printf "\nCould not find \"sha1sum\" or equivalent command on system.  Aborting.\n"
    exit -1
fi

# UUID=$(openssl rand -base64 64 | ${SHASUM_CMD} | awk '{print $1}')
SERVER_SUB_DIR="${SS_SPARK_WORK_DIR}/tmp"

uploadList=" "

# =============================================================================
# -- Functions ----------------------------------------------------------------
# =============================================================================

printUsage()
{
   printf "\nUsage:"
   printf "\n     spark-submit.sh --vcap <vcap-file> [options] <app jar | python file> [app arguments]"
   printf "\n     spark-submit.sh --master [cluster-master-url] --conf 'PROP=VALUE' [options] <app jar | python file> [app arguments]"
   printf "\n     spark-submit.sh --vcap <vcap-file> --kill [submission ID] "
   printf "\n     spark-submit.sh --vcap <vcap-file> --status [submission ID] "
   printf "\n     spark-submit.sh --kill [submission ID] --master [cluster-master-url] --conf 'PROP=VALUE' "
   printf "\n     spark-submit.sh --status [submission ID] --master [cluster-master-url] --conf 'PROP=VALUE' "
   printf "\n     spark-submit.sh --help  "
   printf "\n     spark-submit.sh --version  "
   printf "\n\n     vcap-file:                  json format file that contains spark service credentials, "
   printf "\n                                 including cluster_master_url, tenant_id, instance_id, and tenant_secret"
   printf "\n     cluster_master_url:         The value of 'cluster_master_url' on the service credentials page"
   printf "\n\n options:"
   printf "\n     --help                      Print out usage information."
   printf "\n     --version                   Print out the version of spark-submit.sh"
   printf "\n     --master MASTER_URL         MASTER_URL is the value of 'cluster-master-url' from spark service instance credentials"
   printf "\n     --deploy-mode DEPLOY_MODE   DEPLOY_MODE must be 'cluster'"
   printf "\n     --class CLASS_NAME          Your application's main class (for Java / Scala apps)."
   printf "\n     --name NAME                 A name of your application."
   printf "\n     --jars JARS                 Comma-separated list of local jars to include on the driver and executor classpaths."
   printf "\n     --files FILES               Comma-separated list of files to be placed in the working directory of each executor."
   printf "\n     --conf PROP=VALUE           Arbitrary Spark configuration property. The values of tenant_id, instance_id, tenant_secret, and spark_version can be passed"
   printf "\n     --py-files PY_FILES         Comma-separated list of .zip, .egg, or .py files to place on the PYTHONPATH for Python apps."
   printf "\n\n     --kill SUBMISSION_ID        If given, kills the driver specified."
   printf "\n     --status SUBMISSION_ID      If given, requests the status of the driver specified."
   printf "\n"
   exit 0
}

printVersion()
{
   printf "spark-submit.sh  VERSION : '${VERSION}'\n"
   exit 0
}

logMessage()
{
    if [ "${SS_LOG_ENABLE}" = "true" ]
    then
        printf "$1" >> ${LOG_FILE}
    else
        printf "$1"
    fi
}

logFile()
{
    logMessage "\nContents of $1:\n"
    if [ "${SS_LOG_ENABLE}" = "true" ]
    then
        cat "$1" >> ${LOG_FILE}
    else
        cat "$1"
    fi
}

console()
{
    local output_line=$1
    printf "${output_line}"
    logMessage "${output_line}"
}

endScript()
{
    console "\nSubmission complete.\n"
    console "spark-submit log file: ${LOG_FILE}\n"
}

endScriptWithCommands()
{
    if [ -n "${submissionId}" ]
    then
       console "Job may still be running.\n"
       console "To poll for job status, run the following command:\n"
       if [ ! -z "${VCAP_FILE}" ]
       then
           console "\"spark-submit.sh --status ${submissionId} --vcap ${VCAP_FILE} \" \n"
       else
           console "\"spark-submit.sh --status ${submissionId} --master ${MASTER} --conf 'spark.service.tenant_id=${TENANT_ID}' --conf 'spark.service.tenant_secret=${TENANT_SECRET}' --conf 'spark.service.instance_id=${INSTANCE_ID}'\" \n"
       fi
       console "After the job is done, run the following command to download stderr and stdout of the job to local:\n"
       console "\"curl ${SS_CURL_OPTIONS} -X GET $(get_http_authentication) -H '$(get_http_instance_id)' https://${HOSTNAME}/tenant/data/${SS_SPARK_WORK_DIR}/${submissionId}/stdout > stdout\" \n"
       console "\"curl ${SS_CURL_OPTIONS} -X GET $(get_http_authentication) -H '$(get_http_instance_id)' https://${HOSTNAME}/tenant/data/${SS_SPARK_WORK_DIR}/${submissionId}/stderr > stderr\" \n"
       # console "\"curl ${SS_CURL_OPTIONS} -X GET $(get_http_authentication) -H '$(get_http_instance_id)' https://${HOSTNAME}/tenant/data/${SS_SPARK_WORK_DIR}/${submissionId}/model.zip > model.zip\" \n"

       if [ "${SS_APP_MAIN_UPLOAD}" = "true" ] || [ "${SS_JARS_UPLOAD}" = "true" ] || [ "${SS_FILES_UPLOAD}" = "true" ]
       then
          console "After the job is done, we recommend to run the following command to clean the job workspace: \n"
          console "\"curl ${SS_CURL_OPTIONS} -X DELETE $(get_http_authentication) -H '$(get_http_instance_id)' https://${HOSTNAME}/tenant/data/${SERVER_SUB_DIR}\" \n"
       fi
    fi
    console "spark-submit log file: ${LOG_FILE}\n"
}


base64Encoder()
{
    encoded="`printf $1 | base64`"
    echo "${encoded}"
}

get_from_vcap()
{
    local vcapFilePath=$1
    local vcapKey=$2
    # Handle dos2unix issues.
    local ctrl_m=$(printf '\015')
    echo `grep ${vcapKey}\" ${vcapFilePath} | awk '{print $2}' | sed 's/\"//g' | sed 's/\,//g' | sed "s/${ctrl_m}//g"`
}

get_hostname_from_url()
{
    local url=$1
    echo ${url} | sed -n 's/[^:]*:\/\/\([^:]*\)[:]*.*/\1/p'
}

get_http_authentication()
{
    echo "-u ${TENANT_ID}:${TENANT_SECRET}"
}

get_http_instance_id()
{
    echo "X-Spark-service-instance-id: ${INSTANCE_ID}"
}

get_requested_with_header()
{
    echo "X-Requested-With: ${HEADER_REQUESTED_WITH}"
}

display_master_url_err_msg()
{
    console "ERROR: master URL is missing. Use either --master or --vcap option. Run with --help for usage information.\n"
}

display_err_msg()
{
    console "ERROR: $1 is missing. Use either --vcap or --conf option. Run with --help for usage information.\n"
}

display_err_msg_spark_version()
{
    console "ERROR: Spark service configuration \"spark.service.spark_version\" is missing. Specify the Spark version using  --conf option as \"--conf spark.service.spark_version=<spark version>\". Run with --help for usage information.\n"
}

get_conf_options()
{
  logMessage "\nValues passed with --conf option...\n\n"
  for ((i=0; i<${#CONF_KEY[@]}; ++i))
    do
        conf_key=${CONF_KEY[${i}]}
        conf_val=${CONF_VAL[${i}]}
        logMessage "\t${conf_key} : ${conf_val} \n"
        if [[ "${conf_key}" == "spark.service.tenant_id" ]]; then
           if [[ -z "${TENANT_ID}" ]]; then
              TENANT_ID="${conf_val}"
           elif [[ "${conf_val}" != "${TENANT_ID}" ]]; then #if tenant_id is specified in vcap file and in --conf option, and they are not same, then use the one from --conf option.
              TENANT_ID="${conf_val}"
              logMessage "WARN: configuration \"${conf_key}\" : \"${conf_val}\" does not match with tenant_id in ${VCAP_FILE} file. Using \"${conf_key}\"'s value.\n"
           fi
        fi

        if [[ "${conf_key}" == "spark.service.instance_id" ]]; then
           if [[ -z "${INSTANCE_ID}" ]]; then
              INSTANCE_ID="${conf_val}"
           elif [[ "${conf_val}" != "${INSTANCE_ID}" ]]; then #if instance_id is specified in vcap file and in --conf option, and they are not same, then use the one from --conf option.
              INSTANCE_ID="${conf_val}"
              logMessage "WARN: configuration \"${conf_key}\" : \"${conf_val}\" does not match with instance_id in ${VCAP_FILE} file. Using \"${conf_key}\"'s value. \n"
           fi
        fi

        if [[ "${conf_key}" == "spark.service.tenant_secret" ]]; then
           if [[ -z "${TENANT_SECRET}" ]]; then
              TENANT_SECRET="${conf_val}"
           elif [[ "${conf_val}" != "${TENANT_SECRET}" ]]; then #if tenant_secret is specified in vcap file and in --conf option, and they are not same, then use the one from --conf option.
              TENANT_SECRET="${conf_val}"
              logMessage "WARN: configuration \"${conf_key}\" : \"${conf_val}\" does not match with tenant_secret in ${VCAP_FILE} file. Using \"${conf_key}\"'s value. \n"
           fi
        fi

        if [[ "${conf_key}" == "spark.service.spark_version" ]]; then
           SPARK_VERSION="${conf_val}"
        fi
    done
}

local2server()
{
    local localPath=$1
    local serverPath=$2
    local cmd="curl ${SS_CURL_OPTIONS} -X PUT $(get_http_authentication) -H '$(get_http_instance_id)' --data-binary '@${localPath}' https://${HOSTNAME}/tenant/data/${serverPath}"
    console "\nUploading ${localPath}\n"
    logMessage "local2server command: ${cmd}\n"
    local result=$(eval "${cmd}")
    uploadList+="$(fileNameFromPath ${localPath})"
    logMessage "local2server result: ${result}\n"
}

deleteFolderOnServer()
{
    local serverDir=$1
    local cmd="curl ${SS_CURL_OPTIONS} -X DELETE $(get_http_authentication) -H '$(get_http_instance_id)' https://${HOSTNAME}/tenant/data/${serverDir}"
    console "\nDeleting workspace on server\n"
    logMessage "deleteFolderOnServer command: ${cmd}\n"
    local result=$(eval "${cmd}")
    logMessage "deleteFolderOnServer result: ${result}\n"
}

local2server_list()
{
    local localFiles=$1
    local files=$2
    OIFS=${IFS}
    IFS=","
    localFileArray=(${localFiles})
    fileArray=(${files})
    IFS=${OIFS}

    for ((i=0; i<${#localFileArray[@]}; ++i))
    do
        local2server ${localFileArray[${i}]} ${fileArray[${i}]}
    done
}

fileNameFromPath()
{
    local path=$1
    local fileName="`echo ${path} | awk 'BEGIN{FS="/"}{print $NF}'`"
    echo "${fileName}"
}

fileNameFromPath_list()
{
    local paths=$1
    OIFS=${IFS}
    IFS=","
    pathArray=(${paths})
    IFS=${OIFS}
    local fileNames=
    for ((i=0; i<${#pathArray[@]}; ++i))
    do
        local fileName=$(fileNameFromPath ${pathArray[${i}]})
        if [ -z "${fileNames}" ]
        then
            fileNames="${fileName}"
        else
            fileNames="${fileNames},${fileName}"
        fi
    done
    echo "${fileNames}"
}

convert2serverPath()
{
    local fileName=$(fileNameFromPath $1)
    local serverFile="${SERVER_SUB_DIR}/${fileName}"
    echo "${serverFile}"
}

convert2serverPath_list()
{
    local localFiles=$1
    OIFS=${IFS}
    IFS=","
    localFileArray=(${localFiles})
    IFS=${OIFS}
    local serverFiles=
    for ((i=0; i<${#localFileArray[@]}; ++i))
    do
        local serverFile=$(convert2serverPath ${localFileArray[${i}]})
        if [ -z "${serverFiles}" ]
        then
            serverFiles="${serverFile}"
        else
            serverFiles="${serverFiles},${serverFile}"
        fi
    done
    echo "${serverFiles}"
}

convert2submitPath()
{
    local serverFile=$1
    echo "${PREFIX_SERVER_PATH}/${serverFile}"
}

convert2submitPath_list()
{
    local serverFiles=$1
    OIFS=${IFS}
    IFS=","
    serverFileArray=(${serverFiles})
    IFS=${OIFS}
    local submitPaths=
    for ((i=0; i<${#serverFileArray[@]}; ++i))
    do
        local submitPath=$(convert2submitPath ${serverFileArray[${i}]})
        if [ -z "${submitPaths}" ]
        then
            submitPaths="${submitPath}"
        else
            submitPaths="${submitPaths},${submitPath}"
        fi
    done
    echo "${submitPaths}"
}

server2local()
{
    local serverPath=$1
    local localPath=$2
    local cmd="curl ${SS_CURL_OPTIONS} -X GET $(get_http_authentication) -H '$(get_http_instance_id)' -D '${localPath}.header' https://${HOSTNAME}/tenant/data/${serverPath}"
    console "\nDownloading ${localPath}\n"
    logMessage "server2local command: ${cmd}\n"
    local result=$(eval "${cmd}")
    fileExist="`cat "${localPath}.header" | grep "404 NOT FOUND" | wc -l`"
    if [ "${fileExist}" ]
    then
        echo "${result}" > ${localPath}
    fi
    rm -f ${localPath}.header
    return ${fileExist}
}


terminate_spark()
{
    if [ -n "${submissionId}" ]
    then
        logMessage "WARN: Terminate signal received. Stop spark job: ${submissionId}\n"
        local result=$(call_kill_REST)
        logMessage "Terminate result : ${result}\n"
        # Give it some time before polling for status
        sleep ${SS_POLL_INTERVAL}
        local resultStatus=$(call_status_REST)
        driverStatus="`echo ${resultStatus} | sed -n 's/.*\"driverState\" : \"\([^\"]*\)\",.*/\1/p'`"
        echo "Job kill:  ${submissionId} status is ${driverStatus}"
    fi
    endScript
}

ctrlc_handle()
{
    while true
    do
       read -p "Terminate submitted job? (y/n)" isCancel
       case $isCancel in
           [Yy]* ) isCancel=true; break;;
           [Nn]* ) isCancel=false; break;;
           * ) echo "Please answer yes or no";;
       esac
    done

    if [[ "$isCancel" = "true" ]]; then
       terminate_spark
       exit 1
    fi
    while true
    do
       read -p "Continue polling for job status? (y/n)" isPolling
       case $isPolling in
          [Yy]* ) isPolling=true; break;;
          [Nn]* ) isPolling=false; break;;
          * ) echo "Please answer yes or no";;
       esac
    done
    if [[ "$isPolling" = "false" ]]; then
       endScriptWithCommands
       exit 0
    fi
}

substituteArg()
{
    local arg=$1
    local fileName="`echo ${arg} | sed -n 's/.*file:\/\/\([^\"]*\)\"/\1/p'`"
    local newArg=${arg}
    if [ -n "${fileName}" ]
    then
        if [[ "${uploadList}" =~ "${fileName}" ]]; then
            newArg="\"file://${SERVER_SUB_DIR}/${fileName}\""
        fi
   fi
   echo "${newArg}"
}

parsing_appArgs()
{
   local argString=$1
   OIFS=${IFS}
   IFS=","
   local argArray=(${argString})
   IFS=${OIFS}
   local resultArgs=
   for ((i=0; i<${#argArray[@]}; ++i))
   do
        local arg=$(substituteArg ${argArray[${i}]})
        if [ -z "${resultArgs}" ]
        then
            resultArgs="${arg}"
        else
            resultArgs="${resultArgs},${arg}"
        fi
    done
    echo "${resultArgs}"
}

isSparkServiceConf()
{
     local conf_key="$1"
     local spark_service_confs="spark.service.tenant_id spark.service.instance_id spark.service.tenant_secret"
     [[ $spark_service_confs =~ $conf_key ]] && echo "true" || echo "false"
}


submit_REST_json()
{
    local appArgs1="$1"
    local appResource="$2"
    local mainClass="$3"
    local sparkJars="$4"
    local sparkFiles="$5"
    local sparkPYFiles="$6"
    local appArgs=$(parsing_appArgs "${appArgs1}")
    local reqJson="{"
    reqJson+=" \"action\" : \"CreateSubmissionRequest\",  "
    if [ "${PY_APP}" = "true" ]
    then
        local appResourceFileName=$(fileNameFromPath ${appResource})
        if [ -n "${sparkPYFiles}" ]
        then
            local sparkPYFileNames=$(fileNameFromPath_list ${sparkPYFiles})
            if [ -n "${appArgs}" ]
            then
                appArgs="\"--primary-py-file\",\"${appResourceFileName}\",\"--py-files\",\"${sparkPYFileNames}\",${appArgs}"
            else
                appArgs="\"--primary-py-file\",\"${appResourceFileName}\",\"--py-files\",\"${sparkPYFileNames}\""
            fi
        else
            if [ -n "${appArgs}" ]
            then
                appArgs="\"--primary-py-file\",\"${appResourceFileName}\",${appArgs}"
            else
                appArgs="\"--primary-py-file\",\"${appResourceFileName}\""
            fi
        fi
    fi
    reqJson+=" \"appArgs\" : [ ${appArgs} ], "
    reqJson+=" \"appResource\" : \"${appResource}\","
    reqJson+=" \"clientSparkVersion\" : \"${SPARK_VERSION}\","
    reqJson+=" \"mainClass\" : \"${mainClass}\", "
    reqJson+=" \"sparkProperties\" : { "

    ##### properties: spark.app.name
    reqJson+=" \"spark.app.name\" : \"${APP_NAME}\", "

    ##### properties: spark.jars - add appResource to jars list if this is java application
    if [ -n "${sparkJars}" ]
    then
        if [ "${PY_APP}" = "false" ]
        then
            sparkJars+=",${appResource}"
        fi
    else
        if [ "${PY_APP}" = "false" ]
        then
           sparkJars=${appResource}
        fi
    fi
    if [ -n "${sparkJars}" ]
    then
        reqJson+=" \"spark.jars\" : \"${sparkJars}\", "
    fi

    ##### properties: spark.files - add appResource to files list if this is python application
    if [ -n "${sparkFiles}" ]
    then
        if [ -n "${sparkPYFiles}" ]
        then
            sparkFiles+=",${appResource},${sparkPYFFiles}"
        elif [ "${PY_APP}" == "true" ]
        then
            sparkFiles+=",${appResource}"
        fi
    else
        if [ -n "${sparkPYFiles}" ]
        then
            sparkFiles="${appResource},${sparkPYFiles}"
        elif  [ "${PY_APP}" == "true" ]
        then
            sparkFiles="${appResource}"
        fi
    fi
    if [ -n "${sparkFiles}" ]
    then
        reqJson+=" \"spark.files\" : \"${sparkFiles}\", "
    fi

    ##### properties: spark.submit.pyFiles
    if [ -n "${sparkPYFiles}" ]
    then
        reqJson+=" \"spark.submit.pyFiles\" : \"${sparkPYFiles}\", "
    fi

    for ((i=0; i<${#CONF_KEY[@]}; ++i))
    do
        if [[ $(isSparkServiceConf ${CONF_KEY[${i}]}) == "false" ]]; then
           reqJson+=" \"${CONF_KEY[${i}]}\" : \"${CONF_VAL[${i}]}\", "
        fi
    done

    ##### properties: spark.service.* : all properties specific for spark service
    reqJson+=" \"spark.service.tenant_id\" : \"${TENANT_ID}\", "
    reqJson+=" \"spark.service.instance_id\" : \"${INSTANCE_ID}\", "
    reqJson+=" \"spark.service.tenant_secret\" : \"${TENANT_SECRET}\" "

    reqJson+="}"
    reqJson+="}"
    echo ${reqJson}
}

status_kill_REST_json()
{
    reqJson="{"
    reqJson+=" \"sparkProperties\" : { "
    reqJson+=" \"spark.service.tenant_id\" : \"${TENANT_ID}\", "
    reqJson+=" \"spark.service.instance_id\" : \"${INSTANCE_ID}\", "
    reqJson+=" \"spark.service.tenant_secret\" : \"${TENANT_SECRET}\", "
    reqJson+=" \"spark.service.spark_version\" : \"${SPARK_VERSION}\" "
    reqJson+="}"
    reqJson+="}"

    echo ${reqJson}
}

call_status_REST()
{
    local requestBody=$(status_kill_REST_json)
    local cmd="curl ${SS_CURL_OPTIONS} -X GET -H '$(get_requested_with_header)' -i --data-binary '${requestBody}' https://${HOSTNAME}/v1/submissions/status/${submissionId}"
    console "\nGetting status\n"
    logMessage "call_status_REST command: ${cmd}\n"
    local statusRequest=$(eval "${cmd}")
    logMessage "call_status_REST result: ${statusRequest}\n"
    echo "${statusRequest}"
}

call_kill_REST()
{
    local requestBody=$(status_kill_REST_json)
    local cmd="curl ${SS_CURL_OPTIONS} -X POST -H '$(get_requested_with_header)' -i --data-binary '${requestBody}' https://${HOSTNAME}/v1/submissions/kill/${submissionId}"
    console "\nKilling submission\n"
    logMessage "call_kill_REST command: ${cmd}\n"
    local killRequest=$(eval "${cmd}")
    logMessage "call_kill_REST result: ${killRequest}\n"
    echo "${killRequest}"
}


# =============================================================================
# -- Main ---------------------------------------------------------------------
# =============================================================================

trap ctrlc_handle SIGINT

# -- Parse command line arguments ---------------------------------------------

if [[ $# == 0 ]]
then
   printUsage
   exit 1
fi

while [[ $# > 0 ]]
do
    key="$1"
    case $key in
        --help)
            printUsage
            ;;
        --version)
            printVersion
            ;;
        --master)
            MASTER="$2"
            HOSTNAME=$(get_hostname_from_url ${MASTER})
            logMessage "MASTER HOSTNAME: ${HOSTNAME}\n"
            shift
            shift
            ;;
        --jars)
            JARS="$2"
            shift
            shift
            ;;
        --files)
            FILES="$2"
            shift
            shift
            ;;
        --class)
            CLASS="$2"
            shift
            shift
            ;;
        --conf)
            aconf="$2"
            CONF_KEY[${confI}]="`echo ${aconf} | sed -n 's/\([^=].*\)=\(.*\)/\1/p'`"
            CONF_VAL[${confI}]="`echo ${aconf} | sed -n 's/\([^=].*\)=\(.*\)/\2/p'`"
            ((confI++))
            shift
            shift
            ;;
        --vcap)
            VCAP_FILE="$2"
            shift
            shift
            ;;
        --status)
            CHECK_STATUS=true
            submissionId="$2"
            shift
            shift
            ;;
        --kill)
            KILL_JOB=true
            submissionId="$2"
            shift
            shift
            ;;
        --name)
            APP_NAME="$2"
            shift
            shift
            ;;
        --py-files)
            PY_FILES="$2"
            PY_APP=true
            shift
            shift
            ;;
        --deploy-mode)
            DEPLOY_MODE="$2"
            shift
            shift
            ;;
        *)
            if [[ "${key}" =~ ^--.* ]] &&  [[ -z "${APP_MAIN}" ]]; then
                printf "Error: Unrecognized option: ${key} \n"
                printUsage
                exit 1
            else
                if [ -z "${APP_MAIN}" ]
                then
                    APP_MAIN="${key}"
                    shift
                else
                    if [ -z "${app_parms}" ]
                    then
                        app_parms=" \"${key}\" "
                    else
                        app_parms="${app_parms}, \"${key}\" "
                    fi
                    shift
                fi
            fi
            ;;
    esac
done

# -- Initialize log file ------------------------------------------------------

if [ "${SS_LOG_ENABLE}" = "true" ]
then
    rm -f ${LOG_FILE}
    console "To see the log, in another terminal window run the following command:\n"
    console "tail -f ${LOG_FILE}\n\n"
    logMessage "Timestamp: ${EXECUTION_TIMESTAMP}\n"
    logMessage "Date: $(date +'%Y-%m-%d %H:%M:%S')\n"
    logMessage "VERSION: ${VERSION}\n"
    logMessage "\nCommand invocation: ${invokeCommand}\n"
fi

# -- Check variables ----------------------------------------------------------

# Check if both vcap file and --master option are not specified,if so raise error
if [[ -z "${VCAP_FILE}" ]] && [[ -z "${MASTER}" ]];
then
    display_master_url_err_msg
    exit 1
fi


# -- Pull values from VCAP ----------------------------------------------------

if [ ! -z "${VCAP_FILE}" ]
then
   logFile ${VCAP_FILE}

   INSTANCE_ID=$(get_from_vcap ${VCAP_FILE}  "instance_id")
   TENANT_ID=$(get_from_vcap ${VCAP_FILE}  "tenant_id")
   TENANT_SECRET=$(get_from_vcap ${VCAP_FILE} "tenant_secret")
   CLUSTER_MASTER_URL=$(get_from_vcap ${VCAP_FILE} "cluster_master_url")
fi

# -- Check variables ----------------------------------------------------------

# Check if vcap file doesnt contain master url and --master option is not specified, if so raise error.
if [[ -z "${CLUSTER_MASTER_URL}" ]] && [[ -z "${MASTER}" ]]
then
    display_master_url_err_msg
    exit 1
fi

vcap_hostname=$(get_hostname_from_url ${CLUSTER_MASTER_URL})
if [ ! -z "${MASTER}" ]
then
    if [ "${HOSTNAME}" != "${vcap_hostname}" ] # if both the --master option and vcap are specified and they are not same, use the master url from --master option.
    then
        logMessage "WARN: The URL specified in '--master ${MASTER}' option does not match with the URL in 'cluster_master_url ${CLUSTER_MASTER_URL}' in '--vcap' ${VCAP_FILE}. Using ${MASTER} url.\n"
    fi
else
    HOSTNAME="${vcap_hostname}" #If --master option is not specified, then use the master url from vcap.
fi

# If IP address (i.e. not a FQDN), then add "--insecure" curl option.
if [[ "${HOSTNAME}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    SS_CURL_OPTIONS="${SS_CURL_OPTIONS} --insecure"
fi

# -- Get values from --conf option --------------------------------------------

if [ ! -z "${aconf}" ]
then
    get_conf_options
fi

# -- Check variables ----------------------------------------------------------

if [[ -z "${TENANT_ID}" ]]; then
   display_err_msg "TENANT_ID"
   exit 1
elif [[ -z "${TENANT_SECRET}" ]]; then
   display_err_msg "TENANT_SECRET"
   exit 1
elif [[ -z "${INSTANCE_ID}" ]]; then
   display_err_msg "INSTANCE_ID"
   exit 1
fi

if [[ -z "${SPARK_VERSION}" ]]; then
   display_err_msg_spark_version
   exit 1
fi

# -- Handle request for status or cancel  -------------------------------------

if [ "${CHECK_STATUS}" = "true" ]
then
    if [ -n "${submissionId}" ]
    then
        console "$(call_status_REST)\n"
        exit 0
    else
        console "ERROR: You need to specify submission ID after --status option. Run with --help for usage information.\n"
        exit 1
    fi
fi

if [ "${KILL_JOB}" = "true" ]
then
    if [ -n "${submissionId}" ]
    then
        console "$(call_kill_REST)\n"
        exit 0
    else
        console "ERROR: You need to specify submission ID after --kill option. Run with --help for usage information.\n"
        exit 1
    fi
fi

# -- Handle request for submit  -----------------------------------------------

if [ -z "${DEPLOY_MODE}" ] || [ "${DEPLOY_MODE}" != "cluster" ]
then
    console "ERROR: '--deploy-mode' must be set to 'cluster'.\n"
    exit 1
fi

if [ -z "${APP_MAIN}" ]
then
    console "ERROR: The main application file is not specified correctly. Run with --help for usage information.\n"
    exit 1
fi

if [[ "${APP_MAIN}" =~ .*\.py ]]; then
    PY_APP=true
fi

if [ -z "${APP_NAME}" ]
then
    if [ -z "${CLASS}" ]
    then
        APP_NAME=${APP_MAIN}
    else
        APP_NAME=${CLASS}
    fi
fi

if [[ "${PY_APP}" = "false" ]] && [[ -z ${CLASS} ]]; then
   console "ERROR: Missing option --class \n"
   exit 1
fi


# -- Synthesize variables -----------------------------------------------------

if [ -z ${PREFIX_SERVER_PATH} ]; then PREFIX_SERVER_PATH="/gpfs/fs01/user/${TENANT_ID}/data"; fi

# -- Prepare remote path and upload files to the remote path ------------------

posixJars=
if [ "${JARS}" ]
then
    if [ "${SS_JARS_UPLOAD}" = "true" ]
    then
        posixJars=$(convert2serverPath_list ${JARS})
        local2server_list ${JARS} ${posixJars}
        #posixJars=$(convert2submitPath_list ${posixJars})
    else
        posixJars="${JARS}"
    fi
fi

posixFiles=
if [ "${FILES}" ]
then
    if [ "${SS_FILES_UPLOAD}" = "true" ]
    then
        posixFiles=$(convert2serverPath_list ${FILES})
        local2server_list ${FILES} ${posixFiles}
    else
        posixFiles="${FILES}"
    fi
fi

posixPYFiles=
if [ "${PY_FILES}" ]
then
    if [ "${SS_FILES_UPLOAD}" = "true" ]
    then
        posixPYFiles=$(convert2serverPath_list ${PY_FILES})
        local2server_list ${PY_FILES} ${posixPYFiles}
    else
        posixPYFiles="${PY_FILES}"
    fi
fi


if [ "${SS_APP_MAIN_UPLOAD}" = "true" ]
then
    app_server_path=$(convert2serverPath ${APP_MAIN})
    local2server ${APP_MAIN} ${app_server_path}
    #app_server_path=$(convert2submitPath ${app_server_path})
else
    app_server_path=${APP_MAIN}
fi

# -- Compose spark-submit command ---------------------------------------------

mainClass=${CLASS}
if [ "${PY_APP}" = "true" ]
then
    mainClass="org.apache.spark.deploy.PythonRunner"
fi

requestBody=$(submit_REST_json "${app_parms}" "${app_server_path}" "${mainClass}" "${posixJars}" "${posixFiles}" "${posixPYFiles}")

# -- Call spark-submit REST to submit the job to spark cluster ---------------------

cmd="curl ${SS_CURL_OPTIONS} -X POST -H '$(get_requested_with_header)' --data-binary '${requestBody}' https://${HOSTNAME}/v1/submissions/create"
console "\nSubmitting Job\n"
logMessage "Submit job command: ${cmd}\n"
resultSubmit=$(eval "${cmd}")
logMessage "Submit job result: ${resultSubmit}\n"

# -- Parse submit job output to find 'submissionId' value ---------------------

submissionId="`echo ${resultSubmit} | sed -n 's/.*\"submissionId\" : \"\([^\"]*\)\",.*/\1/p'`"
logMessage  "\nSubmission ID: ${submissionId}\n"

if [ -z "${submissionId}" ]
then
    logMessage "ERROR: Problem submitting job. Exit\n"
    endScript
    exit 1
fi

console "\nJob submitted : ${submissionId}\n"

# -- Periodically poll job status ---------------------------------------------

driverStatus="NULL"
jobFinished=false
jobFailed=false
try=1
while [[ "${jobFinished}" == false ]]
do
    console "\nPolling job status.  Poll #${try}.\n"
    resultStatus=$(call_status_REST)
    ((try++))
    driverStatus="`echo ${resultStatus} | sed -n 's/.*\"driverState\" : \"\([^\"]*\)\",.*/\1/p'`"
    console "driverStatus is ${driverStatus}\n"
    case ${driverStatus} in
        FINISHED)
            console "\nJob finished\n"
            jobFinished=true
            ;;
        RUNNING|SUBMITTED)
            console "Next poll in ${SS_POLL_INTERVAL} seconds.\n"
            sleep ${SS_POLL_INTERVAL}
            jobFinished=false
            ;;
        *)
            IS_JOB_ERROR=true
            logMessage "\n\n==== Failed Status output =====================================================\n"
            logMessage "${resultStatus}\n"
            logMessage "===============================================================================\n\n"
            jobFinished=true
            jobFailed=true
            ;;
    esac
done

# -- Download stdout and stderr files -----------------------------------------
logMessage=""
if [ -n "${submissionId}" ]
then
    LOCAL_STDOUT_FILENAME="stdout"
    LOCAL_STDERR_FILENAME="stderr"
    # MODEL_FILENAME="model.zip"
    stdout_server_path="${SS_SPARK_WORK_DIR}/${submissionId}/stdout"

    server2local ${stdout_server_path} ${LOCAL_STDOUT_FILENAME}
    if [ "$?" != 0 ]
    then
        console "Failed to download from ${stdout_server_path} to ${LOCAL_STDOUT_FILENAME}\n"
    else
        logMessage="View job's stdout log at ${LOCAL_STDOUT_FILENAME}\n"
    fi

    stderr_server_path="${SS_SPARK_WORK_DIR}/${submissionId}/stderr"
    server2local ${stderr_server_path} ${LOCAL_STDERR_FILENAME}
    if [ "$?" != 0 ]
    then
        console "Failed to download from ${stderr_server_path} to ${LOCAL_STDERR_FILENAME}\n"
    else
        logMessage="${logMessage}View job's stderr log at ${LOCAL_STDERR_FILENAME}\n"
    fi

    # model_path="${SS_SPARK_WORK_DIR}/${submissionId}/model.zip"
    # server2local ${model_path} ${MODEL_FILENAME}
    # if [ "$?" != 0 ]
    # then
    #     console "Failed to download from ${model_path} to ${MODEL_FILENAME}\n"
    # else
    #     logMessage="${logMessage}View job's stderr log at ${MODEL_FILENAME}\n"
    # fi
fi

# -- Delete transient files on spark cluster ----------------------------------

if [ "${SS_APP_MAIN_UPLOAD}" = "true" ] || [ "${SS_JARS_UPLOAD}" = "true" ] || [ "${SS_FILES_UPLOAD}" = "true" ]
then
    if [ "${jobFinished}" = "true" ]
    then
        deleteFolderOnServer ${SERVER_SUB_DIR}
    fi
fi

# -- Epilog -------------------------------------------------------------------

if [ "${IS_JOB_ERROR}" = "true" ]
then
    console "\nERROR: Job failed.\n"
    console "spark-submit log file: ${LOG_FILE}\n"
    console "${logMessage}"
    exit 1
else
    endScript
    console "${logMessage}"
fi

# -- --------------------------------------------------------------------------
