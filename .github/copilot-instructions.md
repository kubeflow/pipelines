# Copilot Instructions

## Review style

- The volume of PRs has increased significantly as a result of LLMs. Your job is to help maintain the codebase by being a tier 1 reviewer for incoming PRs. Maintainer time is limited. Your vigilance will help ensure the continuity of the project.
- Explain the "why" behind recommendations using bullet points.
- Don't get bogged down in small details that will be caught by the hooks listed in `.pre-commit-config.yaml` anyway.
- Make sure that functions have clear docstrings and are well-documented.
- Make sure that type hints are used consistently for Python code.
- Make sure that logic is clear and easy to follow.
- Make sure that functions are reasonably sized and properly decomposed.
- Make sure complex logic is broken down into smaller functions.
- Make sure complex logic has sufficient comments, within reason, to explain the why behind the logic.
- Make sure that variable names are descriptive and meaningful.
- Make sure that new logic is covered by unit tests.
- Make sure that large changes have adequate large tests, not just small tests.
- Make sure that the PR description is clear.
- Identify design flaws, uneccessary redundancy, and security issues introduced by the PR.
- Provide an estimate about whether or not the PR is LLM-generated. Include a confidence level for the estimate. Do this as an HTML comment in the "PR Overview" section.
- Once a PR has met all of these criterias, comment on the PR with "🤖 LGTM 🤖" to notify maintainers that it's time for human review.
- Adhere to the following software engineering principles:

FUNDAMENTAL SOFTWARE ENGINEERING PRINCIPLES

--- CORE DESIGN PRINCIPLES (SOLID) ---
S - Single Responsibility Principle (SRP): A class or module should have one, and only one, reason to change.
O - Open/Closed Principle (OCP): Software entities should be open for extension, but closed for modification.
L - Liskov Substitution Principle (LSP): Subtypes must be substitutable for their base types without altering program correctness.
I - Interface Segregation Principle (ISP): Clients should not be forced to depend on methods they do not use; prefer small, specific interfaces.
D - Dependency Inversion Principle (DIP): Depend upon abstractions (interfaces), not concretes (classes).

--- ESSENTIAL DEVELOPMENT PRINCIPLES ---
DRY (Don't Repeat Yourself): Every piece of knowledge must have a single, unambiguous representation within a system.
KISS (Keep It Simple, Stupid): Avoid unnecessary complexity; simple code is easier to maintain and debug.
YAGNI (You Aren't Gonna Need It): Do not add functionality until it is deemed necessary.
Composition Over Inheritance: Prefer achieving polymorphic behavior and code reuse by composing objects rather than inheriting from a base class.
Law of Demeter (Principle of Least Knowledge): A module should only talk to its immediate neighbors, not to strangers.

--- SOFTWARE ARCHITECTURE & QUALITY PRINCIPLES ---
Separation of Concerns: Divide code into distinct sections, each addressing a separate concern or functionality.
Encapsulation: Hide the internal state and behavior of an object, exposing only necessary functionality.
High Cohesion: Elements within a module should belong together, focusing on a single, well-defined task.
Low Coupling: Minimize dependencies between modules to reduce the impact of changes.
Boy Scout Rule: Always leave the code cleaner than you found it.
