# Kubeflow Pipelines Kustomize Manifest Folder Structure

Folder structure:
```
kustomize
├── cluster-scoped-resources
│   ├── README.md (explains this folder structure)
│   └── kustomization.yaml (lists all cluster folders in base/)
├── base
│   ├── cache-deployer
│   │   └── cluster-scoped (not included in cache-deployer's kustomization.yaml)
│   │       ├── clusteroles
│   │       └── clusterrolebindings
│   ├── argo
│   │   └── cluster-scoped
│   │       └── workflow-crd.yaml
│   ├── application
│   │   └── cluster-scoped
│   │       ├── application-crd.yaml
│   │       └── ...
│   ├── pipeline
│   │   └── cluster-scoped
│   │       ├── viewer-crd.yaml
│   │       ├── scheduledworkflow-crd.yaml
│   │       └── ...
│   └── ...
├── namespaced (centralized configs to make namespace configurable)
└── env
    ├── gcp
    │   ├── cluster-scoped-resources (set ../../cluster-scoped-resources as base)
    │   └── kustomization.yaml (set ../../namespaced as base)
    ├── aws
    └── dev
```

* User facing manifest entrypoint is `cluster-scoped-resources` package and `env/<env-name>` package.
    * `cluster-scoped-resources` should collect all cluster-scoped resources.
    * `env/<env-name>` should collect env specific namespace scoped resources.
* Universal components live in `base/<component-name>` folders.
    * If a component requires cluster-scoped resources, it should have a folder inside named `cluster-scoped` with related resources, but note that `base/<component-name>/kustomization.yaml` shouldn't include the `cluster-scoped` folder. `cluster-scoped` folders should be collected by top level `cluster-scoped-resources` folder.
* Env specific overlays live in `env/<env-name>` folders.

When users need to deploy/upgrade Kubeflow Pipelines standalone, they should
run the following commands:
```
export VERSION=<kfp-version>
kubectl apply -k github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$VERSION
kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
kubectl apply -k github.com/kubeflow/pipelines/manifests/kustomize/env/dev?ref=$VERSION # or other env folders
```

Constraints we need to comply with (that drove above structure):
* CRDs must be applied separately, because if we apply CRs in the same `kubectl apply` command, the CRD may not have been accepted by k8s api server (e.g. Application CRD).
* [A Kubeflow 1.0 constraint](https://github.com/kubeflow/pipelines/issues/2884#issuecomment-577158715) is that we should separate cluster scoped resources from namespace scoped resources, because sometimes different roles are required to deploy them. Cluster scoped resources usually need a cluster admin role, while namespaced resources can be deployed by individual teams managing a namespace. Reference
* #3376 introduced clusterrole and clusterrolebinding to kustomize manifests, so we now need a structure that can hold not just CRDs as cluster-scoped resources
