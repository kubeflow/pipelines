# Go source discovery contract for Dockerfiles

This document is the normative contract for Go source discovery performed by
`go-version-metadata`. Changes to this contract require an intentional policy
change and corresponding changes to the authoritative typed scenarios in
`docker_contract_test.go`. The JSON is a checked export of those scenarios. A
new example alone is not a reason to extend an ad hoc scanner.

The contract is tied to the BuildKit Dockerfile parser version pinned in the
root `go.mod`. BuildKit owns Dockerfile parsing, escape-directive handling,
continuation removal, JSON decoding, typed instruction fields, and Docker word
normalization. The helper must not independently emulate those rules.

## Results

Every Dockerfile produces exactly one result:

- `managed`: the file has exactly one top-level Go source and that source is a
  single-line instruction of the exact form
  `FROM golang:X.Y.Z[-flavor]@sha256:<64 lowercase hex> AS <alias>`. Keywords,
  spaces, repository name, digest spelling, and alias syntax must match the
  canonical form exactly.
- `unsupported`: the file has a statically visible Go source that is not the
  one canonical source, has more than one canonical source, uses an unsupported
  shell for a shell-form executable field, uses shell syntax outside the
  supported POSIX grammar, or uses a valid Docker construct whose semantics are
  explicitly outside the bounded policy below.
- `irrelevant`: none of the included value fields contains a statically visible
  Go source and no unsupported shell is used to interpret an executable field.
- `invalid`: the pinned parser or typed-instruction parser rejects the
  Dockerfile, one of the explicitly listed offline structural checks below
  fails, an instruction is not part of the pinned typed instruction set, or a
  parsing/resource contract is exceeded.

A canonical source does not hide another source: one canonical source plus any
unsupported source is `unsupported`. An `ONBUILD` payload is never managed;
payloads that BuildKit forbids, including `ONBUILD FROM`, are `invalid`.

## What is a Go source

A Go image source is an image reference whose final repository component is
exactly `golang`, optionally qualified by a registry, namespace, tag, digest,
or Docker transport prefix. Docker repository names are lowercase; concrete
token discovery is additionally ASCII-case-insensitive so invalid uppercase
spellings fail conservatively. Names such as `golangci`, `my-golang`,
`golang.foo`, and `golang/tools` are not Go image sources.

A Go download source is a URL whose normalized value starts a Go distribution
download under `https://go.dev/dl/go...` or the legacy
`https://dl.google.com/go/go...` location. URL scheme and host matching are
case-insensitive.

In executable fields the policy is deliberately conservative: any static Go
image token or Go download URL in a semantic value is a source, regardless of
which program consumes it. The helper does not try to recognize `docker pull`,
`curl`, `crane`, or any other command by name.

## Included typed fields

Only semantic values are inspected. Identifiers, keys, destinations, and raw
instruction text are not substituted for typed values.

| Instruction | Included values | Reason |
| --- | --- | --- |
| `FROM` | `BaseName` | Pulls an external image unless it is a prior local stage. |
| `COPY` | `From` | May pull an external image. Ordinary sources and the destination are excluded. |
| `RUN --mount` | Each mount's `From` | May pull an external image. Other mount fields are excluded. |
| `ADD` | Remote source URLs | Docker fetches them. Local sources and the destination are excluded. |
| `RUN` | Shell command, exec argv, and inline/heredoc files | Executable content. |
| `CMD`, `ENTRYPOINT` | Shell command or exec argv | Executable content. |
| `HEALTHCHECK CMD` | Shell command or exec argv | Executable content. `NONE` and options are excluded. |
| `ARG` | Default values only | Reserved propagation values; arbitrary ARG dataflow is not evaluated. |
| `ENV` | Values and key-policy validation | Reserved propagation values; arbitrary ENV dataflow is not evaluated. |
| `ONBUILD` | The same included fields in its recursively parsed payload | Deferred execution does not exempt a source. |

Go-bearing `ARG` and `ENV` values are reserved even when unused. This is the
explicit alternative to partially evaluating Docker variable flow. Their names
are identifiers and are permitted, including the name `golang`, except for
names that collide with POSIX special parameters or ASCII positional digit
parameters. Docker permits declarations such as `ARG 0=go` and then expands
`$0`. Literal ARG names in that set are `unsupported`. ENV keys are first
normalized with the pinned BuildKit lexer, so quoted and escaped spellings
cannot bypass the same restriction; ENV keys containing variable expansion are
also `unsupported` rather than partially evaluated. This includes valueless ARG
declarations. Unicode digits are ordinary Docker identifiers and do not fall
under this restriction.

