{
  parts(_env, _params):: {
    local params = _env + _params,

    local argo = import "pipeline/pipeline/argo.libsonnet",
    local minio = import "pipeline/pipeline/minio.libsonnet",
    local mysql = import "pipeline/pipeline/mysql.libsonnet",
    local pipeline_apiserver = import "pipeline/pipeline/pipeline-apiserver.libsonnet",
    local pipeline_scheduledworkflow = import "pipeline/pipeline/pipeline-scheduledworkflow.libsonnet",
    local pipeline_persistenceagent = import "pipeline/pipeline/pipeline-persistenceagent.libsonnet",
    local pipeline_ui = import "pipeline/pipeline/pipeline-ui.libsonnet",
    local spartakus = import "pipeline/pipeline/spartakus.libsonnet",

    local name = params.name,
    local namespace = params.namespace,
    local apiImage = params.apiImage,
    local scheduledWorkflowImage = params.scheduledWorkflowImage,
    local persistenceAgentImage = params.persistenceAgentImage,
    local uiImage = params.uiImage,
    local mysqlImage = params.mysqlImage,
    local minioImage = params.minioImage,
    local deployArgo = params.deployArgo,
    local reportUsage = params.reportUsage,
    local usageId = params.usageId,
    reporting:: if (reportUsage == true) || (reportUsage == "true") then
      spartakus.all(namespace, usageId)
    else [],
    argo:: if (deployArgo == true) || (deployArgo == "true") then
      argo.parts(namespace).all
    else [],
    all:: minio.all(namespace, minioImage) +
          mysql.all(namespace, mysqlImage) +
          pipeline_apiserver.all(namespace, apiImage) +
          pipeline_scheduledworkflow.all(namespace, scheduledWorkflowImage) +
          pipeline_persistenceagent.all(namespace, persistenceAgentImage) +
          pipeline_ui.all(namespace, uiImage) +
          $.parts(_env, _params).argo +
          $.parts(_env, _params).reporting,
  },
}
