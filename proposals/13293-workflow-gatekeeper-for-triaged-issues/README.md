# KEP-13293: Gatekeep CI Workflows for Approved Issues

## Summary
With the mainstream introduction of coding agents within the last year, Kubeflow Pipelines (KFP), like many other open-source projects, has seen a large increase in AI-generated drive-by PRs. 
A drive-by PR is a pull request opened by a new contributor that generally does not resolve a relevant raised issue and is often never followed up on.
When a new contributor raises a PR without a groomed issue or KEP, the result is often a heavily opinionated implementation/user experience that makes the PR review process harder.
The consequence of these PRs are overburdened maintainers, an overloaded CI and more limited community resources for new contributors with a genuine interest. 
This KEP proposes a two-part approach to PR validation: a workflow for marking issues ready to be addressed, and a workflow gatekeeping CI execution on a valid linked issue.

## Definitions
- **New contributor**: a contributor to KFP that is not a Kubeflow member.

## Motivation
The three primary consequences of (predominantly) AI-generated drive-by PRs are maintainer fatigue, CI overload and a resulting lack of support for genuine new contributors. 
These consequences threaten the health and longevity of KFP and drive the following goals. 

### Goals
1. Reduce the KFP maintainer burden by closing unfocused PRs that take away focus from KFP community priorities.
2. Streamline onboarding fatigue for new contributors by requiring them to work on issues labeled `/good-first-issue`

### Non-Goals
1. This proposal does not enhance support for the PR review process. It aims to reduce extraneous PRs.

## Proposal
This KEP proposes a two-part approach: a GitHub Action workflow to label good first issues, and a GitHub Action workflow to validate that PRs raised by new contributors link an issue with this label.

### I. Introduce '/good-first-issue' label for beginner-friendly issues
@modichika proposes the following workflow to validate beginner-friendly issues in the following PR: https://github.com/kubeflow/pipelines/pull/13372
1. Utilize `gpt-4o-mini` with GH Models API to analyze issue for scope, context, guidance and complexity and decide if the issue is a good candidate for first-time contributors.
2. If the issue is a good candidate, it is labeled `/good-first-issue`

### II. PRs from new contributors must link valid issue to run CI workflows
Add a GitHub Action workflow triggered on PRs from new contributors:
1. Parse the PR body for a linked issue.
2. Validate that the linked issue is both an existing KFP issue and also labeled `/good-first-issue`.
3. If the linked issue does not meet the above criteria, the PR is auto-closed with the following message:
```text
Closing this PR automatically. Link an issue labeled '/good-first-issue' in the PR body and then reopen. See CONTRIBUTING.md for more details.
```
4. If the linked issue does meet the above criteria, the PR is labeled `/ok-to-test` and CI continues.

### III. Documentation
Documentation must be added to [CONTRIBUTING.md](https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md) to explain `/good-first-issue` label criteria.

### IV. Managing Pre-existing PRs and Issues
**Note that only PRs from non-KF members will be evaluated. PRs from KF members will not be affected.**
After these two proposed workflows are merged into master, the following steps are required to handle pre-existing work:
1. All current PRs from new contributors that have been inactive for 4+ weeks will be closed with an explanatory message.
2. Current PRs from new contributors that have been inactive for less than 4 weeks will not be closed, but when the contributor pushes new commits, the PR will be re-evaluated for an issue.
3. All current issues will be evaluated for `/good-first-issue` label.

### Risks and Mitigations
#### Large CI Change
This KEP design introduces a large CI change. 
All existing workflows are modified to depend on the gatekeeper workflow. 
Although this change can be tested on singular PRs, it will not be possible to performance-test this CI change over any length of time before merging changes into master. 
This risk is mitigated by the test cases below, tested on my fork.

#### Discouraging New Contributors
Although one of the goals of this KEP is to free up KFP community resources to be able to support genuine new contributors, one potential unintended consequence is discouraging new contributors with a more lengthy contribution process.
This risk is mitigated by the process documentation added to CONTRIBUTING.md, which is also linked in the event an issue is auto-closed.
For this reason, the gatekeeper workflow will not be executed on PRs with `(docs)` or `(chore)` in the title, allowing new contributors to bypass the gatekeeper workflow for these smaller changes.

#### Slowing Down Work Under Current Review
The linked issue requirement will slow down PRs currently under review, as the author will need to create an issue which must then be labeled `/good-first-issue`.
However, this label can also be manually applied by maintainers in the case that the label fails, but the PR is a good fit for the contributor.

## Drawbacks
While the overall goal of this KEP is to reduce maintainer/reviewer fatigue, the design does require more maintainer/reviewer involvement at the issue level, given that issues must be labeled `/good-first-issue`.


## Alternatives
### Remove AI triage
If the KFP community is not in favor of using `gpt-4o-mini`, either a different AI model can be proposed, or the workflow can be modified to tag
maintainers/approvers for manual approval. However, this option would require more maintainer work, which this KEP is trying to reduce.


## Adapting this proposal to other projects
This design is easily adaptable to other repositories and projects. 
The issue triage workflow can be modified to rely either more or less heavily on Copilot AI triage. 
The gatekeeper workflow can be easily modified to include more or less stringent requirements.


## Implementation History
Kubeflow Pipelines has not introduced a workflow in its history thus far to limit PRs to approved issues.
This idea was proposed in the KFP Community Meeting on 2026-04-10.