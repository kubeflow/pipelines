# ML Pipeline Test Infrastructure

This folder contains the integration and end-to-end (E2E) tests for the ML pipeline. Tests are executed using Kind (Kubernetes IN Docker) to simulate a Kubernetes cluster locally. GitHub Actions (GHAs) handle automated testing on pull requests.

At a high level, a typical test workflow will:
- Build images for all components.
- Create a dedicated test namespace in the cluster.
- Deploy the ML pipeline using the newly built components.
- Run the tests.
- Clean up the namespace and temporary resources.

These steps are performed in the same Kubernetes cluster.

---

## Running Tests Locally with Kind

To run tests locally, set up a Kind cluster and follow the same steps as the GitHub Actions workflows. This section details the process.

### Setup

1. **Install Prerequisites**:
   - A container engine like [Podman](https://podman.io) or [Docker](https://docs.docker.com/get-docker/)
   - [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
   - [kubectl](https://kubernetes.io/docs/tasks/tools/)

2. **Set Up a Kind Cluster**:
   Create a configuration file for your Kind cluster (optional):
   ```yaml
   kind: Cluster
   apiVersion: kind.x-k8s.io/v1alpha4
   nodes:
     - role: control-plane
     - role: worker
    ```
   Create the cluster:

    `kind create cluster --name kfp-test-cluster --config kind-config.yaml`

    Verify the cluster:

    `kubectl cluster-info --context kind-kfp-test-cluster`

3.  **Prepare the Test Environment**:

    -   Install Python test dependencies:
        
        `pip install -r test/requirements.txt`

    -   Deploy Kubeflow Pipelines to the Kind cluster:
        
        `kubectl apply -k manifests/`

4.  **Run the Tests**: 
Execute the desired test suite:

    `pytest test/kfp-functional-test/`

For additional guidance on deploying Kubeflow Pipelines in Kind, refer to:

-   [Kind Local Cluster Deployment Guide](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/installation/localcluster-deployment/#kind)
-   [Operator Deployment Guide](https://www.kubeflow.org/docs/components/pipelines/operator-guides/installation/#deploying-kubeflow-pipelines)


## Automated Testing with GitHub Actions


Tests are automatically triggered on GitHub when:

-   A pull request is opened or updated.

GitHub Actions workflows are defined in the `.github/workflows/` directory.

### Reproducing CI Steps Locally

To replicate the steps locally:

1.  Clone the Kubeflow Pipelines repository:

    `git clone https://github.com/kubeflow/pipelines.git
    cd pipelines`

2.  Follow the steps outlined in the **Running Tests Locally with Kind** section.

3.  To mimic the GitHub Actions environment, export any required environment variables found in the workflow files.

* * * * *

Troubleshooting
---------------

**Q: Why is my test taking so long?**

-   The first run downloads many container images. Subsequent runs will be faster due to caching.
-   If you experience high latency, ensure the local system running Kind has sufficient resources (CPU, memory).

**Q: How do I clean up the Kind cluster?**

-   Delete the Kind cluster:
    `kind delete cluster --name kfp-test-cluster`