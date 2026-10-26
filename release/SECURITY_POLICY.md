# Release security policies

The default `zero-fixable` policy rejects every finding with an available fix
and every scanner failure. Image scans use Trivy's `comprehensive` detection
priority, including Go dependencies in OS-packaged executables. Raw scanner JSON
and stderr remain in the qualification evidence directory.

## Explicit exception files

For a release that intentionally accepts a known finding, supply a JSON file:

```sh
kfpr run qualify-release --security-exceptions /path/to/exceptions.json
```

This explicitly selects `approved-exceptions`; omitting the option on a fresh
checkpoint keeps `zero-fixable`. The checkpoint remembers the absolute path.
To return to the strict default, select `--security-policy zero-fixable` and
qualify again.

```json
{
  "schema": 1,
  "exceptions": [
    {
      "scope": "source",
      "artifact": "test/tools/example/go.mod",
      "vulnerability": "CVE-2026-12345",
      "package": "example.org/dependency",
      "installed_version": "1.2.3",
      "justification": "Test-only dependency; remediation tracked in the next maintenance update.",
      "expires": "2026-12-31"
    }
  ]
}
```

Copy identities from the retained scanner report. Every field is required;
unknown fields, duplicate identities and wildcards are rejected. `scope` is
`source`, `python`, or `image`. `artifact` is the exact scanner target path for
source, the audited interpreter minor such as `3.9` for Python, or the immutable
`sha256:…` image ID for an image. Image exceptions therefore apply only to the
scanned image, not all future builds of its tag. Use the exact advisory ID the
scanner reports, including PYSEC or GHSA identifiers where applicable.

Expiry is a UTC calendar date, inclusive through that day. An expired file
blocks both qualification and publication. Qualification records the full
exception document and its canonical SHA256 alongside all findings and reasons;
publication rereads the file and verifies that content and expiry still match.
Editing the exception file requires fresh qualification. Keep the file available
until publication finishes. Exceptions never waive scanner errors, malformed
reports, missing targets, missing Python lanes or incomplete image coverage.
`--force` does not bypass these gates.

## Historical 2.17.3 exceptions

`2.17.3-approved-exceptions-v1` is an explicitly selected policy for release
**2.17.3 only**. It preserves the approved Python 3.9 compatibility floor and
records documented nonrelease tooling findings. It never accepts release-image
findings or scanner errors. Another release, target, package version, or advisory
requires a separately reviewed decision; this policy does not extend to them.

The complete, exact allowlist is in
[`kfpr/security_policy_2173.py`](kfpr/security_policy_2173.py). It records the
scope, target, package, advisory ID, installed version, and reason for each
accepted identity. The list is imported with the release tool so switching to a
maintenance branch cannot change or remove the selected policy implementation.

The approved baseline contains 80 unique identities and 83 reported occurrences:

- **74 source findings** in `.github/resources/scripts/kfp-readiness/requirements.txt`,
  `test/frontend-integration-test/package-lock.json`, and
  `test/tools/project-cleaner/go.mod`, classified as `nonrelease-tooling`.
- **6 unique Python 3.9 findings, reported 9 times**: `requests==2.32.5`
  (`PYSEC-2026-2275`) and `urllib3==2.6.3` (`PYSEC-2026-141`,
  `PYSEC-2026-142`, `PYSEC-2026-4175`, `PYSEC-2026-4176`,
  `PYSEC-2026-4177`), classified as `python39-compatibility`. The interpreter
  identity is read from the audited environment, rather than inferred from its
  path or position in the interpreter list. These exceptions do not apply to
  newer Python interpreters.

`pip` and `setuptools` bootstrap findings are **not** accepted. Installer tooling
that is not required by the installed SDK runtime is handled separately before
runtime auditing. Any remaining installer finding still blocks qualification.

## Evidence and publication gates

`security.json` retains every finding in its original scan and reports accepted
and blocking findings separately, with acceptance reasons and exact matching
identities. `fixable_count` remains the total number of fixable findings;
`accepted_count` and `blocking_count` partition it. `residual_count` records
findings with no available fix. Duplicate scanner occurrences remain visible.
The policy succeeds only when `blocking_count` is zero and no scanner errors
occurred. A fixable image finding remains blocking unless an explicit image exception
matches its immutable image ID and full finding identity. Source and Python
exceptions cannot accept image findings.

Publication must bind evidence to the selected policy and release version and
recompute dispositions from the saved scans using `evaluate_policy`. A stored
`passed` flag or accepted count alone is insufficient authorization.

The regression test command is:

```bash
PYTHONPATH=release python -m unittest kfpr.security_test
```
