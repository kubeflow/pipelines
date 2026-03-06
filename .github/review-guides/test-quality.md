# Test Quality Review Guide

Load when reviewing PRs that add/modify tests, or should include tests.

---

## Coverage requirements

- Every new public function: happy path + edge cases + error paths
- New store/SQL methods: unit tests with mock database, edge cases (empty, not-found)
- New API endpoints: integration tests in `backend/test/v2/api/`
- Compilation changes: golden files in `test_data/compiled-workflows/`
- Pipeline execution changes: E2E tests in `backend/test/end2end/`
- New algorithmic code (BFS, tree walks): dedicated unit tests -- E2E alone is insufficient

## Proportional coverage

| Change size | Expected |
|-------------|----------|
| Bug fix (< 50 lines) | Unit test reproducing bug + fix |
| Small feature (50-200) | Unit tests + integration if API-facing |
| Medium feature (200-500) | Unit + integration + golden files |
| Large feature (500+) | Unit + integration + E2E + goldens + negative tests |

## Validation symmetry

- Validation at N layers (API server, driver, SDK) requires tests at ALL N layers
- Error messages across layers must use consistent terminology

## Test organization

- Utilities go in shared packages (`backend/test/testutil/`, `e2e_utils/`), not individual test files
- New tests go in existing test files following project conventions, not new files unless introducing a new module
- Test helpers in `_test.go` or test utility packages, not production code

## Assertions and matchers

- Assertion failure messages must match what the assertion actually checks
- Use specific assertions (`Equal(y)`) over generic (`NotTo(BeNil())`) when value is known
- If matchers are relaxed (e.g., skipping `SecurityContext`), add compensating test elsewhere

## Test naming

- Follow pattern: `"valid input - <type> <feature>"` / `"invalid input - <type> <feature>"`
- Group related cases together; use descriptive parameter names

## Fixture integrity

- Never modify shared fixtures for new features -- create new ones
- Golden files must reflect ALL changes the code produces, not a subset
- Tests must not normalize violations (no `"Valid - Privileged"` for forbidden features)
- Sleep durations: 5s max (not 20s+); no hardcoded paths or env-specific values

## Resource management

- `defer response.Body.Close()` after HTTP calls
- `defer f.Close()` after file opens
- Tests order-independent and parallel-safe (no name collisions)
- Clean up resources in `AfterEach`, not via namespace deletion

## Proto change test coverage

Three-file sync required. Full process in [testing-ci.md](../../docs/agents/testing-ci.md#proto-changes-require-test-coverage).

## CI workflow quality

- Top-level comment explaining purpose
- Matrix job names include all parameters: `name: Tests - K8s=${{ matrix.k8s_version }}`
- New `go.mod` dependencies: justified, maintained, license-compatible
