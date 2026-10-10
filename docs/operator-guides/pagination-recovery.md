# Pagination across an upgrade

A page token identifies a position within an ordered listing. It does not describe
which rows a client has already consumed. Kubeflow Pipelines 2.18 places database
NULL values last; older MySQL ascending listings placed them first. Continuing a
cursor across that change can skip or repeat records.

## Server behavior

New tokens carry an ordering version. The server rejects unversioned tokens for
ascending sorts whose fields can contain NULL, even when the particular cursor
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

Existing SDK clients expose the API error; upgrading the server does not add
automatic recovery to already installed clients. On this specific error:

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
