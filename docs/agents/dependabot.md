# Dependency updates

Scheduled Dependabot updates target the root and SDK Python workspace through uv.
Independent Python projects retain normal pip version and security updates.
The workspace tracks `pyproject.toml` manifests and `uv.lock`; root and SDK
`requirements.txt` exports are no longer committed. Dependency fixes must update
the source constraints and lockfile, including the vulnerable locked version.
There are no duplicate workspace pip exports for Dependabot to update separately.

CI checks lock consistency, installs the frozen workspace, builds SDK distributions,
and installs the wheel in a clean Python 3.11 pip environment. Optional pip consumers
can generate dependency-only artifacts with uv 0.13.4:

```bash
bash .github/resources/scripts/export_python_requirements.sh /tmp/kfp-python-exports
```

The explicit output directory receives `workspace-requirements.txt` and
`kfp-requirements.txt`. These frozen, unhashed exports omit local workspace projects,
so they can be used outside the source checkout. Install the built SDK wheel separately.
Do not commit exports. `hack/update-all-requirements.sh` updates the workspace lock;
modern release preparation builds distributions without recreating exports. Older
release source checkouts that contain tracked exports retain their original behavior.

React, ReactDOM, and their typings are grouped together for both version and
security updates. Grouping does not waive peer compatibility, source review,
or the normal CI and merge requirements.

Vitest and `@vitest/*` packages are grouped for version and security updates:
coverage providers require the exact runner release. Keep these groups before
the generic npm group and update the frontend manifest and lock together.
Security grouping does not guarantee a non-vulnerable peer is updated; verify
the resolved peer versions and the coverage suite before merging.
The locked-version guard covers both `frontend` and `frontend/server`; each
package installs its own runner and coverage provider and must stay aligned.
