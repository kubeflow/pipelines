# UI modernization deployment and browser baseline

Source baseline: [`02cbc725ac9ddcd950f4400d8355dd78bfcd6c57`](https://github.com/kubeflow/pipelines/tree/02cbc725ac9ddcd950f4400d8355dd78bfcd6c57) for [#14572](https://github.com/kubeflow/pipelines/issues/14572). Captured 2026-09-26. Source contracts and registry identity are recorded below; live deployment compatibility and rollback remain unverified.

## Deployment contract

| Surface                | Current contract and source                                                                                                                                                                                                                                                                                                                                                                                  | Required migration evidence                                                                                                                       |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Container              | [Dockerfile](../../Dockerfile) builds the client and Node server together, serves `/client` from `/server`, exposes 3000, and starts `node dist/server.js ../client/ 3000`. It records frontend commit, tag and build date. [.nvmrc](../../.nvmrc) and [package.json](../../package.json) pin Node `24.14.0` and npm `11.17.0`.                                                                              | Build the existing image target with the pinned toolchain; verify startup, assets and build identity.                                             |
| Service and deployment | [Service](../../../manifests/kustomize/base/pipeline/ml-pipeline-ui-service.yaml) port 80 targets container 3000. [Deployment](../../../manifests/kustomize/base/pipeline/ml-pipeline-ui-deployment.yaml) retains non-root execution, runtime-default seccomp, dropped capabilities, mounted UI config, health probes and `Recreate` strategy.                                                               | Compare rendered manifests; retain names, ports, probes, configuration and security context.                                                      |
| URLs and assets        | [Server](../../server/app.ts) serves root and `/pipeline`; [Vite](../../vite.config.mts) uses relative assets (`base: './'`); [client](../../src/index.tsx) uses `HashRouter`.                                                                                                                                                                                                                               | Test direct entry, refresh, copied hash routes and relative assets at root and `/pipeline/`, including through the supported reverse proxy.       |
| Backend and health     | Server retains API proxying and `apis/v2beta1`. [Health handler](../../server/handlers/healthz.ts) returns frontend/backend build identities, multi-user mode, pipeline store and `apiServerReady`.                                                                                                                                                                                                          | Record both immutable image identities and health metadata. HTTP 200 alone is insufficient: check `apiServerReady` and representative data flows. |
| Standalone             | Default [deployment configuration](../../server/configs.ts) renders the full navigation.                                                                                                                                                                                                                                                                                                                     | Verify list/detail/create flows and error recovery in standalone mode.                                                                            |
| Kubeflow embedding     | [Multi-user overlay](../../../manifests/kustomize/base/installs/multi-user/pipelines-ui/deployment-patch.yaml) selects Kubeflow deployment and authorization. [HTML handler](../../server/handlers/index-html.ts) injects deployment/navigation flags and dashboard client script. Navigation defaults to hidden in Kubeflow; [KubeflowClient](../../src/lib/KubeflowClient.tsx) supplies namespace context. | Preserve HTML runtime placeholders, bootstrap behavior, embedding, namespace switching, hidden navigation and permitted/denied operations.        |
| Artifacts and viewers  | [Server](../../server/app.ts) retains artifact, log, pod and TensorBoard handlers. Deployment uses the existing shared TensorBoard signing-key contract.                                                                                                                                                                                                                                                     | Verify previews, downloads, logs and TensorBoard after rollout and rollback without changing identity, proxy or signing-key configuration.        |
| Rollout lifecycle      | Deployment uses `Recreate` to avoid overlapping pods during adoption of the existing shared TensorBoard key.                                                                                                                                                                                                                                                                                                 | Preserve strategy and signing-key lifecycle; account for controlled pod replacement when qualifying rollout and rollback.                         |

## Browser compatibility

The current [Browserslist configuration](../../package.json) uses `supports es6-module` for production and the latest Chrome, Firefox and Safari for development. Vite targets `es2015`. Neither establishes tested CSS compatibility or a versioned product support policy. The existing [UI smoke harness](../../scripts/ui-smoke-test/README.md) describes Playwright headless Chrome capture, which does not establish Firefox or Safari acceptance.

- [ ] Agree on explicit supported browser versions before adopting the new UI foundation.
- [ ] Check proposed dependencies and CSS features against that browser floor.
- [ ] Verify core workflows in Chromium, Firefox and WebKit/Safari, recording exact browser versions and viewport sizes. Distinguish WebKit harness results from actual Safari qualification.

## Immutable rollback candidate

The baseline [image configuration](../../../manifests/kustomize/base/pipeline/kustomization.yaml) selects `2.17.1` for both the UI and API images. A public registry read on 2026-09-26 resolved the UI tag to this OCI index:

```text
ghcr.io/kubeflow/kfp-frontend@sha256:f30091c0bb8e413f231479256f4e625bd48525bbe7041c550757b41a40c7e055
```

| Runtime platform | Child digest                                                              |
| ---------------- | ------------------------------------------------------------------------- |
| linux/amd64      | `sha256:f85a92edcba1019071599c1817f86b84d563d72ad06c92b2678f17d6362a600e` |
| linux/arm64      | `sha256:8bb4a7c21460b4cdb845113545090a6c49293f6444858637a3266d634993c531` |

The index also contains attestation descriptors. Registry identity and runtime architectures are verified. **This candidate is not yet a known-good rollback image for the modernization backend.** The release tag does not prove correspondence to source baseline `02cbc725`, preservation of its server behavior, or compatibility with the target backend.

## Deployment and rollback evidence required

- [ ] Record immutable UI/backend image digests, source/build identities, deployment mode, runtime platform and health metadata from a successful baseline deployment.
- [ ] Exercise the [route/action inventory](route-inventory.md) against that deployment; retain results for standalone and Kubeflow embedding, relevant pipeline stores, namespace/auth behavior, deep links, creation, run/task inspection, artifacts, logs and viewers.
- [ ] Qualify the previous and modernized UI images against the same unchanged backend and deployment contract. Preserve existing browser preferences through the transition.
- [ ] Rehearse restoring the previous UI image and rerun representative workflows. Record image identities, rollout outcome, health and functional results. The UI cutover must not require a backend migration or destructive storage change.

These deployment, browser and rollback checklists remain open. Source review and registry metadata are the evidence available in this document; no live compatibility or rollback test result is claimed.
