{
  parts(_env, _params):: {
    local params = _env + _params,

    local argo = import "ml-pipeline/ml-pipeline/argo.libsonnet",
    local minio = import "ml-pipeline/ml-pipeline/minio.libsonnet",
    local mysql = import "ml-pipeline/ml-pipeline/mysql.libsonnet",
    local pipeline_apiserver = import "ml-pipeline/ml-pipeline/pipeline-apiserver.libsonnet",
    local pipeline_scheduledworkflow = import "ml-pipeline/ml-pipeline/pipeline-scheduledworkflow.libsonnet",
    local pipeline_persistenceagent = import "ml-pipeline/ml-pipeline/pipeline-persistenceagent.libsonnet",
    local pipeline_ui = import "ml-pipeline/ml-pipeline/pipeline-ui.libsonnet",
    local spartakus = import "ml-pipeline/ml-pipeline/spartakus.libsonnet",

    local name = params.name,
    local namespace = params.namespace,
    local apiImage = params.apiImage,
    local scheduledWorkflowImage = params.scheduledWorkflowImage,
    local persistenceAgentImage = params.persistenceAgentImage,
    local uiImage = params.uiImage,
    local deployArgo = params.deployArgo,
    local reportUsage = params.reportUsage,
    local usage_id = params.usage_id,
    reporting:: if (reportUsage == true) || (reportUsage == "true") then
                  spartakus.all(namespace,usage_id)
                else [],
    argo:: if (deployArgo == true) || (deployArgo == "true") then
             argo.parts(namespace).all
           else [],
    all:: minio.parts(namespace).all +
          mysql.parts(namespace).all +
          pipeline_apiserver.all(namespace,apiImage) +
          pipeline_scheduledworkflow.all(namespace,scheduledWorkflowImage) +
          pipeline_persistenceagent.all(namespace,persistenceAgentImage) +
          pipeline_ui.all(namespace,uiImage) +
          $.parts(_env, _params).argo +
          $.parts(_env, _params).reporting,
  },
}
