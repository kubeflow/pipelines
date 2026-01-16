# Kubeflow Pipelines Backend

## Overview

This directory contains code for the components that comprise the Kubeflow
Pipelines backend.

This README will help you set up your coding environment in order to build and run the Kubeflow Pipelines backend. The KFP backend powers the core functionality of the KFP platform, handling API requests, workflow management, and data persistence.

## Prerequisites
Before you begin, ensure you have:
- [Go installed](https://go.dev/doc/install)
- Docker or Podman installed (for building container images)

Note that you may need to restart your shell after installing these resources in order for the changes to take effect.

## Testing

To run all unittests for backend:

```
go test -v -cover ./backend/...
```

If running a [local API server](#run-the-kfp-backend-locally-with-a-kind-cluster), you can run the integration tests
with:

```bash
LOCAL_API_SERVER=true go test -v ./backend/test/v2/integration/... -namespace kubeflow -args -runIntegrationTests=true
```

To run a specific test, you can use the `-run` flag. For example, to run the `TestCacheSingleRun` test in the
`TestCache` suite, you can use the `-run 'TestCache/TestCacheSingleRun'` flag to the above command.

## Build

The API server itself can be built using:

```
go build -o /tmp/apiserver backend/src/apiserver/*.go
```

The API server image can be built from the root folder of the repo using:
```
export API_SERVER_IMAGE=api_server
docker build -f backend/Dockerfile . --tag $API_SERVER_IMAGE
```
### Deploying the APIServer (from the image you built) on Kubernetes

First, push your image to a registry that is accessible from your Kubernetes cluster.

Then, run:
```
kubectl edit deployment.v1.apps/ml-pipeline -n kubeflow
```
You'll see the field reference the api server container image (`spec.containers[0].image: gcr.io/ml-pipeline/api-server:<image-version>`).
Change it to point to your own build, after saving and closing the file, apiserver will restart with your change.

### Building client library and swagger files

After making changes to proto files, the Go client libraries, Python client libraries and swagger files
need to be regenerated and checked-in. Refer to [backend/api](./api/README.md) for details.

### Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `./update_requirements.sh` to update and pin the transitive
dependencies.


### Building conformance tests (WIP)

Run
```
docker build . -f backend/Dockerfile.conformance -t <tag>
```

## API Server Development

### Run the KFP Backend Locally With a Kind Cluster

This deploys a local Kubernetes cluster leveraging [kind](https://kind.sigs.k8s.io/), with all the components required
to run the Kubeflow Pipelines API server. Note that the `ml-pipeline` `Deployment` (API server) has its replicas set to
0 so that the API server can be run locally for debugging and faster development. The local API server is available by
pods on the cluster using the `ml-pipeline` `Service`.

#### Prerequisites

* The [kind CLI](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) is installed.
* The following ports are available on your localhost: 3000, 3306, 8080, 9000, and 8889. If these are unavailable,
  modify [kind-config.yaml](../tools/kind/kind-config.yaml) and configure the API server with alternative ports when
  running locally.
* If using a Mac, you will need to modify the
  [Endpoints](../manifests/kustomize/env/dev-kind/forward-local-api-endpoint.yaml) manifest to leverage the bridge
  network interface through Docker/Podman Desktop. See
  [kind #1200](https://github.com/kubernetes-sigs/kind/issues/1200#issuecomment-1304855791) for an example manifest.
* Optional: VSCode is installed to leverage a sample `launch.json` file.
  * This relies on dlv: (go install -v github.com/go-delve/delve/cmd/dlv@latest)

#### Provisioning the Cluster

To provision the kind cluster, run the following from the Git repository's root directory:

```bash
make -C backend dev-kind-cluster
```

This may take several minutes since there are many pods. Note that many pods will be in "CrashLoopBackOff" status until
all the pods have started.

> [!NOTE]
> The config in the `make` command above sets the `ml-pipeline` `Deployment` (api server) to have 0 replicas. The intent is to replace it with a locally running API server for debugging and faster development. See the following steps to run the API server locally, and connect it to the KFP backend on your Kind cluster. Note that other backend components (for example, the persistence agent) may show errors until the API server is brought up and connected to the cluster.

#### Launching the API Server With VSCode

After the cluster is provisioned, you may leverage the following sample `.vscode/launch.json` file to run the API
server locally:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API Server (Kind)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/src/apiserver",
      "env": {
        "POD_NAMESPACE": "kubeflow",
        "DBCONFIG_MYSQLCONFIG_HOST": "localhost",
        "OBJECTSTORECONFIG_HOST": "localhost",
        "OBJECTSTORECONFIG_PORT": "9000",
        "METADATA_GRPC_SERVICE_SERVICE_HOST": "localhost",
        "METADATA_GRPC_SERVICE_SERVICE_PORT": "8080",
        "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST": "localhost",
        "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT": "8888",
        "V2_LAUNCHER_IMAGE": "ghcr.io/kubeflow/kfp-launcher:master",
        "V2_DRIVER_IMAGE": "ghcr.io/kubeflow/kfp-driver:master"
      },
      "args": [
        "--config",
        "${workspaceFolder}/backend/src/apiserver/config",
        "-logtostderr=true"
      ]
    }
  ]
}
```

#### Using the Environment

Once the cluster is provisioned and the API server is running, you can access the API server at
[http://localhost:8888](http://localhost:8888)
(e.g. [http://localhost:8888/apis/v2beta1/pipelines](http://localhost:8888/apis/v2beta1/pipelines)).

You can also access the Kubeflow Pipelines web interface at [http://localhost:3000](http://localhost:3000).

You can also directly connect to the MariaDB database server with:

```bash
mysql -h 127.0.0.1 -u root
```

### Scheduled Workflow Development

If you also want to run the Scheduled Workflow controller locally, stop the controller on the cluster with:

```bash
kubectl -n kubeflow scale deployment ml-pipeline-scheduledworkflow --replicas=0
```

Then you may leverage the following sample `.vscode/launch.json` file to run the Scheduled Workflow controller locally:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Scheduled Workflow controller (Kind)",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/backend/src/crd/controller/scheduledworkflow",
            "env": {
                "CRON_SCHEDULE_TIMEZONE": "UTC"
            },
            "args": [
                "-namespace=kubeflow",
                "-kubeconfig=${workspaceFolder}/kubeconfig_dev-pipelines-api",
                "-mlPipelineAPIServerName=localhost"
            ]
        }
    ]
}
```

### Remote Debug the Driver

These instructions assume you are leveraging the Kind cluster in the
[Run Locally With a Kind Cluster](#run-locally-with-a-kind-cluster) section.

#### Build the Driver Image With Debug Prerequisites

Run the following to create the `backend/Dockerfile.driver-debug` file and build the container image
tagged as `kfp-driver:debug`. This container image is based on `backend/Dockerfile.driver` but installs
[Delve](https://github.com/go-delve/delve), builds the binary without compiler optimizations so the binary matches the
source code (via `GCFLAGS="all=-N -l"`), and copies the source code to the destination container for the debugger.
Any changes to the Driver code will require rebuilding this container image.

```bash
make -C backend image_driver_debug
```

Then load the container image in the Kind cluster.

```bash
make -C backend kind-load-driver-debug
```

Alternatively, you can use this Make target that does both.

```bash
make -C kind-build-and-load-driver-debug
```

#### Run the API Server With Debug Configuration

You may use the following VS Code `launch.json` file to run the API server which overrides the Driver
command to use Delve and the Driver image to use debug image built previously.

VSCode configuration:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch API server (Kind) (Debug Driver)",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/backend/src/apiserver",
            "env": {
                "POD_NAMESPACE": "kubeflow",
                "DBCONFIG_MYSQLCONFIG_HOST": "localhost",
                "OBJECTSTORECONFIG_HOST": "localhost",
                "OBJECTSTORECONFIG_PORT": "9000",
                "METADATA_GRPC_SERVICE_SERVICE_HOST": "localhost",
                "METADATA_GRPC_SERVICE_SERVICE_PORT": "8080",
                "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST": "localhost",
                "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT": "8888",
                "V2_LAUNCHER_IMAGE": "ghcr.io/kubeflow/kfp-launcher:master",
                "V2_DRIVER_IMAGE": "kfp-driver:debug",
                "V2_DRIVER_COMMAND": "dlv exec --listen=:2345 --headless=true --api-version=2 --log /bin/driver --"
            }
        }
    ]
}
```

GoLand configuration:
1. Create a new Go Build configuration
2. Set **Run Kind** to Directory and set **Directory** to /backend/src/apiserver absolute path
3. Set the following environment variables

   | Argument                                     | Value     |
   |----------------------------------------------|-----------|
   | POD_NAMESPACE                                | kubeflow  |
   | DBCONFIG_MYSQLCONFIG_HOST                    | localhost |
   | OBJECTSTORECONFIG_HOST                       | localhost |
   | OBJECTSTORECONFIG_PORT                       | 9000      |
   | METADATA_GRPC_SERVICE_SERVICE_HOST           | localhost |
   | METADATA_GRPC_SERVICE_SERVICE_PORT           | 8080      |
   | ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST | localhost |
   | ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT | 8888      |
   | V2_LAUNCHER_IMAGE                            | localhost |
   | V2_DRIVER_IMAGE                              | localhost |
   | V2_DRIVER_COMMAND                            | dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec /bin/driver -- |
4. Set the following program arguments: --config ./backend/src/apiserver/config -logtostderr=true --sampleconfig ./backend/src/apiserver/config/test_sample_config.json

#### Starting a Remote Debug Session

Start by launching a pipeline. This will eventually create a Driver pod that is waiting for a remote debug connection.

You can see the pods with the following command.

```bash
kubectl -n kubeflow get pods -w
```

Once you see a pod with `-driver` in the name such as `hello-world-clph9-system-dag-driver-10974850`, port forward
the Delve port in the pod to your localhost (replace `<driver pod name>` with the actual name).

```bash
kubectl -n kubeflow port-forward <driver pod name> 2345:2345
```

Set a breakpoint on the Driver code in VS Code. Then remotely connect to the Delve debug session with the following VS
Code `launch.json` file:

VSCode configuration:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Connect to remote driver",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/go/src/github.com/kubeflow/pipelines",
            "port": 2345,
            "host": "127.0.0.1",
            "substitutePath": [
                { "from": "${workspaceFolder}", "to": "/go/src/github.com/kubeflow/pipelines" }
            ]
        }
    ]
}
```

