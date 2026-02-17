# Manifests for Argo workflows

Kubeflow Pipelines uses [Argo Workflows](https://argoproj.github.io/argo-workflows/) as the underlying workflow execution engine.

This folder contains preconfigured Argo Workflows installations used in Kubeflow Pipelines distributions that use **remote references** to the upstream Argo Workflows repository instead of local copies.

## Argo Workflows Dependency Documentation

Refer to [Argo Workflows README.md](../../../../third_party/argo/README.md).

## Upgrading Argo Workflows

Refer to [Argo Workflows Upgrade documentation](../../../../third_party/argo/UPGRADE.md).

## Remote References Implementation

KFP uses remote Git references to Argo Workflows manifests instead of maintaining local copies. This approach:

* Eliminates licensing concerns about copying manifests
* Simplifies deployment and upgrade processes
* Ensures direct tracking of upstream changes

All kustomization files reference manifests directly from the [Argo Workflows repository](https://github.com/argoproj/argo-workflows) using versioned Git references.

