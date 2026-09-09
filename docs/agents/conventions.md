# Conventions

## Reuse and design

- Search for and reuse existing helpers before adding code. Refactor an existing implementation if that avoids duplication.
- Use descriptive names; avoid unexplained abbreviations and single-letter names.
- Keep `ResourceManager` focused on run/job persistence and lifecycle coordination.
- Keep execution-engine behavior behind compiler or execution abstractions. Shared code must remain engine-neutral; do not downcast to `*util.Workflow` in shared layers.
- Put reusable interfaces in neutral packages and use natural domain types. Preserve documented ownership and field-wise override behavior.

## Tests

- Add a unit test for non-trivial functions, methods, and exported APIs. Add coverage when changed behavior needs it.
- Run the relevant tests before submitting. Document unrelated pre-existing failures in the PR description.

## Comments and documentation

- Comment only non-obvious constraints, invariants, workarounds, or surprising behavior.
- Keep comments short. Do not narrate implementation, current-PR history, or obvious test setup.
- Error messages must state the problem and corrective action.
- Write concise GoDoc for exported Go APIs. Python SDK public docstrings are user-facing Sphinx documentation.

## Commits

- Sign commits with `git commit -s`.
- Do not add AI agents as commit co-authors.
- Follow [`CONTRIBUTING.md`](../../CONTRIBUTING.md) for DCO and PR conventions.