`LABEL`, `MAINTAINER`, `WORKDIR`, `EXPOSE`, `USER`, `VOLUME`, `STOPSIGNAL`,
ordinary `COPY` operands, local `ADD` operands, destinations, option values not
listed above, comments, and parser directives are excluded. `SHELL` configures
interpretation but its argv is not itself a Go source field.

An instruction outside the pinned typed instruction set is `invalid`. This
prevents an unknown instruction from silently introducing a new source-bearing
field. Updating BuildKit therefore requires reviewing this table and the
conformance matrix.

## Local identifiers

Local stage identifiers are not image sources, but each instruction has its
own resolver. BuildKit lowercases aliases when storing them, while `FROM`
looks up its expanded base spelling without lowercasing it: `AS Base` followed
by `FROM base` is local, but `FROM Base` is external. `FROM` resolves only
previously declared named aliases;
`COPY --from` resolves prior named aliases and in-range zero-based numeric stage
indices; `RUN --mount=from` resolves only named aliases, so a numeric token is
an external image name. COPY and RUN named lookups are case-insensitive.
Negative, out-of-range, or forward numeric `COPY --from` indices are `invalid`. The external base of
`FROM ... AS golang` is still classified before the new alias is recorded, so
`FROM golang:latest AS golang` is unsupported.

Stage aliases, ARG names, ENV names, shell assignment names, and shell parameter
names are identifiers. A `golang` substring in an identifier is not a source.
Assignment values and parameter operator operands are values and are inspected.
Thus `golang=alpine` and `${#golang}` are irrelevant, while
`IMAGE=golang:latest` and `${IMAGE:-golang:latest}` are unsupported.

The offline stage-graph checks cover those instruction-specific top-level
rules, plus self/cycle detection among local references. Named local references
to the current or a later stage are `invalid` where the instruction resolves
named stages. Deferred `ONBUILD COPY --from` and
`ONBUILD RUN --mount=from` depend on the eventual child build's stage namespace;
numeric references and names that denote the defining/current or a later local
stage are therefore `unsupported`, not guessed. Variable expansion in these
deferred source fields is `invalid`, matching BuildKit's field restriction.
This is not a claim to perform
complete Dockerfile2LLB validation: filesystem/context checks, build-argument
dependent graphs, deferred child-build graphs, and other solver-time checks are
outside this offline contract.

## Docker words and runtime shells

Docker word fields are normalized with BuildKit's lexer using the parsed
Dockerfile escape token. Dockerfile escapes are never reused as runtime-shell
escapes.

Shell-form executable fields use a real POSIX shell parser. The supported shell
configuration is the default Linux shell or exactly `SHELL ["/bin/sh","-c"]`
(the equivalent executable spelling `sh` is also accepted). Quotes concatenate
literal word parts, backslash-newline is removed, comments end at a newline,
assignment names and parameter names remain identifiers, and heredoc contents
are visited through the parsed shell representation. Resource accounting is
performed while visiting lexer/parser nodes and semantic literal values.

Any other explicit `SHELL` makes every subsequent shell-form `RUN`, `CMD`,
`ENTRYPOINT`, or `HEALTHCHECK CMD-SHELL` in that stage `unsupported`, even if
its raw text contains no Go token. Exec-form instructions remain independent of
`SHELL`. A local stage inherits the configured shell from its referenced local
stage; a new external stage starts with the default Linux shell.

Exec form is decoded by BuildKit into argv. Each argv value is inspected as a
static value without inventing semantics for the executable. There is one
explicit recursive rule: argv shaped as `sh -c <script>` or
`/bin/sh -c <script>` is parsed using the supported POSIX shell parser. Arguments
after the script are positional parameters. Other interpreters, including
PowerShell, `cmd.exe`, Python, and `eval`, are not recursively interpreted.

The static contract detects literal concatenation such as `g\olang`,
`go"lang"`, and `go'lang'`, and literal operands inside parameter expansions.
It does not claim to resolve values produced solely at runtime by files, secret
mounts, command substitution, network responses, or an interpreter not covered
above. A statically visible Go token within those constructs is still detected.

