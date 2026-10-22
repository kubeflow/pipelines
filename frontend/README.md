# Kubeflow Pipelines Frontend

This section of the codebase contains the Kubeflow Pipelines (KFP) Frontend.

## Current Stack

- React 19 with TypeScript on Vite 7
- MUI v5 with Emotion
- TanStack Query v5
- React Router v8 (declarative hash routing)
- Vitest with Testing Library v16 for UI tests
- Vitest for frontend server tests
- Storybook 10 for component development

## Quick Start Development

This guide will get you started with development on KFP standalone mode.

### Prerequisites

You will need the following installed in your environment:

* [Docker]
* [Kubectl]
* [Kind]
* [Kustomize]
* [Node] version specified in the [.nvmrc]
* npm version specified in [package.json]

> [!Note]
> MAC users have reported positive experiences using [Docker + Colima] when using Kind environments. Consider
> using a similar setup if you are on a MAC and encountering issues with Docker VM.

### Deploy KFP

Clone and then deploy KFP:

```bash
git clone https://github.com/kubeflow/pipelines.git ${WORKING_DIRECTORY}
cd ${WORKING_DIRECTORY}
make -C backend kind-cluster-agnostic
```

The above command will deploy KFP in standalone mode.
It uses sensible defaults (MySQL database) for quick setup. If you need to customize the backend configuration (e.g., use PostgreSQL instead of MySQL), please refer to the [backend README](../backend/README.md) for advanced deployment options using `dev-kind-cluster`.


You can access the KFP UI by port-forwarding the KFP UI Kubernetes Service:

```bash
kubectl -n kubeflow  port-forward svc/ml-pipeline-ui 3000:80
```