GoLand configuration:
1. Create a new Go Remote configuration and title it "Delve debug session"
2. Set **Host** to localhost
3. Set **Port** to 2345

Once the Driver pod succeeds, the remote debug session will close. Then repeat the process of forwarding the port
of subsequent Driver pods and starting remote debug sessions in VS Code until the pipeline completes.

For debugging a specific Driver pod, you'll need to continuously port forward and connect to the remote debug session
without a breakpoint so that Delve will continue execution until the Driver pod you are interested in starts up. At that
point, you can set a break point, port forward, and connect to the remote debug session to debug that specific Driver
pod.

### Using a Webhook Proxy for Local Development in a Kind Cluster

The Kubeflow Pipelines API server typically runs over HTTPS when deployed in a Kubernetes cluster. However, during local development, it operates over HTTP, which Kubernetes admission webhooks do not support (they require HTTPS). This incompatibility prevents webhooks from functioning correctly in a local Kind cluster.

To resolve this, a webhook proxy acts as a bridge, allowing webhooks to communicate with the API server even when it runs over HTTP.

This is used by default when using the `dev-kind-cluster` Make target.

### Deleting the Kind Cluster

Run the following to delete the cluster (once you are finished):

```bash
kind delete clusters dev-pipelines-api
```

## TLS Certificate Rotation (Pod-to-Pod TLS)

