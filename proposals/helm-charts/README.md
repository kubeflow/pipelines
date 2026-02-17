# KEP-XXXX: Helm Charts for Kubeflow Pipelines

## Summary

This proposal is to build a minimalistic, and maintainable Helm charts for Kubeflow Pipelines (KFP). These charts will serve as an alternative installation method to Kustomize, reflecting Kustomize defaults 1:1 to ensure consistency across deployment options.

## Motivation

Currently, Kubeflow Pipelines is primarily deployed using Kustomize manifests. This involves managing **278+ individual YAML files** (138 in base, 78 in env overlays, 57 in third-party). While Kustomize is powerful, a significant portion of the Kubernetes community and GitOps tooling (like ArgoCD) relies heavily on Helm.

Providing official Helm charts will:
- Lower the barrier to entry for users familiar with Helm.
- Simplify integration with GitOps workflows.
- Standardize the Helm deployment experience across the Kubeflow ecosystem, aligning with similar efforts in trainer and other components.

### Goals

1.  **Minimalistic Charts**: Create Helm charts that are easy to understand and maintain, avoiding unnecessary complexity.
2.  **1:1 Parity with Kustomize**: Ensure that the default values in the Helm charts replicate the defaults in the existing Kustomize manifests 1:1.
    -   **Definition**: Running `helm install` with default values MUST result in the same functional Kubernetes resources (Images, Args, Env Vars, ConfigMaps, Resource Limits) as `kustomize build`.
    -   **Exceptions**: Helm-specific labels (e.g., `app.kubernetes.io/managed-by: Helm`) are expected differences.
