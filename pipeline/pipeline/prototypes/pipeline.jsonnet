// @apiVersion 0.1
// @name io.ksonnet.pkg.pipeline
// @description ML pipeline. Currently includes pipeline API server, frontend and dependencies.
// @shortDescription ML pipeline
// @param name string Name to give to each of the components
// @optionalParam apiImage string gcr.io/ml-pipeline/api-server:0.1.3 API docker image
// @optionalParam scheduledWorkflowImage string gcr.io/ml-pipeline/scheduledworkflow:0.1.3 schedule workflow docker image
// @optionalParam persistenceAgentImage string gcr.io/ml-pipeline/persistenceagent:0.1.3 persistence agent docker image
// @optionalParam uiImage string gcr.io/ml-pipeline/frontend:0.1.3 UI docker image
// @optionalParam deployArgo string false flag to deploy argo
// @optionalParam reportUsage string false flag to report usage
// @optionalParam mysqlImage string mysql:5.6 mysql image
// @optionalParam minioImage string minio/minio:RELEASE.2018-02-09T22-40-05Z minio image

local k = import "k.libsonnet";
local all = import "pipeline/pipeline/all.libsonnet";

std.prune(k.core.v1.list.new(all.parts(env, params).all))