Docker-word alternatives are complete only for direct substitution and the
setness/emptiness operators `-`, `:-`, `+`, and `:+`. Other valid parameter
operators, including prefix/suffix pattern removal, are classified
`unsupported` after BuildKit identifies them as active syntax; escaped and
single-quoted spellings remain literals. Unknown values use non-textual typed
identities rather than forgeable sentinel strings. Repeated occurrences of one
variable share one assignment, including partial suffix constraints at every
occurrence. Active Docker special parameters (`$`, `?`, `#`, `!`, `-`, `@`,
and `*`) and ASCII positional digit parameters are always `unsupported` by the
bounded policy. Parent-image environment metadata is unavailable to offline
analysis and can supply those names even when the current Dockerfile has no
declaration. Escaped or single-quoted dollar text remains literal.
Symbolic image values are matched as complete references using a
bounded automaton derived from the `distribution/reference` grammar. The
automaton distinguishes bracketed IPv6 authorities, registry ports, repository
paths, tags, and digests; a colon after a namespace therefore cannot be
reinterpreted as a registry port. It omits only the tag maximum length and
digest-encoding minimum length, which safely over-approximates invalid suffixes
without allowing a Go source to bypass discovery. Variables act as correlated
state transformations, not independent wildcards. They remain conservatively
source-capable when that shared value can introduce path delimiters: for
example, `g${X}l${X}ng` becomes
`g/golal/golang` for `X=/gola`. Image classification tracks repository, tag,
and digest state separately. Symbolic direct substitutions may introduce
repository path, tag, or digest boundaries, so anchored fragments such as
`go${VALUE}latest` are detected when some value can form `golang:<tag>`, while
an unknown inside `alpine:<tag>` cannot change the repository.

POSIX runtime-shell prefix/suffix pattern-removal nodes are replaced by typed
unknown result nodes before the complete word is expanded. This preserves
composition with adjacent defaults and substitutions, so
`${X#?}${Y:-olang:latest}` cannot be split into independently scanned strings.
Ordinary unanchored uses such as removing a filename suffix remain irrelevant.

## Resource contract

The public helper process has a ten-second caller deadline. Independently, the
helper enforces these deterministic limits:

- request contents: 4 MiB;
- parsed Docker instructions: 100,000;
- nested instruction/`ONBUILD` depth: 256;
- discovered candidates: 10,000;
- one Docker word passed to BuildKit normalization: 16 KiB;
- total distinct Docker words passed to BuildKit normalization: 32 KiB;
- distinct variables in one Docker word or normalized runtime-shell symbolic
  value: 6;
- branch alternatives for one Docker word: 729 (all unset, empty, and
  nonempty states for up to six variables);
- total Docker alternative-expansion input work: 1 MiB;
- symbolic image-reference automaton states: 128;
- symbolic variable transformations: 2,048;
- symbolic reference match work: 1 MiB state visits;
- POSIX shell AST nodes: 100,000;
- POSIX shell AST depth: 256; and
- total visited or normalized semantic literal bytes: 16 MiB.

Repeated alias/word objects and normalized symbolic values are memoized and
charged once across Docker and runtime-shell fields. Exhausting symbolic search
work is an explicit `invalid` resource error, never evidence that a source was
found. Limits for shell depth and work are computed from lexer-visible or
AST-visible structures, not
from raw character patterns such as counting `${`. Exceeding a deterministic
limit is `invalid`; exceeding the external deadline is a helper failure. Fuzz
tests must assert bounded completion and no panic for inputs within the public
request-size limit.

Before invoking BuildKit word normalization, every semantic Docker word is
charged against the per-word and total distinct-input limits above. Every typed
field that BuildKit marks expandable is normalized; candidate admission has no
literal prefilter that could skip a value assembled across expansion
boundaries. Both irrelevant and source-bearing nested parameter defaults
therefore complete within the public deadline or fail with a deterministic
resource error. Escaped, quoted, or expanded values within the normalization
budget use the pinned BuildKit implementation.

Typed instruction validation and later candidate inspection share one
Docker-word alternatives cache. Validation preserves the original typed field
instead of replacing it with one normalized result, so the original span,
every feasible result, and all later inspections consume input/work and
normalized-output budgets exactly once.

## Differential acceptance

The typed scenarios in `docker_contract_test.go` are the authoritative
acceptance matrix; `testdata/docker-contract.json` must match them field for
field. Its
`buildkitWordOracles` compare normalized Docker words with the pinned BuildKit
implementation. Its `shellOracles` use a fixed, trusted corpus, an explicitly
listed environment, and a test-provided `capture` executable to compare word
construction with `/bin/sh`. Oracle cases never contain untrusted generated
shell text. Every `dockerConformance` record contains both a named semantic
domain and an executable helper-classification expectation. Ordinary tests run
that expectation; the opt-in Docker lane runs it together with Docker's
acceptance result. A typed scenario may include explicit build arguments as a
Docker-only witness for one feasible symbolic assignment. The helper
classification always uses the unbound Dockerfile and must remain sound for
every assignment. Finding numbers are historical metadata, not a coverage
criterion. `executableCrossProducts` generates every configured source
kind × instruction field × shell/exec/heredoc form × top-level/`ONBUILD`
context. Heredoc forms outside `RUN` remain explicit negative grammar cases,
so adding an axis value cannot silently omit unsupported intersections.