3.  **Maintainability**: Structure charts to be easily updated alongside Kustomize changes.
4.  **Alignment with Trainer**: Adopt the structure and patterns used in the [kubeflow/trainer](https://github.com/kubeflow/trainer/tree/master/charts/kubeflow-trainer) Helm charts as the reference implementation.
5.  **Testing**: Implement robust CI/CD testing infrastructure for Helm-based deployments, leveraging existing efforts in `kubeflow/manifests`.
6.  **Sensible Configuration**: Expose only sensible settings relevant to most users in values.yaml, while allowing advanced configuration via raw manifest overrides if necessary.

### Non-Goals

1.  **Replacing Kustomize**: Helm charts are an additive installation method, not a replacement for Kustomize. Kustomize will likely remain the source of truth for raw manifests.
2.  **Supporting "All-in-one" Kubeflow Platform initially**: This KEP focuses specifically on KFP charts. While these will fit into a larger Kubeflow Platform Helm chart, the scope here is KFP and its components.

## Proposal

### Chart Location and Versioning

**Suggestion**: The Helm charts should be hosted in the `kubeflow/pipelines` repository, co-located with the source code.
-   **Why**: This allows chart releases to be versioned in lock-step with backend releases (e.g., Chart v2.3.0 releases with KFP v2.3.0).
-   **Nuance**: This deviates from the centralized `kubeflow/manifests` pattern. `kubeflow/manifests` will likely need to consume these charts as upstream dependencies or Git submodules.

### Chart Structure and Complexity

KFP is a **distributed system** comprising multiple microservices (`api-server`, `frontend`, `persistence-agent`, `scheduled-workflow`, `viewer`, `cache-server`).

**Structure**: To achieve the requested **minimalism** while maintaining modularity, we will group tightly coupled components into **4 logical subcharts**, mirroring the upstream Kustomize `base` structure:

1.  `charts/kubeflow-pipelines/` (Umbrella)
    -   `charts/kfp-platform/`: **The Core**. Contains API Server, Persistence Agent, Scheduled Workflow, Frontend, Viewer, and Visualization Server.
        -   *Rationale*: These components are versioned together and are almost always deployed as a unit in `base/pipeline`.
    -   `charts/kfp-metadata/`: Metadata Writer (GRPC) + Metadata Envoy.
    -   `charts/kfp-cache/`: Cache Server + Cache Deployer.
    -   `charts/kfp-profile-controller/`: Multi-user Profile Controller (Optional, only for multi-user mode).

**Open Question (Component Grouping)**:
The initial proposal groups `metadata-writer` (a deployment in `base/pipeline`) with `kfp-metadata`. However, `metadata-writer` is a client of the metadata store and is coupled with the core pipeline logic. Maintainers should decide if it belongs in `kfp-platform` (Core) to reflect this coupling, or if `kfp-metadata` should encompass both storage and writers.

### Values and Configuration

To maintain maintainability and parity:
-   `values.yaml` will use flat, descriptive keys derived from the Kustomize `params.env` files.
-   **Open Question (Values Structure)**: Should we stick to flat keys (e.g., `dbHost`) to match Kustomize 1:1, or use standard nested Helm conventions (e.g., `db.host`)? The latter is more idiomatic for Helm users but slightly diverges from the Kustomize param names.
-   Defaults will be automatically synchronized or rigorously manually checked against Kustomize defaults.
-   We will avoid excessive templating logic that deviates from standard Kubernetes resource definitions.

### Dependency Management

KFP has strict external dependencies, specifically **Object Storage (Minio/S3)** and **Relational Database (MySQL)**.
-   The Helm chart MUST provide a mechanism to:
    1.  Provision in-cluster instances of Minio/MySQL (for easy testing/dev) - possibly via subcharts like `bitnami/mysql`.
    2.  Configure connection details for external managed services (RDS/S3) - using a clean `values.yaml` structure for secrets/hosts.
-   This adds complexity compared to `trainer` which is self-contained.
-   **Requirement**: The chart MUST support a `tags` or `enabled` mechanism (e.g., `tags.subcharts.mysql: false`) to disable in-cluster provisioning. Configuration for "External DB/S3" must be a first-class citizen, distinct from inner-provisioning logic. Helm templating will be used to conditionally render the correct config (e.g., `if .Values.externalDb.enabled`).
-   **Argo Workflows**: This will be treated as an **optional** dependency (managed via a boolean toggle like `argo.install`). To maintain 1:1 parity with Kustomize, it will be **enabled by default**, but users with existing Argo installations can easily disable it (`argo.install: false`).

### CI/CD and Testing

Testing will utilize the infrastructure set up in `kubeflow/manifests` (likely using `kind` clusters and GitHub Actions).
-   **Linting**: `helm lint` and `ct lint`.
-   **Installation**: Verify successful installation of charts on a kind cluster.
-   **Functionality**: Run a basic "smoke test" (e.g., submitting a simple pipeline via the API or specific run check) after Helm installation to verify the KFP stack is operational.


### User Stories (Optional)

#### Story 1
As a cluster administrator familiar with standard Kubernetes tooling, I want to install Kubeflow Pipelines using `helm install` so that I can easily integrate it into my existing Helm-based GitOps workflow (e.g., ArgoCD).

### Notes/Constraints/Caveats

Beyond the surface-level YAMLs, we have identified three critical implementation challenges that the Helm chart must address to be viable:

1.  **Script Injection (Profile Controller)**:
    -   The Profile Controller relies on a large Python script ([sync.py](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py)) injected via `kustomize configMapGenerator`.
    -   *Constraint*: The Helm chart cannot just reference a file.
    -   *Solution*: Use Helm's `.Files.Get` function. Place `sync.py` inside the chart directory (e.g., `files/sync.py`) and inject it into the ConfigMap template using `{{ .Files.Get "files/sync.py" | indent 4 }}`. This is a standard pattern for larger scripts and avoids the maintenance nightmare of inlining 450+ lines of code into a YAML template.

2.  **Webhook CA Management**:
    -   KFP's `MutatingWebhookConfiguration` (e.g., `pipelineversions`) has an empty `caBundle` in base. It relies on `cert-manager` injection via Kustomize patches.
    -   *Constraint*: The Helm chart MUST support `cert-manager` annotations by default but SHOULD also provide a manual method (`caBundle` value) for users without cert-manager.

3.  **Cluster-Scoped Resource Lifecycle**:
    -   KFP creates `ClusterRoles` and `MutatingWebhookConfigurations`.
    -   *Constraint*: Helm `uninstall` deletes these. If multiple KFP instances share a cluster (rare but possible for CRDs), this is destructive. We must use `helm.sh/resource-policy: keep` or separate these into a "CRD/Common" chart.

4.  **Kustomize Vars & Generators**:
    -   *Findings*: `configMapGenerator` is used for scripts (see point 1). `vars` are used for Namespace injection.
    -   *Strategy*: Replace Kustomize `$(kfp-namespace)` vars with Helm `{{ .Release.Namespace }}`. Logic in scripts (like `sync.py`) that relies on namespace perception must be verified to work with this injection.

### Risks and Mitigations

KFP has deep integrations with **Istio** and **Cert-Manager** which the Helm chart must handle:

1.  **Istio VirtualServices**:
    -   KFP relies on `VirtualService` resources to route traffic through the Kubeflow Gateway.
    -   *Complexity*: These are found in the `installs/multi-user` overlay, NOT `base`. Therefore, the Helm chart must **NOT assume** Istio is present by default. It must conditionally render these only if enabled (`global.istio.enabled` or `multiUser: true`) to match the Kustomize layering.

2.  **AuthorizationPolicies**:
    -   To support Multi-User Isolation, KFP applies strictly scoped `AuthorizationPolicy` resources.
    -   *Complexity*: Also part of the `multi-user` overlay. These policies depend on the user's identity provider (Dex, OIDC). The Helm chart needs a flexible [values.yaml] section for `oidc.issuer` and `oidc.groups`.

3.  **Cert-Manager Integration**:
    -   The `api-server` and `webhook` use certificates for mutual TLS.
    -   *Complexity*: Kustomize uses a `Certificate` CRD. Helm must detect if `cert-manager` is installed (via Capabilities) or strictly enforcing it as a dependency.

## Design Details

### Trade-off Analysis: Umbrella Chart Approach

We chose the Umbrella Chart pattern (parent `kubeflow-pipelines` chart managing sub-charts for each component) over a Monolithic Chart or Independent Charts.

**Advantages (Pros):**
1.  **Modular Maintenance**: Each component (e.g., `api-server`, `persistence-agent`) has its own chart. This allows maintainers to focus on specific context without wading through a massive 5000-line template file.
2.  **Flexible Deployment**: Users can choose to disable specific sub-charts (e.g., `enabled: false` for `frontend` if building a headless platform) easily via `values.yaml`.
3.  **Global Configuration**: We can use Helm's Global Values (`global.commonLabels`, `global.images.registry`) to set configuration once at the top level and have it propagate to all sub-charts automatically.
4.  **Isolation**: Changes to the Viewers chart won't accidentally break the API Server chart templates.

**Disadvantages (Cons):**
1.  **Value Propagation Complexity**: Passing values from parent to child can be tricky. We must be disciplined about using `global` values vs. specific overrides to avoid "value spaghetti."
2.  **Directory Structure Overhead**: It creates a deeper directory structure (`charts/parent/charts/child/templates/...`).
3.  **Versioning Overhead**: We effectively manage multiple chart versions (one for the parent, one for each child), though we can mitigate this by keeping them in sync.

### Mapping Kustomize Patches to Helm Values

To achieve 1:1 parity, we have analyzed the existing 14+ Kustomize patches and mapped them to Helm strategies. The table below covers the distinct patterns:

| Category | Kustomize Patch (Source) | Helm Strategy (Destination) | Reason |
| :--- | :--- | :--- | :--- |
| **DB Config** | [base/postgresql/pipeline/ml-pipeline-apiserver-deployment-patch.yaml](https://www.github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/postgresql/pipeline/ml-pipeline-apiserver-deployment-patch.yaml) | **Values**: `db.host`, `db.port`, `db.name` | Config varies per environment (Dev/Prod/Cloud). |
| | [base/postgresql/cache/cache-deployment-patch.yaml](https://www.github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/postgresql/cache/cache-deployment-patch.yaml) | **Values**: `db.host` | Same as above. |
| **Cloud Integration** | [env/gcp/gcp-configurations-patch.yaml](https://www.github.com/kubeflow/pipelines/blob/master/manifests/kustomize/env/gcp/gcp-configurations-patch.yaml) | **Conditional Logic**: `{{ if eq .Values.platform "gcp" }}` | Only needed on GCP (injects `PROJECT_ID`). |
| **Dev Environment** | [env/dev/api-server-patch.yaml](https://www.github.com/kubeflow/pipelines/blob/master/manifests/kustomize/env/dev/api-server-patch.yaml) | **Values**: `images.tag` (dev/master) | Dev uses different image tags/flags. |
| **Multi-User Mode** | `base/installs/multi-user/.../deployment-patch.yaml` (8 files) | **Conditional Logic**: `{{ if .Values.multiUser }}` | Feature toggle injecting headers (`KUBEFLOW_USERID`). |
| | [base/installs/multi-user/pipelines-ui/configmap-patch.yaml](https://www.github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/installs/multi-user/pipelines-ui/configmap-patch.yaml) | **Conditional Logic**: `{{ if .Values.multiUser }}` | Toggles UI capabilities for multi-user. |
| **Argo Workflow** | `third-party/argo/.../workflow-controller-deployment-patch.yaml` | **Templated Args**: `args: ["--executor-image", "{{ .Values.argo.executorImage }}"]` | Version upgrades require changing this. |

**Why not hardcode?**
We **cannot** hardcode these because they are exactly the parts of the system that change.
-   **Hardcoded in Helm**: Container ports (`8888`), Label selectors, Internal file paths.
-   **Templated**: Anything found in a Kustomize [env/](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/params.env) or `installs/` overlay.

### Frontend Considerations
This proposal involves deployment of the KFP Frontend. The Helm chart for the frontend must correctly configure the backend API proxy and other runtime environment variables that the frontend expects (e.g., `ML_PIPELINE_SERVICE_HOST`).

**Trade-off: Service Naming**:
Kustomize hardcodes service names (e.g., `ml-pipeline`). Helm defaults to dynamic naming (`{{ .Release.Name }}-...`) to allow multiple installations.
-   To maintain 1:1 parity with current Kustomize expectations (where env vars are hardcoded), we may need to default the `fullnameOverride` or `nameOverride` to the standard Kustomize names.
-   Alternatively, we must ensure all environment variables referencing these services are templated to use the dynamic Helm names.

### Test Plan

-   **Automated Testing**:
    -   Unit tests for Helm templates (if complex logic exists).
    -   Integration tests in CI: Install chart -> Wait for pods -> Run checks.
    -   **Tests (CI Diffing)**: This is to prevent drift. A CI job must:
        1.  Run `kustomize build ...` to generate the expected reference.
        2.  Run `helm template ...` with equivalent values.
        3.  Strip Helm-specific labels and comments.
        4.  Compare the output. Any diff failure blocks the PR. This replaces manual verification as the primary parity check.
-   **Manual Verification**:
    -   Verify parity by rendering both Helm and Kustomize manifests and diffing the output (accounting for labeling differences).

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

## Implementation History

-   **Experimental**: Initial experimental charts were developed by Kunal Dugar in [kubeflow/manifests#3237](https://github.com/kubeflow/manifests/pull/3237).

## Drawbacks

-   **Dual Maintenance**: Maintaining both Kustomize and Helm requires discipline to ensure they don't diverge. Automation or strict policy is required.

## Alternatives

-   **Kustomize-only**: Continue as-is. Rejected due to strong community demand for Helm.
-   **Helmify**: Auto-generate Helm charts from Kustomize. This is a valid approach but often produces unidiomatic charts. A "minimalistic, hand-crafted but synced" approach is preferred for user experience.