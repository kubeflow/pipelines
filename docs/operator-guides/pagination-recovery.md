# Pagination across an upgrade

A page token identifies a position within an ordered listing. It does not describe
which rows a client has already consumed. Kubeflow Pipelines 2.18 places database
NULL values last; older MySQL ascending listings placed them first. Continuing a
cursor across that change can skip or repeat records. Older descending pagination
also omitted trailing NULL rows, and historical scanners could encode a NULL cursor
as an empty string or zero. Both sort directions therefore require protection.

## Server behavior

New tokens carry an ordering version. The server rejects unversioned tokens for
sorts whose fields can contain NULL in either direction, even when the particular cursor
contains a non-NULL value: other rows in the same listing may still be NULL.
Compatible legacy tokens remain accepted. An unsupported ordering version also
requires a new listing. Existing identifier, filter and authorization checks remain
in force; the version is not an authorization credential.

The server returns gRPC `FAILED_PRECONDITION` (code 9), exposed as HTTP 400 by the
REST gateway, with an actionable message and this entry in `details`:

```json
{
  "@type": "type.googleapis.com/google.rpc.ErrorInfo",
  "reason": "PAGINATION_RESTART_REQUIRED",
  "domain": "kubeflow.org"
}
```

Clients should recognize the structured reason and domain rather than matching
message text. The server does not silently return page one in response to an
incompatible cursor.

## UI behavior

The updated UI recognizes this condition, clears its saved page tokens and
selection, and requests page one once with the same filters, sort and page size.
A brief notice explains that the list returned to its first page. If the retry
fails, the error is displayed; the UI does not retry indefinitely.

## SDK and custom clients

The updated Python SDK exports `kfp.client.is_pagination_restart_required(error)`
to recognize this condition without matching message text. List methods still
raise the original API exception and do not retry. Upgrading the server does not
add this helper or automatic recovery to already installed clients.

```python
from kfp.client import is_pagination_restart_required

try:
    page = client.list_runs(page_token=saved_token, sort_by="finished_at")
except Exception as error:
    if is_pagination_restart_required(error):
        # Discard or reconcile earlier results before starting a new traversal.
        saved_token = None
    raise
```

The updated `kfp` CLI prints restart guidance and exits nonzero. Remove
`--page-token` only after discarding or reconciling earlier output. It does not
silently print a replacement first page.

The handwritten Go API clients preserve this condition through wrapped errors.
Use `api_server.IsPaginationRestartRequired(err)` to recognize it. Their list
methods do not retry; `ListAll` returns an error without partial results. If
reusing its request parameters, clear the page token before a new traversal.

On this specific error:

1. Discard the saved page token and any accumulated results from that traversal.
2. Repeat the original list request without `page_token`, retaining its criteria.
3. Continue using the tokens returned by the new traversal. Bound retries and
   surface a persistent error instead of looping during a rollout.

Do not append the restarted first page to an earlier partial export. For consumers
that already performed side effects per record, reconcile prior progress before
replaying records. These steps restart a listing, not a workload, pod or service.

## Mixed-version rollout

Tokens for affected sorts use a versioned format that older servers reject, rather
than interpreting them with the old ordering. Older servers cannot return the new
structured restart reason, so these requests can show a generic invalid-token
error until the API-server rollout completes. Compatible sorts retain the existing
wire format. A one-time UI retry is not a guarantee that an affected listing can
complete while requests alternate between old and new replicas.

Coordinate the API-server rollout and verify affected listings after all replicas
are updated. This contract addresses NULL ordering; it does not promise continuity
across arbitrary collation changes, pre-existing source pagination defects, or
changes to the underlying records while a listing is in progress.
