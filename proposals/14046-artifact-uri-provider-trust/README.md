# KEP-14046: Artifact URI Provider Query Trust Model

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [The rule](#the-rule)
  - [Where it's enforced](#where-its-enforced)
  - [Single-user vs multi-user](#single-user-vs-multi-user)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Frontend Considerations](#frontend-considerations)
- [Migration Strategy](#migration-strategy)
- [Test Plan](#test-plan)
<!-- /toc -->

## Summary

An artifact URI in KFP can carry a query string, like `s3://bucket/path?endpoint=...`. KFP treats
that query as real object-store configuration. This KEP defines who is allowed to set that query
and what it can control, then fixes the launcher and frontend so an admin's own configuration
can no longer be bypassed by it.

Tracks [kubeflow/pipelines#14046](https://github.com/kubeflow/pipelines/issues/14046).

## Motivation

Artifact URIs can be written by a namespace user, not just an admin. An `Importer` step takes a
URI as a constant or a runtime parameter, so a namespace editor can put anything in that query
string, including one for a bucket the admin already configured.

The code that opened the object-store connection used to trust that query the moment one was
present, before checking whether the admin had configured anything different for that exact
bucket. An admin could set `team-bucket` to use `minio.internal:9000` with specific credentials,
and a tenant-authored URI querying `?endpoint=attacker.example` for that same bucket would win,
because the admin's setting was never even checked.

A namespace editor is meant to be a lesser-trusted user than the platform admin. Being able to
make the artifact service connect, using its own live credentials, to a server of the tenant's
choosing is the shape of SSRF: reaching internal services or leaking credentials somewhere they
were never meant to go.

### Goals

- Admin-configured provider settings for a bucket always win over an artifact URI's own query.
- An artifact URI's query is still honored when the admin has configured nothing for that
  provider, so existing pipelines keep working.
- Reject the clearest unsafe cases on that fallback path: `disableSSL=true` and a literal
  loopback, link-local, or private IP as the endpoint.
- Enforce the same rule independently in the Go launcher and the frontend server, so a bug in
  one can't undo the other.

### Non-Goals

- Defending against DNS rebinding or a redirect to an unsafe address. That needs resolve-time IP
  pinning or `NetworkPolicy` egress rules at the cluster level, not a config check.
- Deciding whether the `kfp-launcher` ConfigMap itself needs stricter RBAC. Today a namespace
  owner can already edit it with `kubectl`, so this KEP protects against a browser or API-level
  tenant, not someone with direct namespace access.
- Proving the namespaced artifact service has no more credentials or network reach than a
  tenant's own workloads. If that's already true, most of the severity here goes away on its
  own, but it's a separate piece of work to verify and document.

## Proposal

### The rule

For a bucket outside the pipeline's own root, in order:

1. If the admin has an `Override` for that exact bucket and key prefix, it wins. The artifact's
   query is not even read.
2. If the admin has a `Default` for the provider, even with no bucket-specific override, it
   wins too.
3. Only when the admin has configured nothing at all for that provider does the query get a
   say. Even then, an admin can turn this off with `allowUnmanagedProviderQueries: false`, and
   `disableSSL=true` or a private/loopback/link-local endpoint is rejected regardless.

An artifact under the run's own `defaultPipelineRoot` always inherits that root's trusted
session and never reaches this decision at all. This only matters for artifacts that live
somewhere else, which is only possible through an `Importer` or a custom pipeline root.

A hostname endpoint is not resolved through DNS to check it. That would not stop DNS rebinding
anyway, since the address can change between the check and the real connection, and it would add
a live network call to a function that should just be parsing config.

### Where it's enforced

- `backend/src/v2/config/s3.go`, `S3ProviderConfig.ProvideSessionInfo`. `MinioProviderConfig`
  gets the same behavior through delegation.
- `backend/src/v2/component/launcher_v2.go`, `fetchNonDefaultBuckets`. This is the path a
  component takes to download an artifact it did not produce itself. It used to open a bucket
  with no policy applied at all.
- `frontend/server/helpers/provider-policy.ts`, `resolveS3ProviderInfo`. Applies the same rule
  before the frontend server honors a browser-supplied `providerInfo` value.

### Single-user vs multi-user

In single-user (standalone) mode, this barely matters. Anyone who can submit a pipeline already
has full `kubectl` access to the cluster, so an unusual endpoint through a query string does not
grant anything new.

In multi-user mode the gap is real, but bounded. The `kfp-launcher` ConfigMap lives in the
tenant's own namespace, and Kubeflow's default namespace-owner role already allows editing
ConfigMaps there. So this protects a browser or API-level tenant, someone who can create runs
and author artifact URIs but has no direct `kubectl` access to their namespace. It does not stop
someone who already has that access, since they could edit the policy itself.

### Risks and Mitigations

An admin who has not configured any provider settings for a bucket is still relying on the
unmanaged-query fallback. `allowUnmanagedProviderQueries` defaults to true so nothing already
deployed breaks, and every use of that path logs a deprecation warning so an admin can see who
depends on it before a future release changes the default.

## Frontend Considerations

The frontend cannot share Go code, so `provider-policy.ts` reimplements the same decision by
hand, including the loopback/link-local/private IP check, since Node has no direct equivalent of
Go's `net.IP.IsPrivate`. It is wired into `minio-helper.ts`, the one place a client-supplied
`providerInfo` value is actually applied to a connection.

## Migration Strategy

Nothing changes today for pipelines already relying on an unmanaged query string, because the
gate defaults to true. Pipelines whose artifacts sit under an admin `Override` or `Default` were
already correct and are unaffected.

## Test Plan

- Unit tests proving a matching `Override` and a bare `Default` both win over a hostile query,
  in both the Go and the frontend implementations.
- Unit tests for the `allowUnmanagedProviderQueries` gate in all four states, and for the
  `disableSSL` and private-IP guardrails, in both implementations.
- A regression test for the second Go loophole, proving a downstream component's input artifact
  now goes through the same policy instead of an unguarded default client.
- A fixed test that previously asserted the bypass bug as correct behavior, on both the Go and
  frontend sides.
