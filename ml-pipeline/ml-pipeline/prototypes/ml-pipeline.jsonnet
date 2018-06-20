// @apiVersion 0.1
// @name io.ksonnet.pkg.ml-pipeline
// @description ML pipeline. Currently includes pipeline API server, frontend and dependencies.
// @shortDescription ML pipeline
// @param name string Name to give to each of the components
// @optionalParam namespace string default Namespace
// @optionalParam api_image string null API docker image
// @optionalParam scheduler_image string null Scheduler docker image
// @optionalParam scheduledworkflow_image string null schedule workflow docker image
// @optionalParam persistenceagent_image string null persistence agent docker image
// @optionalParam ui_image string null UI docker image
// @optionalParam deploy_argo string null flag to deploy argo
// @optionalParam report_usage string null flag to report usage

local k = import "k.libsonnet";
local all = import "ml-pipeline/ml-pipeline/all.libsonnet";

std.prune(k.core.v1.list.new(all.parts(params).all))
