# Dependencies

Python constraints and `uv.lock` are the workspace source of truth. Update the
lock with source constraints and validate using frozen installs on Python 3.11 and
3.13. The Docker Python package is retained for component image building; it is
not a local execution engine. The Kubernetes client supplies SDK configuration
model types. REST-client-only and legacy cloud-registry dependencies are removed.

Frontend dependencies and `package-lock.json` cover only the native application,
its shared widgets and tests. There is no nested server or mock-backend install.
Keep React and its typings, Vitest packages, and TypeScript ESLint packages grouped.
Rebuild the offline viewer assets after dependency changes.

Dependabot covers uv, the frontend npm package, GitHub workflow-script requirements,
the native Dockerfiles, and GitHub Actions. Backend/Go/release-branch entries are
retired. The Argo pin lives in `third_party/argo/VERSION`; changing it requires
schema/resource and real-cluster verification.
