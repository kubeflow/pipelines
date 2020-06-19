# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution;
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Coding style

The Python part of the project will follow [Google Python style guide](http://google.github.io/styleguide/pyguide.html). We provide a [yapf](https://github.com/google/yapf) configuration file to help contributors auto-format their code to adopt the Google Python style. Also, it is encouraged to lint python docstrings by [docformatter](https://github.com/myint/docformatter).

The frontend part of the project uses [prettier](https://prettier.io/) for formatting, read [frontend/README.md#code-style](frontend/README.md#code-style) for more details.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## Pull Request Title Convention

We enforce a PR title convention to quickly indicate type and scope of a PR.
We also parse it programmatically for changelog generation.

PR titles should
* be a user facing description of the change
* be [conventional](https://www.conventionalcommits.org/en/v1.0.0/)
* append fixed issue(s) at the end (if any)

Examples:
* `"fix(ui): fixes empty page. Fixes #1234"`
* `"feat(backend): configurable service account. Fixes #1234, fixes #1235"`
* `"chore: refactor some files"`
* `"test: fix CI failure. Part of #1234"` // use "part of" when a PR is an work item of some issue, but shouldn't close the issue when merged.

A quick summary of the conventions we care about:
### PR Title Structure
```
<type>[optional scope]: <description> [Fixes #<issue-number>]
```
### PR Type
Type can be one of
* feat: A new feature
* fix: A bug fix (however a bug fix to test infra is not user facing, so it should be test type instead)
* docs: Documentation only changes
* chore: Anything else that do not need to be user facing
* test: Adding missing tests or correcting existing tests
* refactor: A code change that neither fixes a bug nor adds a feature
* perf: A code change that improves performance

Note, only feature and fix type PRs will be included in CHANGELOG, because they are
user facing.

If you think the PR contains multiple types, you can choose the major one or
split the PR to focused sub-PRs.

If you are not sure which type your PR is and it does not have user impact,
use `chore` as the fallback.

### PR Scope
Scope is optional, it can be one of
* frontend - UI or UI server related, folder `frontend`, `frontend/server`
* backend - Backend, folder `backend`
* sdk - `kfp` python package, folder `sdk`
* sdk/client - `kfp-server-api` python package, folder `backend/api/python_http_client`
* components - Pipeline components, folder `components`
* deployment - Kustomize or gcp marketplace manifests, folder `manifests`
* metadata - Related to machine learning metadata (MLMD), folder `backend/metadata_writer`
* cache - Caching, folder `backend/src/cache`
* swf - Scheduled workflow, folder `backend/src/crd/controller/scheduledworkflow`
* viewer - Tensorboard viewer, folder `backend/src/crd/controller/viewer`

If you think the PR is related to multiple scopes, you can choose the major one or
split the PR to focused sub-PRs. Note, suggest splitting a huge PR because different scopes
usually have different reviewers, it helps getting through the review process faster too.

If you are not sure, or the PR doesn't fit into above scopes. You can either
omit the scope because it's optional, or propose an additional scope here.

## Community Guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google.com/conduct/).
