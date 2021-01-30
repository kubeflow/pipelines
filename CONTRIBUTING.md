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
## Unit Testing Best Practices

* Testing via Public APIs

### Golang
* Put your tests in a different package: Moving your test code out of the package allows you to write tests as though you were a real user of the package. You cannot fiddle around with the internals,
instead you focus on the exposed interface and are always thinking about any noise that you might be adding to your API. Usually the test code will be put under the same folder
but with a package suffix of `_test`. https://golang.org/src/go/ast/example_test.go (example)
* Internal tests go in a different file: If you do need to unit test some internals, create another file with `_internal_test.go`
as the suffix.
* Write table driven tests: https://github.com/golang/go/wiki/TableDrivenTests (example)

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## Pull Request Title Convention

We enforce a pull request (PR) title convention to quickly indicate the type and scope of a PR.
PR titles become commit messages when PRs are merged. We also parse PR titles to generate the changelog.

PR titles should:
* Provide a user-friendly description of the change.
* Follow the [Conventional Commits specification](https://www.conventionalcommits.org/en/v1.0.0/).
* Specifies issue(s) fixed, or worked on at the end of the title.

Examples:
* `fix(ui): fixes empty page. Fixes #1234`
* `feat(backend): configurable service account. Fixes #1234, fixes #1235`
* `chore: refactor some files`
* `test: fix CI failure. Part of #1234`

The following sections describe the details of the PR title convention.

### PR Title Structure
PR titles should use the following structure.
```
<type>[optional scope]: <description>[ Fixes #<issue-number>]
```

Replace the following:

*  **`<type>`**: The PR type describes the reason for the change, such as `fix` to indicate that the PR fixes a bug. More information about PR types is available in the next section. 
*  **`[optional scope]`**: (Optional.) The PR scope describes the part of Kubeflow Pipelines that this PR changes, such as `frontend` to indicate that the change affects the user interface. Choose a scope according to [PR Scope section](#pr-scope).
*  **`<description>`**: A user friendly description of this change.
*  **`[ Fixes #<issues-number>]`**: (Optional.) Specifies the issues fixed by this PR.

### PR Type
Type can be one of the following:
* **feat**: A new feature.
* **fix**: A bug fix. However, a PR that fixes test infrastructure is not user facing, so it should use the test type instead.
* **docs**: Documentation changes.
* **chore**: Anything else that does not need to be user facing.
* **test**: Adding or updating tests only. Please note, **feat** and **fix** PRs should have related tests too.
* **refactor**: A code change that neither fixes a bug nor adds a feature.
* **perf**: A code change that improves performance.

Note, only feature, fix and perf type PRs will be included in CHANGELOG, because they are user facing.

If you think the PR contains multiple types, you can choose the major one or
split the PR to focused sub-PRs.

If you are not sure which type your PR is and it does not have user impact,
use `chore` as the fallback.

### PR Scope
Scope is optional, it can be one of the following:
* **frontend**: user interface or frontend server related, folder `frontend`, `frontend/server`
* **backend**: Backend, folder `backend`
* **sdk**: `kfp` python package, folder `sdk`
* **sdk/client**: `kfp-server-api` python package, folder `backend/api/python_http_client`
* **components**: Pipeline components, folder `components`
* **deployment**: Kustomize or gcp marketplace manifests, folder `manifests`
* **metadata**: Related to machine learning metadata (MLMD), folder `backend/metadata_writer`
* **cache**: Caching, folder `backend/src/cache`
* **swf**: Scheduled workflow, folder `backend/src/crd/controller/scheduledworkflow`
* **viewer**: Tensorboard viewer, folder `backend/src/crd/controller/viewer`

If you think the PR is related to multiple scopes, you can choose the major one or
split the PR to focused sub-PRs. Note, splitting large PRs that affect multiple
scopes can help make it easier to get your PR reviewed, since different scopes
usually have different reviewers.

If you are not sure, or the PR doesn't fit into above scopes. You can either
omit the scope because it's optional, or propose an additional scope here.

## Community Guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google.com/conduct/).
