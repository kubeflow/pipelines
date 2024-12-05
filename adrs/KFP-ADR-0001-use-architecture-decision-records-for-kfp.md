# Use Architecture Decision Records for Kubeflow Pipelines

|                |                          |
|----------------|--------------------------|
| Date           | 2024-04-24               |
| Scope          | Kubeflow Pipelines       |
| Status         | Accepted                 |
| Authors        | [Humair Khan](@HumairAK) |
| Supersedes     | N/A                      |
| Superseded by: | N/A                      |
| Issues         |                          |
| Other docs:    | none                     |

# Kubeflow Pipelines Architecture Decision Records

"Documenting architectural decisions helps a project succeed by helping current and future contributors understand the reasons for doing things a certain way." [1]

## What is an ADR?

An architecture decision record is a short text file in a Markdown format. Each record describes a set of forces and a single decision in response to those forces. [2]

An ADR is not a technical design, a team-level internal procedure, or a roadmap. An ADR does not replace detailed technical design documents or good commit messages.

## Why

Using an Architecture Decision Record (ADR) offers many benefits, particularly in managing the complexity and longevity of software projects.

Some examples include: 

1. ADRs capture the why behind critical architectural choices, not just the what.
   * This helps current and future team members understand the reasoning behind decisions, particularly when the rationale is no longer obvious.
2. Improve Communication and Collaboration.
   * They serve as a single source of truth for architectural decisions.
   * By documenting options and their trade-offs, ADRs encourage structured decision-making and transparency.
3. Enable Traceability
   * ADRs create a decision history that allows teams to trace architectural choices back to their original context, assumptions, and goals

See references below for more exhaustive lists on how ADRs can be a net benefit, especially to an open source project, 
where transparency in decision making is key.

## Goals

* Capture the Why Behind Decisions
* Foster Clear Communication
* Enable Decision Traceability
* Encourage Thoughtful, Deliberate Decisions
* Preserve Institutional Knowledge

## Non-Goals

* Not a substitute for technical or user documentation
* Not a substitute or replacement for meaningful commit messages 

## How

We will keep each ADR in a short text file in Markdown format.

We will keep ADRs in this repository, https://github.com/kubeflow/pipelines, under the `./adrs` folder.

ADRs will be numbered sequentially and monotonically. Numbers will not be reused.

If a decision is reversed, we will keep the old one around, but mark it as superseded. (It's still relevant to know that it was the decision, but is no longer the decision.)

We will use a format with just a few parts, so each document is easy to digest.

## Alternatives

**Current Approach**

One alternative is to not do ADRs, and stick to the current approach of doing google docs or similar and presenting it in KFP calls. 
The pros for this approach is that it's relatively low overhead and simple. Communicating changes/editing google docs is also immensely easier than on a markdown PR. 

The cons are plentiful however:
* No way to preserve these docs effectively
* Does not live near the codebase
* Difficult to enforce immutability 
* No way to formalize an "approval" process (i.e. something akin to a PR "merge")
* Doc owners are not maintainers, and access can be revoked at any time
* Hard to keep track off google documents

## Reviews

| Reviewed by | Date | Notes |   
|-------------|------|-------|

## References

* https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions
* https://adr.github.io/
* https://docs.aws.amazon.com/prescriptive-guidance/latest/architectural-decision-records/adr-process.html
* https://github.com/joelparkerhenderson/architecture-decision-record?tab=readme-ov-file#what-is-an-architecture-decision-record

## Citations

* [1] Heiko W. Rupp, https://www.redhat.com/architect/architecture-decision-records
* [2] Michael Nygard, https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions
