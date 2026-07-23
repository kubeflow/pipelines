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

See the [Local testing](../AGENTS.md#local-testing) section in AGENTS.md.

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

## API Server Configuration

### Driver Pod Labels and Annotations

The API server supports configuring custom labels and annotations for driver pods through the configuration file or the Kubernetes ConfigMap. This is useful for integration with service mesh (Istio), monitoring systems, or other infrastructure requirements.

**Configuration via config.json:**
```json
{
  "DRIVER_POD_LABELS": "{\"sidecar.istio.io/inject\":\"true\",\"app\":\"ml-pipeline-driver\"}",
  "DRIVER_POD_ANNOTATIONS": "{\"proxy.istio.io/config\":\"{\\\"holdApplicationUntilProxyStarts\\\":true}\"}"
}
```

Write each value as a JSON string here, the same way it is written in the ConfigMap. Writing it
as a JSON object instead is refused at startup, with an error naming the string form. The
configuration loader folds the keys of such an object to lower case while reading the file, so
`example.com/BuildID` would reach the driver pod as `example.com/buildid`, and two keys that
differ only in case would be merged into one before anything could check them, taking one of
the values with them. Where both of those keys need lowering, which one survives follows map
iteration order, so the same file could start the API server on one restart and stop it on the
next. The string form is parsed directly and keeps the keys exactly as written.

This file supplies the built in default and is baked into the API server image. On a standard
install it is not the setting an operator changes, because the Deployment always passes both
keys in from the ConfigMap as environment variables, and an environment variable takes
precedence over this file even when it is empty. Use the ConfigMap below unless you build your
own image and also stop the Deployment from passing those variables.

**Configuration via Kubernetes ConfigMap (`pipeline-install-config`):**
```yaml
DRIVER_POD_LABELS: '{"sidecar.istio.io/inject":"true"}'
DRIVER_POD_ANNOTATIONS: '{"proxy.istio.io/config":"{\"holdApplicationUntilProxyStarts\":true}"}'
```

Every entry inside the object must have a string value. Write `"true"` rather than the bare boolean `true`, and never `null`. Three things that look similar behave differently: a value of `"null"` is refused, an entry such as `{"app": null}` is refused, and a configuration file key whose value is a bare `null` is read as if the key were absent, because the configuration loader reports it that way and gives no means to tell the two apart. The configuration is validated at API server startup: malformed JSON, any value that is not a string, and label keys, label values, or annotation keys that Kubernetes would reject all cause startup to fail with a descriptive error rather than being silently ignored. Annotation values are free form, so a value such as an inline JSON document is accepted.

Startup validation also holds the annotations to the 256 KiB Kubernetes allows across one object's annotations, though the driver pod carries a few more that the compiler adds, so passing that check bounds your own input rather than guaranteeing the pod will be accepted. Keep the values far below it in any case. Each setting arrives as one environment variable holding its whole JSON document, and Linux caps a single environment string at 32 memory pages, which is around 128 KiB on the usual 4 KiB page size and more where pages are larger. A value over that cap stops the API server from starting at all, so it cannot report anything about it.

**Terminating an injected Istio sidecar:**

Where the proxy is injected as an ordinary sidecar container, one more annotation is needed.
Argo shuts an injected sidecar down by running `/bin/sh -c 'kill 1'` inside it, which the Istio
proxy does not act on, so the proxy keeps running after the driver has finished and the pod
stays in `Running`. Argo lets you replace that command per container, and the default KFP
install already grants the `pods/exec` permission it needs through the `pipeline-runner` Role.
If the driver runs under a service account with narrower permissions, grant `pods/exec` there
as well.

```yaml
DRIVER_POD_LABELS: '{"sidecar.istio.io/inject":"true"}'
DRIVER_POD_ANNOTATIONS: '{"proxy.istio.io/config":"{\"holdApplicationUntilProxyStarts\":true}","workflows.argoproj.io/kill-cmd-istio-proxy":"[\"pilot-agent\", \"request\", \"POST\", \"quitquitquit\"]"}'
```

See the Argo Workflows [sidecar injection](https://argo-workflows.readthedocs.io/en/latest/sidecar-injection/) guide for the full description of the problem.

Note: Labels and annotations with the prefix `pipelines.kubeflow.org/` are reserved and will be filtered out to prevent overriding system metadata.

Changes to driver pod configuration require an API server restart, and each API server reads the configuration once as it starts, so what a run gets depends on which one compiles it.

A recurring run that did not pin its pipeline is compiled through the runs API each time it triggers, by whichever API server handles that call. Once the restart has completed, that is the new configuration. During the restart itself it need not be: the default rollout brings a replacement up before retiring the one it replaces, so for a short window both are serving and the scheduler reaches either.

A recurring run that pinned its pipeline is instead recompiled by the reconciliation an API server runs once as it starts, and its ScheduledWorkflow is updated when the resulting spec differs. That reconciliation stops at the first recurring run it cannot process and is not retried, so if one fails, for example because its pipeline version cannot be read, the ones after it keep their previous specification until the next restart. A run triggered while the reconciliation is still in flight also uses the previous specification.

The short version is that a restart is what applies a change, and a run created while the restart is in progress may still be built from the configuration you replaced.

The compiler adds `sidecar.istio.io/inject: "false"` as an annotation to every template that does not already carry that annotation, which includes the driver templates. Setting the injection label above is enough, because Istio prefers the label over the annotation when both are present, and the annotation form is deprecated. Setting `sidecar.istio.io/inject` through `DRIVER_POD_ANNOTATIONS` also works, since the compiler leaves an annotation alone once it is set.

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

## Contributing
### Code Style

Backend codebase follows the [Google's Go Style Guide](https://google.github.io/styleguide/go/). Please, take time to get familiar with the [best practices](https://google.github.io/styleguide/go/best-practices). It is not intended to be exhaustive, but it often helps minimizing guesswork among developers and keep codebase uniform and consistent.

We use [golangci-lint](https://golangci-lint.run/) tool that can catch common mistakes locally (see detailed configuration [here](https://github.com/kubeflow/pipelines/blob/master/.golangci.yaml)). It can be [conveniently integrated](https://golangci-lint.run/usage/integrations/) with multiple popular IDEs such as VS Code or Vim.

Finally, it is advised to install [pre-commit](https://pre-commit.com/) in order to automate linter checks (see configuration [here](https://github.com/kubeflow/pipelines/blob/master/.pre-commit-config.yaml))
