# Language-Specific Review Checklists

Load when reviewing code changes. Pick the relevant language section.

---

## Python (SDK, kfp-kubernetes)

- Type hints on all function signatures
- Docstrings on all public APIs/properties (feed Sphinx docs); include usage examples for new DSL options
- No SDK-only imports in executor/runtime paths (guarded by `_KFP_RUNTIME`)
- YAPF, isort (`--profile google`), pycln, docformatter compliance
- Return type annotations must cover ALL runtime types (e.g., `Optional[Union[str, Dict]]` not just `Optional[str]`)
- Cross-language structs: add `# Must stay aligned with Go struct in ...` comments
- Protobuf field presence: use `msg.HasField('field')`, not `msg.field is None`
- `Iterable[str]` params: guard against bare `str` input (iterates as individual chars)
- No bare `Exception` catches, mutable defaults, or missing `__init__.py` exports

## Go (backend)

### Error handling and control flow
- Every `err :=` must be checked or returned. Swallowed errors cause silent failures.
- Wrap errors: `fmt.Errorf("context: %w", err)` or `util.Wrap()`
- `return` inside loops: verify it should be `continue`, not premature exit
- Loops assigning to a variable: verify intermediate values aren't discarded
- No variable shadowing of function arguments
- No unnecessary `else` after `return`
- Re-read existing comments in modified functions -- they may be stale after refactoring

### Concurrency
- No nested goroutines (`go func() { go func() { ... } }()`)
- Reuse K8s clients from struct fields, not `kubernetes.NewForConfig()` per call
- Fire-and-forget goroutines: require `context.WithTimeout` + error logging
- Functions always returning `nil` after launching goroutine: flag (caller can't handle errors)

### Code quality
- `golangci-lint run` passes
- Context propagation (`context.Context` as first parameter)
- Resource cleanup (`defer` close for readers/connections)
- Ginkgo conventions for integration/E2E tests
- Extract repeated `fmt.Sprintf` patterns (3+ times) into named functions
- No deprecated protobuf fields in new code or tests
- Descriptive variable names, especially in security-critical code (`securityContext` not `sc`)
- camelCase for locals (`parentDAGID` not `parent_dag_id`)

## TypeScript (frontend)

- Prettier, ESLint, TypeScript strict mode, React peer compatibility
- Vitest tests for new components
- No direct modification of generated API clients
- No inline styles (use Tailwind/Material-UI), no direct DOM manipulation

## Protocol Buffers

- Field numbers never reused; deprecated fields marked `[deprecated = true]`
- New fields additive (backward compatible); use `optional` for presence detection
- API comments precise: "will" not "may" when behavior is deterministic
- "At most one" semantics enforced at compile/validation time
- Full proto change checklist in [impact-analysis.md](impact-analysis.md#proto-change-checklist)
