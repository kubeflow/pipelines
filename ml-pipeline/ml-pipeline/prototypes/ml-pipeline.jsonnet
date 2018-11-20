// @apiVersion 0.1
// @name io.ksonnet.pkg.ml-pipeline
// @description ML pipeline. Currently includes pipeline API server, frontend and dependencies.
// @shortDescription ML pipeline
// @param name string Name to give to each of the components
// @optionalParam api_image string gcr.io/ml-pipeline/api-server:0.1.2 API docker image
// @optionalParam scheduledworkflow_image string gcr.io/ml-pipeline/scheduledworkflow:0.1.2 schedule workflow docker image
// @optionalParam persistenceagent_image string gcr.io/ml-pipeline/persistenceagent:0.1.2 persistence agent docker image
// @optionalParam ui_image string gcr.io/ml-pipeline/frontend:0.1.2 UI docker image
// @optionalParam deploy_argo string false flag to deploy argo
// @optionalParam report_usage string false flag to report usage

local k = import "k.libsonnet";
local all = import "ml-pipeline/ml-pipeline/all.libsonnet";

std.prune(k.core.v1.list.new(all.parts(env, params).all))
