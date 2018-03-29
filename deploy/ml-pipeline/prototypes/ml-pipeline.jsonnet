// @apiVersion 0.1
// @name io.ksonnet.pkg.ml-pipeline
// @description ML pipeline. Currently includes pipeline API server, frontend and dependencies.
// @shortDescription ML pipeline
// @param name string Name to give to each of the components
// @optionalParam namespace string default Namespace
// @optionalParam api_image string null API docker image
// @optionalParam ui_image string null UI docker image

local k = import "k.libsonnet";
local all = import "ml-pipeline/ml-pipeline/all.libsonnet";

std.prune(k.core.v1.list.new(all.parts(params).all))
