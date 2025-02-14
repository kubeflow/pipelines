# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Legal

Kubeflow uses Developer Certificate of Origin ([DCO](https://github.com/apps/dco/)).

Please see https://github.com/kubeflow/community/tree/master/dco-signoff-hook#signing-off-commits to learn how to sign off your commits.

## Contribution Guidelines

To propose a new feature or a change that alters some existing user experience
or creates a new user experience, follow these steps:

### Step 1: Establish Context

Search on KFP GitHub issues list to see if the same or similar proposal has been
made in the past.  The historical context can help you draft a better
proposal. Sometimes you will find a very similar proposal was already presented,
discussed thoroughly, and that it is either awaiting contribution (in active
development) or was rejected (often due to timing or conflicting scope with
other plans). To avoid confusion and conflicts, where possible, please
contribute to existing issues before creating new ones.

### Step 2: Create Feature Request

Create a new issue using the “Feature Request” template if no existing issue is
found. Fill in answers to the template questions. To avoid delays, provide as
much information as needed for initial review. Keep in mind that new features
should comply with backward-compatibility and platform-portability requirements.

### Step 3: Initial Team Triage

Wait for a member from the Kubeflow Pipelines team (under
orgs/kubeflow/teams/pipelines/ in
[org.yaml](https://raw.githubusercontent.com/kubeflow/internal-acls/master/github-orgs/kubeflow/org.yaml))
to comment on the issue. The team aims for triaging new issues on a weekly
basis, but cannot at this time provide a guarantee on when your issue will be
reviewed. The team will work with you to determine if your change is trivial and
can proceed or whether it is nontrivial and needs a more detailed design
document and review.

### Step 4: Design Review

If the team agreed with the overall proposal, you would be asked to write a
design documentation, explaining why you want to make a change, what changes are
you proposing, and how do you plan to implement it. The design review process
would be required by default unless the team agreed that the change is too
trivial. It is recommended that you use this [Google doc template](https://docs.google.com/document/d/1VrfuMo8ZeMmV75a-rUq9SO-E6KotBodAf-P0WZeFDZA/edit?usp=sharing&resourcekey=0-BklOgu8ivhdLCplZuPDZZg) (You need to join [kubeflow-discuss](https://groups.google.com/g/kubeflow-discuss) google group to get access)
for your design, and share it with kubeflow-discuss@googlegroups.com for
commenting. After sharing the design documentation, you could optionally join a
session of the bi-weekly Kubeflow Pipelines community meetings
[[agenda](http://bit.ly/kfp-meeting-notes)] to present or further discuss your
proposal. A proposal may still get rejected at this stage if it comes with
unresolved drawbacks or if it does not align with the long term plans for the
project.

### Step 5: Implementation

After you get formal approval from a Kubeflow Pipelines team member, you can
implement your design and send a pull request. Make sure existing tests are all
passing and new tests are added when applicable. Remember to link to the feature
request issue to help reviewers catch up on the context.

## Project Structure

Kubeflow Pipelines consists of multiple components. Before you begin, learn how to [build the Kubeflow Pipelines component container images](./developer_guide.md##build-image). To get started, see the development guides:

* [Frontend development guide](./frontend/README.md)
* [Backend development guide](./backend/README.md)

## Coding style

### SDK
See the [SDK-specific Contribution Guidelines](sdk/CONTRIBUTING.md) for contributing to the `kfp` SDK.

### Frontend

The frontend part of the project uses [prettier](https://prettier.io/) for formatting, read [frontend/README.md#code-style](frontend/README.md#code-style) for more details.

### Backend
Use [gofmt](https://pkg.go.dev/cmd/gofmt) package to format your .go source files. Read [backend/README.md#code-style](backend/README.md#code-style) for more details. 

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

* **`<type>`**: The PR type describes the reason for the change, such as `fix` to indicate that the PR fixes a bug. More information about PR types is available in the next section.
* **`[optional scope]`**: (Optional.) The PR scope describes the part of Kubeflow Pipelines that this PR changes, such as `frontend` to indicate that the change affects the user interface. Choose a scope according to [PR Scope section](#pr-scope).
* **`<description>`**: A user friendly description of this change.
* **`[ Fixes #<issues-number>]`**: (Optional.) Specifies the issues fixed by this PR.

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
