# Kubeflow Pipelines Helm Chart

## Known Differences vs Kustomize (platform-agnostic)

This chart aims for functional parity with the Kustomize deployment in
`manifests/kustomize/env/platform-agnostic`. Some differences are intentional
due to Helm subcharts and install semantics:

- Argo Workflows is installed via the `argo-workflows` subchart. Resource names
  and some RBAC object names differ from the Kustomize namespace-install
  manifests, even when `singleNamespace` is enabled.
- Argo Workflows CRDs are disabled in values, but Helm can still install CRDs
  from other sources; Kustomize applies CRDs separately outside the
  platform-agnostic overlay.
- MySQL is deployed via the Bitnami `mysql` subchart, which uses a StatefulSet,
  headless service, and additional resources (PDB, NetworkPolicy, Secret) that
  do not exist in the Kustomize deployment (which uses a simple Deployment + PVC).
- KFP CRDs in `charts/kubeflow-pipelines/crds` are installed by Helm, while the
  Kustomize platform-agnostic overlay does not include them by default.

If you need exact 1:1 resource parity with Kustomize, replace the Argo and MySQL
subcharts with Kustomize-equivalent native templates.

## Compatibility Options

To reduce naming drift without removing subcharts, this chart provides optional
compatibility resources (enabled by default):

- `compat.argo.rbac`: Adds `argo-role` and `argo-binding` to match the
  namespace-install RBAC objects from Kustomize.
- `compat.argo.priorityClass`: Adds `PriorityClass` `workflow-controller`.
- `compat.mysql.pvc`: Creates `PersistentVolumeClaim` `mysql-pv-claim` and sets
  MySQL to use it via `mysql.primary.persistence.existingClaim`.
- `compat.mysql.deployment`: Creates a Kustomize-style MySQL `Deployment` and
  `ServiceAccount`/`Service`. This is only rendered when `mysql.enabled=false`
  to avoid running two MySQL workloads.

These are intended to close parity gaps in tooling and tests that expect
Kustomize resource names.
