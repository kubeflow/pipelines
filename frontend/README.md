# Native KFP frontend

The app and offline viewer share graph, artifact-card, inspector and log widgets.
Both web entrypoints load `src/argo.tsx`. Browser adapters use Kubernetes and Argo
Server; there is no KFP REST client, Node application server or mock backend.

Use the Node/npm pins in `.nvmrc` and `package.json`:

```sh
npm ci
npm run test:ci
```

`npm start` uses the deployed authenticated proxy at `http://127.0.0.1:3003` by
default (`KFP_NATIVE_PROXY` overrides it). See `../samples/argo-native/README.md`
for deployment, namespace-scoped authentication and real acceptance tests.

Rebuild the checked-in SDK viewer assets with `npm run build:viewer` after changing
shared UI modules or dependencies. The native bundle and offline/sandbox browser
tests exercise the actual built code, including API and credential boundaries.
