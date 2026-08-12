# Frontend

The frontend is React 19, TypeScript, MUI, TanStack Query, Vitest, Prettier, and ESLint. Use the Node version in `frontend/.nvmrc` and the npm version pinned by `frontend/package.json`.

```bash
cd frontend
npm ci
npm run build
npm run test:ui
npm run lint
npm run typecheck
npm run format:check
```

Use `npm run mock:api` with `npm start` for mock development, or `npm run start:proxy-and-server` with a cluster. `npm run apis:all` requires Docker.

## React effects

- Use `useEffect` only for external synchronization such as browser APIs, timers, subscriptions, storage, imperative DOM APIs, or external async systems.
- Derive UI state during rendering; put user actions and mutation outcomes in event handlers or mutation callbacks.
- Do not let refreshes reset user-controlled state without an explicit product requirement.
- Do not suppress `react-hooks/exhaustive-deps` unless the invariant is documented and tested.
- For every changed effect, classify it as external sync, derived state, event-driven, state reset, or effect chain. Only external sync is normally acceptable.

When effect behavior changes, add or run a focused regression test for duplicate mutation success, refresh-preserved selections, retry recovery, or unexpected mount-time callbacks.

Prettier uses single quotes, trailing commas, and a 100-character line width.
