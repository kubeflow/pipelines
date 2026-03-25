# KFP Frontend Guide

Prerequisites, setup, development workflows, technologies, commands, code generation, testing, feature flags, and troubleshooting for working on `frontend/`.

**See also:** [protobuf-build.md](protobuf-build.md) (frontend proto generation), [testing-ci.md](testing-ci.md) (CI pipeline, formatting checks)

---

## Prerequisites

- Node.js version specified in `frontend/.nvmrc` (currently v24.14.0)
- Java 8+ (required for `java -jar swagger-codegen-cli.jar` when generating API clients)
- Use [nvm](https://github.com/nvm-sh/nvm) or [fnm](https://github.com/Schniz/fnm) for Node version management:

  ```bash
  # With fnm (faster)
  fnm install 24.14.0 && fnm use 24.14.0
  # With nvm
  nvm install 24.14.0 && nvm use 24.14.0
  ```

## Setup and installation

```bash
cd frontend
npm ci  # Install exact dependencies from package-lock.json
```

## Development workflows

### Local development with mock API

Quick start for UI development without backend dependencies:

```bash
npm run mock:api    # Start mock backend server on port 3001
npm start           # Start Vite dev server on port 3000 (hot reload)
```

### Local development with real cluster

For full integration testing against a real KFP deployment:

1. **Single-user mode**:

   ```bash
   # Deploy KFP standalone (see backend-guide.md Local cluster deployment)
   make -C backend kind-cluster-agnostic

   # Scale down cluster UI
   kubectl -n kubeflow scale --replicas=0 deployment/ml-pipeline-ui

   # Start local development
   npm run start:proxy-and-server  # Proxy to cluster + hot reload
   ```

2. **Multi-user mode**:

   ```bash
   export VITE_NAMESPACE=kubeflow-user-example-com
   npm run build
   # Install mod-header Chrome extension for auth headers
   npm run start:proxy-and-server
   ```

## Key technologies and architecture

- **React 18** with TypeScript
- **MUI v5** (`@mui/material`) for components
- **React Router v5** for navigation
- **Dagre** for graph layout visualization
- **D3** for data visualization
- **Vitest + Testing Library** for UI testing
- **Jest** for frontend server tests (UI tests migrated off Jest/Enzyme)
- **Prettier + ESLint** for code formatting/linting
- **Storybook** for component development
- **Tailwind CSS** for utility-first styling

### React effect discipline

When writing or reviewing React code in `frontend/src`:

- Treat every `useEffect` as an escape hatch for external synchronization.
  Valid cases include browser APIs, timers, subscriptions, storage, imperative DOM APIs,
  or coordinating with async systems that exist outside React render.
- Do not use `useEffect` for derived UI state.
  If state can be computed from props, query results, or existing state, derive it during render
  or with a memoized pure helper.
- Do not put user-action behavior in an effect.
  Navigation, snackbars, dialog open/close behavior, and mutation success/failure handling should
  live in event handlers or mutation callbacks.
- Avoid effect chains where one effect sets state that triggers another.
  If a page needs several coupled state transitions, prefer a reducer or a single explicit update path.
- Preserve user-controlled state across refreshes and refetches.
  Query refreshes must not silently reset selected runs, selected artifacts, typed names, or toggles
  unless that reset is an explicit product requirement.
- Treat `eslint-disable react-hooks/exhaustive-deps` as a code smell.
  Only keep it when the invariant is documented in code and covered by a targeted test.
- During reviews, classify each new or changed effect as one of:
  `external sync`, `derived state`, `event-driven`, `state reset`, or `effect chain`.
  Anything except `external sync` requires explicit justification in the diff or review notes.

## Essential commands

- `npm start` - Start Vite dev server with hot reload (port 3000)
- `npm run start:proxy-and-server` - Full development with cluster proxy
- `npm run mock:api` - Start mock backend API server (port 3001)
- `npm run build` - Production build
- `npm run test` - Run Vitest UI tests (same as `test:ui`, with `LC_ALL` set)
- `npm run test:ui` - Run Vitest UI tests
- `npm run test:ui:coverage` - Run Vitest UI tests with coverage
- `npm run test:ui:coverage:loop` - Run Vitest UI coverage with a capped worker count (stability loop)
- `npm run test -u` - Update Vitest snapshots
- `npm run lint` - Run ESLint
- `npm run typecheck` - Run TypeScript typecheck (`tsc --noEmit`)
- `npm run check:react-peers` - Enforce lockfile React peer compatibility for current target (React 18 today)
- `npm run check:react-peers:18` - Preview lockfile React peer compatibility against React 18
- `npm run check:react-peers:19` - Preview lockfile React peer compatibility against React 19
- `npm run format:check` - Fast pre-push check for frontend formatting; for a small TS/TSX diff, `npx prettier --check <changed files>` is the quickest equivalent
- `npm run format` - Format code with Prettier
- `npm run storybook` - Start Storybook on port 6006

## Code generation

The frontend includes several generated code components:

- **API clients**: Generated from backend Swagger specs

  ```bash
  npm run apis        # Generate v1 API clients
  npm run apis:v2beta1 # Generate v2beta1 API clients
  ```

  Note: Ensure `swagger-codegen-cli.jar` is available to `java -jar` when running from `frontend/`
  (e.g., place the JAR in `frontend/` or reference a full path).

- **Protocol Buffers**: Generated from proto definitions

  ```bash
  npm run build:protos              # MLMD protos
  npm run build:pipeline-spec       # Pipeline spec protos
  npm run build:platform-spec:kubernetes-platform # K8s platform spec
  ```
### Effect-focused frontend verification

When changing an effect-heavy frontend component, add or run the smallest relevant regression test:

- mutation success runs exactly once even if related async work resolves later
- refresh/refetch does not overwrite user selection or form edits unless intended
- error banners and dialogs clear after a successful retry or recovery path
- mount-time logic does not emit parent callbacks unless the component contract explicitly requires it

## Testing

- **UI tests**: `npm run test:ui` or `npm test` (Vitest + Testing Library)
- **Server tests**: `npm run test:server:coverage` (Jest)
- **Coverage**: `npm run test:ui:coverage` (Vitest) + `npm run test:coverage` (Vitest UI + Jest server)
- **Stability loop**: `npm run test:ui:coverage:loop` (Vitest coverage with capped workers)
- **CI pipeline**: `npm run test:ci` (format check + lint + typecheck + lockfile React peer check + Vitest UI coverage + Jest coverage)
- **Snapshot tests**: Auto-update with `npm test -u` or `npm run test:ui -- -u` (Vitest)
- **Integration tests**: See `test/frontend-integration-test/README.md` for the containerized local flow. Supported debug env vars include `DEBUG=1`, `HEADLESS=false`, and `WDIO_SPECS=./tensorboard-example.spec.js`; headful runs expose Selenium's noVNC desktop on port `7900`.


## Feature flags

KFP frontend supports feature flags for development:

- Configure in `src/features.ts`
- Access via `http://localhost:3000/#/frontend_features`
- Manage locally: `localStorage.setItem('flags', "")`

## Common development tasks

- **Add new API**: Update swagger specs, run `npm run apis`
- **Update proto definitions**: Modify protos, run respective build commands
- **Add new component**: Create in `atoms/` or `components/`, add tests and stories
- **Debug server**: Use `npm run start:proxy-and-server-inspect`
- **Bundle analysis**: `npm run analyze-bundle`

## Code style and formatting

- **Prettier** config in `.prettierrc.yaml`:
  - Single quotes, trailing commas, 100 char line width
  - Format: `npm run format`
  - Check: `npm run format:check`
- **ESLint** extends `react-app` with custom rules in `.eslintrc.yaml`
- **Auto-format on save**: Configure your IDE with the Prettier extension

## Troubleshooting

- **Port conflicts**: Frontend uses 3000 (React), 3001 (Node server), 3002 (API proxy)
- **Node version issues**: Ensure you're using the version in `.nvmrc`
- **API generation failures**: Check that swagger-codegen-cli.jar is in PATH
- **Proto generation**: Requires `protoc` and `protoc-gen-grpc-web` in PATH
- **Mock backend**: Limited API support; use real cluster for full testing
