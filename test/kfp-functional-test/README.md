# kfp-functional-test

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies:
1. edit [requirements.in](requirements.in)
1. run

    ```bash
    pip-compile requirements.in
    ```
    to update and pin the transitive dependencies.

## Run kfp-functional-test in local

### Via python (Using Kind)

1.  Set up a Kind cluster:
    
    ```bash 
    kind create cluster --name kfp-functional-test-cluster
    ```

2.  Deploy Kubeflow Pipelines to the Kind cluster:

    ```bash
    kubectl apply -k manifests/
    ```

3.  Ensure the cluster is ready:

   ```bash
    kubectl cluster-info --context kind-kfp-functional-test-cluster
   ```

4.  Run the functional test:

    ```bash
    cd {YOUR_ROOT_DIRECTORY_OF_KUBEFLOW_PIPELINES}
    python3 ./test/kfp-functional-test/run_kfp_functional_test.py --host "http://localhost:8080"
    ```

### Via Kind

1.  Set up a Kind cluster:

    ```bash
    kind create cluster --name kfp-functional-test-cluster
    ```

2.  Deploy Kubeflow Pipelines to the Kind cluster:
   
    ```bash
    kubectl apply -k manifests/
    ```

3.  Ensure the cluster is ready:

    ```bash
    kubectl cluster-info --context kind-kfp-functional-test-cluster
    ```

4.  Start a container and run the functional test:
   
    Using Docker:
    ```bash
    docker run -it -v $(pwd):/tmp/src -w /tmp/src python:3.9-slim\
    /tmp/src/test/kfp-functional-test/kfp-functional-test.sh --host "http://localhost:8080"
    ```

    Using Podman:
    ```bash
    podman run -it -v $(pwd):/tmp/src:Z -w /tmp/src python:3.9-slim \
    /tmp/src/test/kfp-functional-test/kfp-functional-test.sh --host "http://localhost:8080"
    ```


## Periodic Functional Tests with GitHub Actions

A periodic GitHub Actions workflow is configured to automatically run functional tests daily. The workflow ensures consistent validation of the Kubeflow Pipelines functionality.

For more details, see the [Periodic Functional Tests GitHub Actions workflow](https://github.com/kubeflow/pipelines/blob/master/.github/workflows/periodic.yml)

