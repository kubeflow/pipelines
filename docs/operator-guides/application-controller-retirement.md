# Retiring the legacy Application controller

KFP's `master` manifests no longer deploy
`gcr.io/ml-pipeline/application-crd-controller:20231101` or the KFP
`app.k8s.io/v1beta1` Application, and no longer install the shared
`applications.app.k8s.io` CRD from either cluster-scoped installation entrypoint.
The controller retirement affects `env/gcp`, `env/dev`,
`env/dev/postgresql`, `env/cert-manager/dev`, and overlays inheriting them,
including `sample`. This change is not backported to 2.18.1.

The integration supplied application metadata, resource grouping, status, and
ownership for the old Google Cloud Console experience. The Application linked
to legacy AI Platform Pipelines, which Google lists as
[shut down on January 31, 2025](https://cloud.google.com/vertex-ai/docs/deprecations).
GCP deployment support, Cloud SQL configuration, and the KFP UI/API remain.
Application-level status and deletion-based uninstall are retired; manage KFP
resources through your deployment tooling instead.

Custom overlays must remove imports of `base/application` and
`third-party/application`, and replacements targeting `kind: Application`.
Also remove imports of `third-party/application/cluster-scoped`; its definition
is removed from master. Keep any existing workload selector labels. Other
applications that need the CRD must manage it independently of KFP.

Fresh installations need no migration. Existing installations require the
procedure below: applying manifests does not delete resources omitted from the
new manifests, so an ordinary apply would leave the old controller running.

## Upgrade existing installations

Perform this migration before applying or pruning the new manifests. First
pause GitOps reconciliation, automated pruning, and any installer that could
recreate the old controller or Application. Do not run `kubectl delete -k`
against the old profile. Do not delete `applications.app.k8s.io`: the cluster-wide
CRD must be preserved in the live cluster because other applications can use it.
Complete the prune-protection steps below **before changing the tracked revision**,
even if your KFP profile never deployed the Application controller. Standard
cluster-scoped installs previously installed this CRD too.

### Protect the existing shared CRD first

Removing a CRD from Git does not delete it with ordinary `kubectl apply`, but
GitOps pruning or `kubectl apply --prune` can. Deleting this CRD deletes its
Application objects across namespaces; recreating the definition does not
restore them. Do not rely on `--cascade=orphan` on the CRD to preserve its custom
resources.

While reconciliation is paused:

1. Inventory `kubectl get applications.app.k8s.io --all-namespaces` and record
   the CRD UID with `kubectl get crd applications.app.k8s.io -o jsonpath='{.metadata.uid}'`.
   If the CRD is absent, no CRD migration is needed.
2. Protect the **live CRD** in the tool that owns it. Persist the protection in
   your existing deployment configuration before removing its manifest, or
   arrange a deliberate handoff to a separately managed platform resource.
   Verify the protection reached the live object; do not rely solely on a local
   edit or a patch that reconciliation could overwrite.

   - **Argo CD:** merge `Prune=false,Delete=false` into the CRD's
     `argocd.argoproj.io/sync-options` annotation, preserving any existing options.
     `Prune=false` prevents sync pruning; `Delete=false` also protects against
     deleting the owning Argo CD Application. These refer to the GitOps
     `argoproj.io` Application, not the retired `app.k8s.io` Application.
     See [Argo CD sync options](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-options/).
   - **Flux Kustomization:** set the CRD annotation
     `kustomize.toolkit.fluxcd.io/prune: disabled`. See
     [Flux resource pruning](https://fluxcd.io/flux/components/kustomize/kustomizations/#prune).
   - **Other installers:** exclude this CRD from deletion/prune inventories using
     that tool's supported mechanism. For plain kubectl, apply without `--prune`
     and do not delete the old cluster-scoped manifest set.

3. Review the planned sync/deletion set. The CRD must not be scheduled for
   deletion. Complete the KFP Application orphaning procedure below before
   allowing removal of that Application. If no KFP Application/controller exists,
   skip that procedure but still protect the shared CRD.
4. After the upgrade, verify the CRD still has the recorded UID and all unrelated
   Applications retain their names and UIDs. Keep prune protection until the
   platform owner has completed a verified management handoff. Do not clear it
   merely to remove an out-of-sync indicator.

These annotations are interpreted by their respective GitOps controllers; they
do not prevent direct Kubernetes deletion. Removal of an unused shared CRD is a
separate cluster-administrator decision, after checking every namespace and all
consumers. It is not part of the KFP upgrade.

### Retire the KFP Application and controller

The old Application uses `addOwnerRef: true`. Deleting it with normal cascading
deletion can delete KFP Deployments, Services, Secrets, ConfigMaps, ServiceAccounts,
Roles, and RoleBindings. Stopping the controller alone does not remove these
ownership references. Use Kubernetes' documented
[orphan deletion](https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/#delete-owner-objects-and-orphan-dependents).

1. Select the intended cluster context and KFP namespace. Inspect the actual
   Application name, selector, and UID; do not assume the default name. Confirm
   that it describes this KFP installation and inventory its dependents. Record
   their names and UIDs and back up the installation using your normal protected
   backup process (resource exports can contain Secrets).

   ```sh
   kubectl config current-context
   KFP_NAMESPACE=kubeflow
   kubectl -n "$KFP_NAMESPACE" get applications.app.k8s.io
   KFP_APPLICATION=your-verified-application-name
   kubectl -n "$KFP_NAMESPACE" get applications.app.k8s.io "$KFP_APPLICATION" -o yaml
   KFP_APPLICATION_UID=$(kubectl -n "$KFP_NAMESPACE" get applications.app.k8s.io "$KFP_APPLICATION" -o jsonpath='{.metadata.uid}')
   ```

2. Inspect the `controller-manager` Deployment before changing it. That name is
   generic: confirm its container runs `/kube-app-manager`, uses the legacy image
   above and the `application` ServiceAccount. Check for other Applications it
   manages in the namespace. If other installations depend on it, stop here and
   coordinate their migration; do not remove a shared controller blindly.

   ```sh
   kubectl -n "$KFP_NAMESPACE" get deployment controller-manager -o yaml
   kubectl -n "$KFP_NAMESPACE" scale deployment controller-manager --replicas=0
   kubectl -n "$KFP_NAMESPACE" wait --for=delete pod \
     -l control-plane=controller-manager --timeout=120s
   ```

   Confirm no controller Pods remain. Keep reconciliation paused throughout the
   following steps. If Pods do not terminate, resolve that before proceeding.

3. Delete only the verified KFP Application, explicitly orphaning dependents:

   ```sh
   kubectl -n "$KFP_NAMESPACE" delete applications.app.k8s.io "$KFP_APPLICATION" \
     --cascade=orphan --wait=true --timeout=120s
   ```

   If deletion stalls, investigate finalizers; do not force-delete the object or
   strip all owner references. Confirm the Application is gone, every inventoried
   dependent still exists with the same UID, and none retains an owner reference
   to `$KFP_APPLICATION_UID`. Include custom resource kinds if your Application
   used a customized `componentKinds` list. For the stock kinds, inspect ownership
   without printing Secret data:

   ```sh
   kubectl -n "$KFP_NAMESPACE" get deployments,services,configmaps,secrets,serviceaccounts,roles,rolebindings \
     -o custom-columns='KIND:.kind,NAME:.metadata.name,UID:.metadata.uid,OWNER_UIDS:.metadata.ownerReferences[*].uid'
   ```

4. After verifying preservation, remove the verified legacy controller resources:

   ```sh
   kubectl -n "$KFP_NAMESPACE" delete deployment controller-manager
   kubectl -n "$KFP_NAMESPACE" delete service controller-manager-service
   kubectl -n "$KFP_NAMESPACE" delete rolebinding application-manager-rolebinding
   kubectl -n "$KFP_NAMESPACE" delete role application-manager-role
   kubectl -n "$KFP_NAMESPACE" delete serviceaccount application
   ```

   Adapt names only for verified customizations. Keep the Application CRD.
   Apply the updated profile with your normal configuration, then resume GitOps
   using the updated revision. Verify KFP readiness, UI/API access, and a pipeline
   run, including your configured database and artifact-store access.

Existing `application-crd-id` labels intentionally remain: they participate in
immutable workload selectors. Do not remove them during this migration.

## Recovery

Before orphaning, recovery is to restore the controller replica count and the
previous deployment configuration. After orphaning, the workloads remain but
Application ownership is gone. Rolling back to an old profile can recreate the
Application/controller and reattach ownership; inspect that behavior before
allowing automated pruning. Restore workloads from backups only if they were
actually deleted; do not replace healthy resources as part of retirement.
