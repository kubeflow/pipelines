// @apiVersion 0.1
// @name io.ksonnet.pkg.pytorch-operator
// @description PyTorch Operator
// @shortDescription PyTorch Operator for distributed pytorch on Kubernetes
// @param name string Name to give to each of the components
// @optionalParam namespace string null Namespace to use for the components. It is automatically inherited from the environment if not set.
// @optionalParam disks string null Comma separated list of Google persistent disks to attach to jupyter environments.
// @optionalParam cloud string null String identifying the cloud to customize the deployment for.
// @optionalParam pytorchJobImage string gcr.io/kubeflow-images-public/pytorch-operator:v0.3.0 The image for the PyTorchJob controller
// @optionalParam pytorchDefaultImage string null The default image to use for pytorch
// @optionalParam pytorchJobVersion string v1alpha2 which version of the PyTorchJob operator to use
// @optionalParam deploymentScope string cluster The scope at which pytorch-operator should be deployed - valid values are cluster, namespace.
// @optionalParam deploymentNamespace string null The namespace to which pytorch-operator should be scoped. If deploymentScope is set to cluster, this is ignored.

local k = import "k.libsonnet";
local operator = import "kubeflow/pytorch-job/pytorch-operator.libsonnet";

std.prune(k.core.v1.list.new(operator.all(params, env)))
