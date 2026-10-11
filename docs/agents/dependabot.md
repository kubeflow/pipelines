# Dependency updates

Scheduled Dependabot updates target the root and SDK Python workspace through uv. Independent
Python projects remain on pip. Root and SDK `requirements.txt` files are
generated exports, not independent dependency manifests.

After a uv dependency update, refresh both exports from the proposed lockfile
using uv 0.10.3, matching CI:

```bash
bash .github/resources/scripts/export_python_requirements.sh
git diff -- requirements.txt sdk/python/requirements.txt
```

Review and commit changed exports to the dependency PR, with a DCO sign-off,
then let normal CI run on the new commit. The exporter uses the frozen lockfile;
it does not resolve newer versions. A source update that changes the exports
continues to fail the requirements consistency check until they are refreshed.
`exclude-paths` only excludes these exports from version updates. Dependabot
security updates can still edit exports directly; see
[dependabot-core#14408](https://github.com/dependabot/dependabot-core/issues/14408)
and [dependabot-core#13912](https://github.com/dependabot/dependabot-core/issues/13912).
The required requirements consistency check is the enforcement boundary: it
regenerates both files from `uv.lock` and rejects any tracked difference,
including an export-only security update. For those PRs, update the vulnerable
dependency in the workspace source/lock first, then regenerate both exports.
Do not restore an old export merely to make CI green; verify the locked version
contains the security fix. Configuration alone does not guarantee exclusive
file ownership. No additional App, credentials, or automated writer is required.

`hack/update-all-requirements.sh` locks and exports the workspace together.
The requirements check and release tooling use the same exporter. Release
tooling retains its existing export commands for older source checkouts.

React, ReactDOM, and their typings are grouped together for both version and
security updates. Grouping does not waive peer compatibility, source review,
or the normal CI and merge requirements.
