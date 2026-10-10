# Live filtered-pagination upgrade acceptance

The existing `upgrade-test.yml` job creates seven V2 experiments on the actual
2.17.2 API and saves each filter's first page before deploying candidate images.
It retains a separate 2.17.2 API replica against the same MySQL database. The
replica has a distinct service selector; the normal KFP service never routes to
it. Port 8888 targets the candidate and port 8889 targets the old replica.

`live_pagination.py` verifies:

- Exact old page tokens continue on the upgraded API, with repeated criteria and
  with token-only requests, for scalar equality, substring, and string-IN filters.
- Requests alternate explicitly between old and candidate endpoints, in both
  directions. Scalar criteria are repeated or omitted; string-IN uses token-only
  continuation when the old reader participates.
- Every traversal returns the seven expected IDs in order, exactly once, and
  terminates. Token cycles, extra/missing/duplicate rows, and unbounded paging fail.
- Changing the filter on a continuation request is rejected by both versions.
- The old reader's existing repeated-IN limitation returns HTTP 400 for both old
  and candidate tokens. This is a recorded compatibility limitation, not a newly
  accepted success path.

The `pagination-acceptance-<run-id>` artifact records the successful traversals
and limitations. The saved tokens remain local to the disposable runner. The
workflow run identifies the exact candidate revision and same-run image build.

The extended matrix adds 32 V1/V2 experiment and run cases: ascending and
descending name, description, creation time, metric, recurring-run ID, scheduled
time, and finished time sorts. The source creates mixed-case names, real NULL metrics, and unset run fields.
Unset timestamps are stored as zero by the source API; this live fixture does not
claim actual SQL NULL timestamp coverage. Each case uses a separate complete inventory to establish
membership; every fresh candidate traversal must return each expected ID exactly
once in the candidate's order. An old server failure never excuses a fresh
candidate failure.

Nullable run sorts in both directions have explicit rollout boundaries:

- A real saved source token or newly issued old-reader token must receive HTTP
  400 with gRPC code 9 and `google.rpc.ErrorInfo` reason
  `PAGINATION_RESTART_REQUIRED`, domain `kubeflow.org`, from the candidate. A
  generic error or silently accepted cursor fails acceptance.
- A new affected candidate token has the `kfp1:` envelope and ordering version 1.
  The actual 2.17.2 reader must reject it with HTTP 400 / gRPC InvalidArgument;
  accepting any page fails acceptance. Historical readers otherwise discard
  unknown JSON metadata when reissuing tokens, so a JSON version field alone
  cannot make mixed-reader traversal safe. Descending cursors also need this
  boundary: old readers can ignore the NULL marker and restart at page one, or
  omit the NULL suffix after a non-NULL cursor.
- Nonnullable cases with working source pagination and unchanged ordering must
  retain saved-token and mixed-reader continuity. Their new tokens
  remain readable by the historical server.

Restart affected paging from page one against the candidate after rollout. The
historical server's generic rejection cannot provide the new structured recovery
signal. Do not describe affected mixed-reader traversal as compatible, even if a
particular dataset happens to look correct. Where the old server cannot emit a
cursor at all, the artifact records unavailable source evidence explicitly.

This is fixed-dataset MySQL coverage, not proof for concurrent mutations,
PostgreSQL, numeric repeated filters, every nullable field, or other resource
endpoints. Case/collation order changes remain a distinct restart consideration;
acceptance of a token alone is not a promise of unchanged membership.

Run the traversal regression tests without a cluster:

```sh
python3 -m unittest discover -s tools/upgrade-readiness -p test_live_pagination.py
```

For the real acceptance, dispatch `.github/workflows/upgrade-test.yml` on the
candidate branch. The normal populated-upgrade job includes this coverage; the
separate scheduling fixture does not need to be enabled.
