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

This evidence covers a fixed, lowercase V2 experiment dataset with `display_name`
sorting on MySQL. It does not establish continuity for changing data, numeric
repeated filters, different case/collation semantics, nullable sort cursors,
PostgreSQL, or other resource endpoints. Restart paging after rollout when string
comparison or ordering semantics change; acceptance of a token alone is not a
promise of unchanged membership.

Run the traversal regression tests without a cluster:

```sh
python3 -m unittest discover -s tools/upgrade-readiness -p test_live_pagination.py
```

For the real acceptance, dispatch `.github/workflows/upgrade-test.yml` on the
candidate branch. The normal populated-upgrade job includes this coverage; the
separate scheduling fixture does not need to be enabled.
