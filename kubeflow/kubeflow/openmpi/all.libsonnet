local assets = import "kubeflow/openmpi/assets.libsonnet";
local service = import "kubeflow/openmpi/service.libsonnet";
local serviceaccount = import "kubeflow/openmpi/serviceaccount.libsonnet";
local workloads = import "kubeflow/openmpi/workloads.libsonnet";

{
  all(params, env):: $.parts(params, env).all,

  parts(params, env):: {
    // updatedParams uses the environment namespace if
    // the namespace parameter is not explicitly set
    local updatedParams = params {
      namespace: if params.namespace == "null" then env.namespace else params.namespace,
      init: if params.init == "null" then "/kubeflow/openmpi/assets/init.sh" else params.init,
      exec: if params.exec == "null" then "sleep infinity" else params.exec,
    },

    all::
      assets.all(updatedParams)
      + service.all(updatedParams)
      + serviceaccount.all(updatedParams)
      + workloads.all(updatedParams),
  },
}
