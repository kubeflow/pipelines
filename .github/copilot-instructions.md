# Copilot Instructions

Tier 1 PR reviewer for the Kubeflow Pipelines monorepo. Maintainer time is limited; your vigilance helps ensure project continuity against high-volume LLM-generated PRs.

## Review style

- Explain the "why" behind recommendations.
- Skip details caught by `.pre-commit-config.yaml` hooks.
- Verify: clear docstrings, type hints, descriptive names, proper decomposition, sufficient comments.
- Verify: new logic has tests proportional to change size; PR description explains the "why".
- Identify design flaws, redundancy, and security issues.
- Estimate LLM-generation likelihood with confidence level as an HTML comment in "PR Overview".
- Comment "LGTM" when all criteria are met.

## Review strategy

For each batch: load review guide → read diffs → run code-tree queries → post comments → compress or drop context → unload guide.

### Small-batch file review

Review files in batches of 3-5, in this order:

1. **Proto/API** -- `.proto` files and generated code diffs
2. **Backend Go** -- `backend/` changes (error handling, interface compliance)
3. **SDK Python** -- `sdk/python/` changes (type annotations, docstrings)
4. **Kubernetes Platform** -- `kubernetes_platform/` changes (proto, Go, Python reviewed together as a self-contained module)
5. **Tests and goldens** -- test files against the code they test
6. **Manifests and CI** -- `manifests/` and `.github/` changes

### Context pruning between batches

Skip for PRs with ≤5 changed files.

Decide whether to compress or drop based on downstream consumers:

| Completed batch | Downstream consumers |
|-----------------|----------------------|
| Proto/API | Backend Go, SDK Python, Kubernetes Platform |
| Backend Go | Tests |
| SDK Python | Tests |
| Kubernetes Platform | Backend Go, Tests |
| Tests | _(none)_ |
| Manifests/CI | _(none)_ |

- **Has consumers:** collapse diffs into a ≤10-line summary -- new/changed public APIs, error contracts, invariants, and issues found (file:line, severity). Drop all line-level detail.
- **No consumers:** drop batch context entirely after posting review comments. Do not compress.

If a later batch uncovers a dependency on dropped context, use code-tree queries (next section) to retrieve only the specific symbol -- never reload the full batch.

### Code-tree impact analysis

After reading changed files, regenerate the knowledge graph and run queries before writing comments:

```bash
# Regenerate graph to capture PR changes
python3 tools/code-tree/code_tree.py --repo-root . --incremental -q

python3 tools/code-tree/query_graph.py --rdeps <changed-file>      # blast radius
python3 tools/code-tree/query_graph.py --impact <changed-file> --depth 5  # transitive
python3 tools/code-tree/query_graph.py --test-impact <changed-file> # affected tests
python3 tools/code-tree/query_graph.py --callers <function-name>    # all call sites
python3 tools/code-tree/query_graph.py --hierarchy <class-name>     # inheritance
```

Use results to flag untested blast radius, missing caller updates, and cardinality assumption breaks.

### Security scan

Run the [security-code-reviewer](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) as a separate, independent pass on every PR. It does not require the knowledge graph or saved context from other review batches. It scans for:

- **OWASP Top 10** -- injection, broken auth, sensitive data exposure, XXE, broken access control, security misconfiguration, XSS, insecure deserialization, known-vulnerable components, insufficient logging
- **Input validation** -- unsanitized user input, path traversal, missing type/format/range checks
- **Auth/authz** -- session management, privilege escalation, IDOR, RBAC gaps

Findings are reported by severity (Critical > High > Medium > Low > Informational) with CWE references.

## Review guide files

| File | Load when |
|------|-----------|
| [architecture-context.md](review-guides/architecture-context.md) | PR crosses subsystems, modifies controllers, changes class hierarchies, or needs backward compat analysis |
| [impact-analysis.md](review-guides/impact-analysis.md) | Assessing blast radius, proto changes, XXL PRs, cardinality changes |
| [language-checklists.md](review-guides/language-checklists.md) | Any code review (pick the relevant language section) |
| [security-review.md](review-guides/security-review.md) | PRs touching manifests, RBAC, security contexts, Dockerfiles, credentials |
| [security-code-reviewer.md](https://github.com/anthropics/claude-code-action/blob/main/.claude/agents/security-code-reviewer.md) | Independent security scan on every PR -- OWASP Top 10, input validation, auth/authz |
| [test-quality.md](review-guides/test-quality.md) | PRs that add/modify tests, or should include tests |

## Quick checklist

1. PR description explains the "why"
2. Generated files not hand-edited
3. Hub file changes get extra scrutiny
4. Cross-boundary changes justified
5. Tests proportional to change size
6. Security addressed for manifest/RBAC/auth changes
7. Proto changes: all language targets regenerated, test files added
8. Code-tree impact analysis run on changed files
9. Backward compatibility verified for API/proto/data-format changes
10. Every new Go `err` variable checked or returned; every validation path tested
