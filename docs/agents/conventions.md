# Conventions

- Reuse existing helpers and keep modules at native engine boundaries.
- Prefer deletion to compatibility wrappers for retired backend behavior.
- Preserve validation and error handling when moving code. Unsupported constructs
  must report actionable errors rather than silently dropping configuration.
- Add tests for nontrivial behavior and run the relevant suites before committing.
- Keep comments concise and explain constraints rather than narrating the code.
- Update schema sources and regenerate bindings; never hand-edit generated code.
- Use signed-off commits and the repository's conventional commit style.
