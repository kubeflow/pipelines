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

## Clean up
To delete the `kubeflow` Openshift Project:

```bash
oc -n kubeflow delete -k .
```

[oc]: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/cli_tools/openshift-cli-oc
