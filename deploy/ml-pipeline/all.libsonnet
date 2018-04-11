{
  parts(params):: {
    local argo = import "ml-pipeline/ml-pipeline/argo.libsonnet",
    local minio = import "ml-pipeline/ml-pipeline/minio.libsonnet",
    local mysql = import "ml-pipeline/ml-pipeline/mysql.libsonnet",
    local pipeline_apiserver = import "ml-pipeline/ml-pipeline/pipeline-apiserver.libsonnet",
    local pipeline_ui = import "ml-pipeline/ml-pipeline/pipeline-ui.libsonnet",
    local spartakus = import "ml-pipeline/ml-pipeline/spartakus.libsonnet",

    local name = params.name,
    local namespace = params.namespace,
    local api_image = params.api_image,
    local ui_image = params.ui_image,
    local report_usage = params.report_usage,
    local usage_id = params.usage_id,
    reporting:: if report_usage then
                  spartakus.all(namespace,usage_id)
                else [],
    all:: argo.parts(namespace).all +
          minio.parts(namespace).all +
          mysql.parts(namespace).all +
          pipeline_apiserver.all(namespace,api_image) +
          pipeline_ui.all(namespace,ui_image) +
          $.parts(params).reporting,
  },
}
