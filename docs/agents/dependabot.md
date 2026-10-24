# Dependency updates

Scheduled Dependabot updates target the root and SDK Python workspace through uv.
Independent Python projects retain normal pip version and security updates.
This includes the optional `local/` native-wheel proof, which is not a root uv
workspace member.
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

## Frontend dependency security constraints

The Kubernetes client override patches its nested `js-yaml` independently of
the server's direct YAML dependency. The `gaxios` override keeps `uuid` on
patched major 11, which retains CommonJS support; major 12 changes to ESM.
The server dependency regression test exercises gaxios's multipart UUID path.

The frontend selector-parser override covers nested Tailwind/PostCSS copies.
Tailwind 3.0.11 generated identical CSS with parser 7.1.7 during verification;
keep validating the CSS build when changing this cross-major override.

### Remaining braces advisory

[GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)
affects `braces` through 3.0.3, with no patched release listed on 2026-10-23.
Frontend and server retain `http-proxy-middleware → micromatch → braces`;
frontend build tooling also retains it through Tailwind's glob/file watchers.
These build tools consume repository-controlled patterns, not request inputs.

Production proxies omit `pathFilter` or use an artifact namespace callback.
Request URLs are not glob patterns in these configurations. The runtime
configuration test `frontend/server/proxy-path-filter.test.ts` enables optional
proxies and checks every registration for absent or function-valued filters.
Reassess reachability if a proxy adds string/array filters, callbacks start
parsing glob patterns, or the matching implementation changes.

This is a reachability assessment, not a patched dependency. The two lockfile
alerts remain pending an upstream fix or removal of their dependency chains.
The integration test project instead removes `braces` through coordinated
WebdriverIO 10/Mocha 12 upgrades.