### Context

When pod-to-pod TLS is enabled, backend components (`ml-pipeline-apiserver`, `metadata-envoy-deployment`, `metadata-grpc-deployment`, `ml-pipeline-persistenceagent`, `ml-pipeline-scheduledworkflow`, and `ml-pipeline-ui`) use the TLS certificate stored in the Kubernetes TLS Secret `kfp-api-tls-cert`. These certificates expire and must be rotated periodically.


Important operational behaviour: updating a Kubernetes Secret does not cause running pods to read the updated secret automatically. Pods mount Secrets as files or reference them via environment variables at container start time. To pick up rotated certificates, you must restart the pods that read those secrets (a rolling restart of the affected deployments).

This section documents a recommended, minimal process for safely renewing TLS certificates and applying them to a cluster that runs Kubeflow Pipelines.

### Which secrets and components are impacted

TLS certificates are stored in the Kubernetes Secret `kfp-api-tls-cert` in the specified namespace (default `kubeflow`).

Which secrets and components are impacted:

* `ml-pipeline-apiserver`
* `metadata-envoy-deployment`
* `metadata-grpc-deployment`
* `ml-pipeline-persistenceagent`
* `ml-pipeline-scheduledworkflow`
* `ml-pipeline-ui`

> **Note**
> The `cache-server` deployment generally does **not** require the CA cert for inbound TLS from pipeline components; confirm whether your deployment references the TLS secret before restarting it.


> **Note**
> Confirm the exact secret name(s) and deployments used in your cluster before running commands. Example discovery commands:

```bash
kubectl get secrets -n <namespace>

# Find deployments referencing the secret
kubectl -n <namespace> get deploy -o name \
  | xargs -n1 kubectl -n <namespace> get deploy -o yaml \
  | grep -C3 "name: <secret-name>" || true
```

### Certified rotation procedure

#### 1. Prepare/obtain new cert and key
Generate or obtain new cert files: `server.crt` and `server.key` (PEM encoded).

#### 2. Update the Kubernetes TLS secret

Replace <namespace> and <secret-name> with your cluster's values:

