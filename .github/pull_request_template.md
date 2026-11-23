## Problem

<!-- Explain the underlying problem and why it matters for a reader with no prior
     context. Omit this section for trivial, self-explanatory changes. -->


**Description of your changes:**
<!-- Explain how the change addresses the problem. -->


## Before and after

<!-- For bug fixes, include a picture, diagram, or animation explaining the problem
     and fix, clearly labeled Before and After. Use screenshots or a recording for
     visible changes, or a diagram such as Mermaid for logic/data-flow changes.
     For other changes, omit this section if it adds no useful context. -->


## Verification

<!-- For bug fixes, complete the reproducer and regression evidence below.
     Keep the automated regression test in the PR and verify that it fails for the
     expected reason before the fix and passes after it. Link passing CI when
     available; if blocked, state what remains unverified.
     For other changes, describe relevant checks and remove inapplicable prompts. -->

**Reproducer:**
<!-- Minimal repeatable steps or exact command, required inputs, and expected
     versus actual behavior. -->

**Regression test and command:**
<!-- Identify the retained automated test and the exact command used to run it. -->

**Before-fix result:**

**After-fix result / CI:**


**Release notes / user impact:**
<!-- Describe affected users/components and the behavior before and after this change.
     Include breaking changes, deprecations, and required migration steps or links.
     Write "None" for changes with no user impact. See CONTRIBUTING.md#release-note-policy.
     For breaking changes, also use ! in the PR title or retain a BREAKING CHANGE:
     footer in the final squash commit; this field alone does not mark the commit. -->


**Checklist:**

- [ ] The description follows the [PR description rules](https://github.com/kubeflow/pipelines/blob/master/AGENTS.md#pull-request-descriptions) and, for bug fixes, the [evidence requirements](https://github.com/kubeflow/pipelines/blob/master/AGENTS.md#bug-fixes-and-pr-evidence).
- [ ] You have [signed off your commits](https://www.kubeflow.org/docs/about/contributing/#sign-off-your-commits)
- [ ] The title for your pull request (PR) should follow our title convention. [Learn more about the pull request title convention used in this repository](https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md#pull-request-title-convention).
<!--
   PR titles examples:
    * `fix(frontend): fixes empty page. Fixes #1234`
       Use `fix` to indicate that this PR fixes a bug.
    * `feat(backend): configurable service account. Fixes #1234, fixes #1235`
       Use `feat` to indicate that this PR adds a new feature.
    * `chore: set up changelog generation tools`
       Use `chore` to indicate that this PR makes some changes that users don't need to know.
    * `test: fix CI failure. Part of #1234`
        Use `part of` to indicate that a PR is working on an issue, but shouldn't close the issue when merged.
-->
