# CI and Workflows

GitHub Actions workflows are in `.github/workflows/`; reusable composite actions are in `.github/actions/`.

- Update this guide when changing workflows, CI matrices, commands, generated outputs, or common failure handling.
- CI covers supported Kubernetes and Argo versions, database and Kubernetes pipeline stores, proxy/cache variants, and GPU scheduling. Preserve the relevant matrix coverage when changing a lane.
- Use `.github/actions/setup-python-pip-cache` for pip caching. Give each dependency set a distinct `cache-scope` and hash every installed requirements file; do not use `setup-python`'s built-in pip cache.
- `validate-generated-files.yml` validates backend-generated outputs. `frontend.yml` runs `npm run apis:all` and rejects stale frontend clients.
- Changes to `frontend/server/handlers/artifacts.ts` trigger `e2e-test.yml` so the multi-user artifact-proxy and SeaweedFS namespace-isolation checks run before artifact authorization changes merge.
- For workflow-only changes, verify referenced working directories, Docker contexts/files, scripts, and local action paths exist.

## Common CI failures

- Registry pull failures for Kind, BuildKit, Python, or Alpine images are usually transient; retry before changing code.
- A Kind checksum mismatch after cache restore means no tests or deployment ran; retry the job.
- SeaweedFS `PutObject` timeouts are artifact-store instability; retry rather than weakening assertions or increasing pipeline timeouts.
- For proxy failures, inspect the `tinyproxy` namespace pods, events, services, endpoints, and endpoint slices.
