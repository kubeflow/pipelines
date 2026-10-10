# Kubeflow Pipelines Standalone for OpenShift

This repository contains deployment resources for Kubeflow Pipelines (KFP) Standalone on OpenShift platform:

> [!Caution]
> This is not a production ready deployment. For a production environment it is highly recommended that you thoroughly inspect the manifests and update them as needed per your use-case.

## Prerequisites
- OpenShift cluster (version 4.x+)
- [oc] CLI tool installed
- Cluster administrator access

## Deploy KFP on Openshift 

Create the `kubeflow` Openshift Project: 

```bash
oc new-project kubeflow
```

Navigate to the Openshift manifests and deploy KFP

```bash
git clone https://github.com/kubeflow/pipelines.git
cd manifests/kustomize/env/openshift/base
oc -n kubeflow apply -k .
```

Access the route via: 

```bash
echo https://$(oc get routes -n kubeflow ml-pipeline-ui --template={{.spec.host}})
```

## Driver agent UID assignment

This overlay removes fixed UIDs from Argo's standard agent Pod, its init/main
containers, the driver plugin and the controller's workload executor defaults.
OpenShift's restricted SCC assigns the namespace UID, including to the init
container that creates the shared plugin token. No `anyuid` grant is needed.

The controller is explicitly configured for the standard init/main layout
(`initlessPod.enabled: false`). Keep that setting while using this overlay:
its workflow PodSpecPatch clears the standard init/main UID fields. The
experimental init-less layout and arbitrary non-KFP container-set templates
require a separate patch design; do not enable them with this patch unchanged.
Preserve the existing TTL/retry defaults when customizing the ConfigMap, and do
not override the UID-clearing workflow patch with fixed values.

Before rollout, verify a compiled KFP agent and workload Pod with
`oc adm policy scc-review` on your cluster, then run a pipeline under its normal
runtime ServiceAccount. Check that both Pods use the intended restricted SCC and
that the driver can read its token and write System Logs. Local manifest and
PodSpecPatch tests do not replace live SCC admission validation.

## Clean up
To delete the `kubeflow` Openshift Project:

```bash
oc -n kubeflow delete -k .
```

[oc]: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/cli_tools/openshift-cli-oc