`wordExpansionCrossProducts` freezes the branch semantics of `:-`, `-`, `:+`,
and `+` across ARG/ENV fields and image/download operands. For these four
conditional operators, unset, empty, and a representative nonempty value are
complete branch states: only setness and emptiness select the operand, not the
nonempty value's contents. Direct unknown substitutions are different. They
are not approximated by the arbitrary nonempty sentinel `x`. Their positions
are preserved symbolically so they can bridge statically anchored candidate
fragments (for example, `go${VALUE}ang` with `VALUE=l`) without assuming that
an otherwise unanchored unknown such as `ENV PATH=${PATH}:/usr/local/bin`
injects an arbitrary Go source. For images, intersection is limited to the
repository component; an unknown Alpine tag cannot change `alpine` into the
`golang` repository. Downloads likewise require static text anchoring the
supported HTTPS host/path prefix around the unknown span.

The contract harness freezes the schema as well as the outcomes. BuildKit-word
and POSIX-shell oracle IDs are exact nonempty sets. Executable products require
the complete 2 source × 4 instruction × 3 form × 2 context matrix (48 cases),
and conditional-word products require the complete 2 source × 2 field × 4
operator × 3 state matrix (48 cases). Operator state maps must contain exactly
`unset`, `empty`, and `nonempty`; an omitted list, axis value, or state is a
structural test failure rather than a vacuous pass.

The pinned lexers emit collision-free NUL-framed identities into a private
construction buffer. That buffer is decoded exactly once into typed literal and
variable segments; matching, repository-component analysis, and stored memoized
results never reparse marker strings. The framing is not a second accepted input
syntax. Raw NUL is rejected uniformly for Docker words and runtime scripts before
classification, while other literal control bytes remain ordinary input and
never become symbolic variables.
Repeated occurrences of one variable or one rewritten runtime pattern result
share an identity. Quote and escape provenance is preserved when deciding
whether a Docker parameter operator is active, so an active `${X}` elsewhere
cannot activate literal `\${X#?}` or `'${X#?}'` text.

Run the Docker-backed grammar corpus on a host with Docker using
`KFP_RUN_DOCKER_CONFORMANCE=1 go test ./tools/go-version-metadata -run
TestDockerContractAgainstDocker`. The runner prepends `# check=skip=all` so the
result represents Dockerfile acceptance rather than optional build-check lint
warnings and bounds captured diagnostic output. The ordinary test suite still
executes every helper oracle when Docker is unavailable.
The coverage test requires exactly one executable Docker-backed case for each
numeric FROM, numeric RUN mount, negative COPY index, normalized local FROM
alias, raw external FROM alias, and case-insensitive COPY alias domain; an
empty or partial conformance list cannot satisfy the contract.

Contract acceptance requires every matrix case, both differential suites, the
resource-boundary tests, the opt-in Docker-backed corpus when Docker is
available, and the repository consistency tests to pass.

## Updater snapshot invariant

Before external digest resolution or live application, the updater validates
its discovery reads and captures one immutable starting snapshot:

1. the exact `HEAD` object ID;
2. the complete Git index: every tracked path's stage-0 entry as
   `(path, full Git mode, object ID)`, with no higher-stage entries; and
3. every managed path's worktree file type, exact bytes, complete filesystem
   permission mode, device/inode identity, and link count.

Managed paths must be singly linked ordinary files and must initially match their stage-0
index entries. Paths with assume-unchanged or skip-worktree flags, or with
clean/smudge filters, are rejected because Git cannot provide a raw-worktree
transaction invariant for them. The updater must compare the complete
snapshot, including object IDs, before and after planning, before application,
after application, and after repository verification. A lock in the Git common
directory supplies a cooperative single-writer transaction: every updater must
take that lock, and processes that mutate a managed inode without it are outside
the transaction guarantee. Live application opens the existing path without
following symlinks, confirms its inode, bytes, mode, and link count, then writes,
truncates, and fsyncs that same inode before validating both the descriptor and
pathname again. It never publishes by exchanging or unlinking pathnames. An
observed non-cooperating change is rejected, but writes through arbitrary open
descriptors cannot be serialized or detected after final validation.

If the updater's own candidate write, truncate, or fsync fails, it immediately
rewrites and fsyncs the original bytes through the still-open descriptor and
confirms the original descriptor and pathname state before propagating the
failure. If that restoration or its validation also fails, automatic recovery
is not claimed: the update fails with the durable recovery bundle retained.
Recovery artifacts persist original bytes and modes before live mutation and
remain usable when a path is missing or truncated.