Navigate to [http://127.0.0.1:3000] to view the UI. You will see something like the following:

![KFP UI](docs/images/kfp-ui.png)

Try uploading and running a pipeline and confirm it works! You can use one of the already uploaded templates. You can also follow the [KFP docs] for instructions on how to write and submit a pipeline. You can use [http://127.0.0.1:3000] as your `Client(host=...)` value.

### Local Development

Now that you have had a chance to check out the UI, we will now scale this UI down and run the UI ourselves locally.

Scale the UI down by running the following:

```bash
# End the port-forwarding by pressing ctrl+D in your terminal, then run:
kubectl -n kubeflow scale --replicas=0 deployment/ml-pipeline-ui
```

You can confirm that the previous [http://127.0.0.1:3000] link no longer works.

Now navigate to the KFP frontend folder, install and build your NPM dependencies:

```bash
cd ${WORKING_DIRECTORY}/frontend
npm install --global "$(node -p 'require("./package.json").packageManager')"
npm ci
npm run build
```

Now run the following:

```bash
npm run start:proxy-and-server
```

You should see the following output

```bash
Server listening at http://localhost:3001
```

Follow this link, and you should be directed to the KFP UI the same as before, except this time you are using the UI running in your local environment!

If you enjoy hot reloading when developing the client side React code, you can subsequently run the following command:

```bash
npm run start
```

You should see output indicating the Vite dev server is running, for example:

```bash
VITE v7.x ready in ...
➜  Local:   http://localhost:3000/
...
```

Follow this link, it should also take you to the same UI. The difference here is that whenever you change client side (React) code locally, you will automatically get the new changes in your browser without having to restart your server.

The local dev bootstrap runs under React Strict Mode. Vitest UI tests are configured to do the same through Testing Library's global `reactStrictMode` setting so direct `render()` calls match dev behavior. Production builds remain outside Strict Mode.

### Mock backend shortcut

For fixture-backed client work that does not need a Kubernetes cluster, run:

```bash
npm run mock:api
npm run start
```

The mock backend serves the primary v2 Pipelines, Experiments, Runs, and Recurring Runs list pages with deterministic fixture data. Use `npm run start:proxy-and-server` against a real KFP deployment when validating native tasks and artifacts, pod logs, authentication, or backend behavior beyond those fixtures.

## Visual Regression Testing

When making UI changes, use the smoke test tool to capture screenshots and generate side-by-side comparisons against a base branch. This catches layout regressions, styling issues, and unintended visual changes before they reach review.

### Quick screenshot of your dev server

The fastest workflow — point the tool at your already-running `npm start` server:

```bash
node scripts/ui-smoke-test/smoke-test-runner.js --current-only --use-existing --url http://localhost:3000
```

Screenshots are saved to `.ui-smoke-test/screenshots/pr/`.

### Compare your branch against master

The full workflow — detects changed backend components, ensures a Kind cluster, builds both frontends, captures screenshots from both, and generates a side-by-side comparison with diff percentages:

```bash
node scripts/ui-smoke-test/smoke-test-runner.js --compare master
```

If your PR only touches frontend code, the backend rebuild is auto-skipped since no backend components changed.

### Compare someone else's PR

Fetch and test a PR you don't have checked out locally:

```bash
node scripts/ui-smoke-test/smoke-test-runner.js --compare master --pr 12756
```

Results are saved to `.ui-smoke-test/screenshots/comparison/`. See [scripts/ui-smoke-test/README.md] for the full command reference, troubleshooting, and architecture details.

## Contributing

For a more comprehensive guide on contributing, please read [CONTRIBUTING.md].

<!REFERENCES>
[Docker]: https://docs.docker.com/engine/install/
[Kind]: https://kind.sigs.k8s.io/#installation-and-usage
[Kustomize]: https://kustomize.io
[Node]: https://www.npmjs.com/package/node
[.nvmrc]: .nvmrc
[package.json]: package.json
[CONTRIBUTING.md]: CONTRIBUTING.md
[scripts/ui-smoke-test/README.md]: scripts/ui-smoke-test/README.md
[http://127.0.0.1:3000]: http://127.0.0.1:3000
[Kubectl]: https://kubernetes.io/docs/tasks/tools/#kubectl
[Docker + Colima]: https://github.com/abiosoft/colima?tab=readme-ov-file#docker
[sample pipeline]: https://raw.githubusercontent.com/kubeflow/pipelines/refs/heads/master/sdk/python/test_data/pipelines/pipeline_with_env.py
[sample pipeline in yaml]: https://raw.githubusercontent.com/kubeflow/pipelines/refs/heads/master/sdk/python/test_data/pipelines/pipeline_with_env.yaml
[KFP docs]: https://www.kubeflow.org/docs/components/pipelines/getting-started/

## Multi-user artifact ownership

These policies describe master and the proposed 3.0 storage-authorization work.
They are not 2.18 migration instructions: the pending
[2.18 compatibility rollback](https://github.com/kubeflow/pipelines/pull/15044)
defers stronger ownership enforcement and removes its unreleased audit control.
See [release scope and first adoption](../docs/operator-guides/artifact-authorization.md#release-scope-and-first-adoption)
for the separate 3.0 migration work.

With `ENABLE_AUTHZ=true`, explicit `KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE=key-prefix`
or `auto` handling an unsupported custom store, artifact previews and downloads
require an object key under
`private-artifacts/<namespace>/...`, or the equivalent prefix configured by the operator through
`ARTIFACT_NAMESPACE_KEY_PREFIX`. This applies to MinIO, S3, GCS, and HTTP(S) artifacts, including
requests made with `ARTIFACTS_SERVICE_PROXY_ENABLED=true`. A namespace-scoped metadata record does
not establish ownership of an arbitrary object: a pipeline can import an existing URI.

**Compatibility change where prefix enforcement applies:** custom-root objects outside that
namespace prefix return HTTP 403. Deployments with isolated namespace storage credentials can
select [storage authorization](#storage-authoritative-artifact-access) to retain existing roots.
For shared credentials, configure
pipeline roots with the tenant namespace segment and write or copy authorized artifacts to those
paths. Updating metadata alone does not move existing objects. Keep storage credentials and tenant
proxy access isolated; the path policy does not replace storage or network authorization.

For example, namespace `team-a` can serve `s3://bucket/private-artifacts/team-a/run/model`, but not
`s3://bucket/shared/model`. A different common prefix, such as `tenant-data`, may be configured by
the operator; the next path segment must still be the authorized namespace. Do not disable
authorization to restore custom-root access in a multi-user deployment.

With `ENABLE_AUTHZ=true`, HTTP(S) artifacts use the shared UI's configured `HTTP_BASE_URL` and,
when needed, its `HTTP_AUTHORIZATION_KEY` and `HTTP_AUTHORIZATION_DEFAULT_VALUE`, even when the
namespace artifact proxy is enabled. Without a shared UI `HTTP_BASE_URL`, these requests return
HTTP 400. The shared UI checks redirect targets itself rather than forwarding HTTP(S) requests to
tenant services that may run older redirect handling code.
Redirects must stay on the configured origin and under the same namespace prefix. Cross-origin
signed URLs and same-origin signed URLs outside that prefix return HTTP 403. Standalone mode keeps
its existing redirect behavior.

Standalone mode and namespace-proxied volume artifacts are unchanged. Under `key-prefix`, the `artifact-only` mode
(also accepted as legacy `mlmd-only`) requires metadata evidence in addition to the namespace
path policy; it does not enable arbitrary custom roots.

## Storage-authoritative artifact access

`KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE=auto` is the default. Managed
`minio://mlpipeline/...` artifacts and administrator-approved external S3
credential-file brokers use namespace storage identities after trusted controller
readiness, without moving already authorized objects or requiring metadata records.
The multi-user overlay enables managed proxies. Pending readiness, revoked external
approval or a failed lookup returns an actionable HTTP 503, never a shared-credential
fallback. Unapproved custom layouts retain prefix enforcement. Both `auto` and explicit `storage`
retain ownership/prefix checks for `minio://mlpipeline/artifacts/`, which may hold private V1
outputs from before the SeaweedFS switch despite its stock shared storage grant.
A ready record does not make those historical objects public. It also does not
certify that all effective storage grants are narrow: retained administrator
grants and the intentionally all-namespace writable `shared/` root become
readable to namespace viewers. Review effective grants, viewer roles and existing
roots before rollout; use `key-prefix` until broader viewer access is intended.

Custom stores without the controller's IAM verification API can stay pending even
with pre-created namespace Secrets. Select `key-prefix` before rollout to retain
ownership checks, or explicitly use `storage` only after verifying genuinely
restricted identities and the namespace receiver. See the
[legacy-root and custom-store guidance](../docs/operator-guides/artifact-authorization.md).

For the [external broker adapter](../docs/operator-guides/artifact-authorization.md#external-storage-with-rotating-broker-credentials),
the administrator can configure a reusable `adoption-policies.json` policy in
`kfp-artifact-storage-config`. Trusted provisioning automation publishes current
namespace UID/principal bindings; labelled binding/template ConfigMaps in the
installation namespace supply the approved identity and receiver configuration.
KFP enrolls matching existing and new namespaces automatically. Explicit
`approvals.json` entries remain available; overlapping authorities fail closed.
The controller reconciles receiver rollout; temporary credentials refresh from
the approved file without receiver restarts. Administrators must verify the
provisioner and its storage grants; this does not discover arbitrary workload
identities or verify their external IAM policies.

Explicit `storage` remains available for reviewed custom identities and trusted
namespace launcher-provider configuration; explicit `key-prefix` retains the
prefix/metadata policy. Versioned receivers enforce namespace binding while legacy
routes preserve receiver-protocol compatibility during normal rolling deployment
between compatible UI/API versions. This does not restore APIs removed in 3.0 or
establish a complete 2.18-to-3.0 upgrade. No manual UI/controller ordering or opt-in
is needed for the supported managed path; reads can need retry during convergence. See
[enforcement scope and rollout](../docs/operator-guides/artifact-authorization.md).