```bash
kubectl create secret tls <secret-name> \
  --cert=server.crt \
  --key=server.key \
  --dry-run=client -o yaml | kubectl apply -f -
```

Example:

 ```bash
 kubectl create secret tls kfp-api-tls-cert \
    --cert=server.crt \
    --key=server.key \
    --dry-run=client -o yaml | kubectl apply -f -
 ```

#### 3. Restart affected deployments (rolling restart)

Restart the deployments so they mount/read the updated secret:

```bash
kubectl rollout restart deploy -n <namespace> ml-pipeline-apiserver
kubectl rollout restart deploy -n <namespace> ml-pipeline-persistenceagent
kubectl rollout restart deploy -n <namespace> metadata-envoy-deployment
kubectl rollout restart deploy -n <namespace> metadata-grpc-deployment
kubectl rollout restart deploy -n <namespace> ml-pipeline-scheduledworkflow
kubectl rollout restart deploy -n <namespace> ml-pipeline-ui
```

Or restart all pipeline-related deployments discovered earlier.

#### 4. Verify rollout and pod readiness

```bash
kubectl get pods -n <namespace>
kubectl rollout status deploy/ml-pipeline-apiserver -n <namespace>
```

Check logs for errors:

```bash
kubectl describe secret <secret-name> -n <namespace>
```

Optional: connect to the API server over TLS (from inside cluster or via port-forward) to confirm certificate in use.

Example (port-forward the API server service and test TLS locally):

```bash
kubectl -n <namespace> port-forward svc/ml-pipeline 8443:443

# Confirm the certificate presented by the server
openssl s_client -connect localhost:8443 -servername ml-pipeline 2>/dev/null | openssl x509 -noout -subject -issuer -dates
```

### 5. Automate restart after secret update

Kubernetes does not automatically restart pods when a Secret changes. Options:

Example: manually update the secret and restart the dependent deployments

```bash
NAMESPACE="<namespace>"
SECRET_NAME="kfp-api-tls-cert"

# Update the TLS secret (same command as above)
kubectl create secret tls "${SECRET_NAME}" \
  --cert=server.crt \
  --key=server.key \
  --dry-run=client -o yaml | kubectl apply -f - -n "${NAMESPACE}"

# Restart dependent deployments (rolling restart)
kubectl rollout restart deploy/ml-pipeline-apiserver -n "${NAMESPACE}"
kubectl rollout restart deploy/ml-pipeline-persistenceagent -n "${NAMESPACE}"
kubectl rollout restart deploy/metadata-envoy-deployment -n "${NAMESPACE}"
kubectl rollout restart deploy/metadata-grpc-deployment -n "${NAMESPACE}"
kubectl rollout restart deploy/ml-pipeline-scheduledworkflow -n "${NAMESPACE}"
kubectl rollout restart deploy/ml-pipeline-ui -n "${NAMESPACE}"

# Wait for rollouts
kubectl rollout status deploy/ml-pipeline-apiserver -n "${NAMESPACE}"
```

### Best practices & notes

* Rotation interval: rotate certificates before they expire. Track expiry with the following command:

```bash
openssl x509 -enddate -noout -in server.crt
```

* **Automation**: if using `cert-manager`, automate issuance and renewal, and consider automating rollout restarts (see below).
* **Minimize disruption**: perform rollouts during maintenance windows or low activity windows.
* **Auditing**: record rotation times and certificate fingerprints for traceability.
* **Validate**: after restarts, verify connectivity between components and watch logs for TLS handshake issues.
* Use an operator or controller that watches the Secret and triggers kubectl `rollout restart` on dependent deployments.
* Use `cert-manager` and the common checksum/annotation pattern to trigger a redeploy when the Secret changes. For example, add an annotation that contains a checksum of the secret; updating the annotation forces a rolling update:

## Contributing
### Code Style

Backend codebase follows the [Google's Go Style Guide](https://google.github.io/styleguide/go/). Please, take time to get familiar with the [best practices](https://google.github.io/styleguide/go/best-practices). It is not intended to be exhaustive, but it often helps minimizing guesswork among developers and keep codebase uniform and consistent.

We use [golangci-lint](https://golangci-lint.run/) tool that can catch common mistakes locally (see detailed configuration [here](https://github.com/kubeflow/pipelines/blob/master/.golangci.yaml)). It can be [conveniently integrated](https://golangci-lint.run/usage/integrations/) with multiple popular IDEs such as VS Code or Vim.

Finally, it is advised to install [pre-commit](https://pre-commit.com/) in order to automate linter checks (see configuration [here](https://github.com/kubeflow/pipelines/blob/master/.pre-commit-config.yaml))
