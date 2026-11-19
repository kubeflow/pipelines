# Run a Pipeline

[Compile the pipeline](compile-a-pipeline.md) to native Argo YAML, then submit it
through Kubernetes, Argo CLI or the native UI. This execution path has no KFP API
server and does not provide `kfp.Client` submission compatibility.

## Native UI

For the local deployment, open
`http://127.0.0.1:3003/argo.html?namespace=kfp-argo` and connect with a
namespace-scoped token. The complete deployment and authentication commands are
in `samples/argo-native/README.md` in the repository.

1. Open **Pipelines and versions** and upload the compiled WorkflowTemplate YAML.
2. Select a version, its parameters, and an optional experiment.
3. Choose **Start run**.
4. Open **Runs** to inspect the graph, task metadata, typed artifacts and logs.
5. Check specific runs to compare their metrics.

Experiments are ConfigMaps and membership is a workflow label. Deleting an
experiment does not delete its runs. Recurring runs are ordinary CronWorkflows;
new schedules start suspended and can be enabled from **Recurring runs**.

## Kubernetes and Argo CLI

```sh
kubectl --context orbstack -n kfp-argo apply -f pipeline.yaml
argo submit -n kfp-argo --from workflowtemplate/NAME_IN_PIPELINE_YAML --watch
```

For a standalone `Workflow` instead, use `kubectl create -f run.yaml` or
`argo submit run.yaml`. A `generateName` resource is created, not updated with
`kubectl apply`.

When node status is offloaded, use Argo Server for workflow details and logs,
not only the Kubernetes API. For the local authenticated proxy:

```sh
export ARGO_SERVER=127.0.0.1:3003
export ARGO_BASE_HREF=/argo
export ARGO_HTTP1=true
export ARGO_SECURE=false
export ARGO_TOKEN="Bearer $(kubectl --context orbstack -n kfp-argo create token argo-ui-editor --duration=2h)"
argo list -n kfp-argo
```

These HTTP settings are for a loopback port-forward only. Production deployments
need authenticated same-origin ingress and normal TLS validation. Do not embed a
cluster-admin token in the browser.

## Run identity and history

A workflow's Kubernetes UID is its run ID. The UI deduplicates live and archived
results by UID and uses Argo's archive endpoints after the live Workflow is deleted.
Artifact and cache metadata retain the original producing run/task identity.

The local verification script exercises archive access, offloaded status, native
approval gates, denied mutations, recurring-run persistence, cache reuse and
artifact retention through the actual deployed APIs.
