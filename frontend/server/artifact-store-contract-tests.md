# Artifact storage compatibility tests

The UI server uses a bucket-bound `ArtifactStore` with two operations: streaming
`openRead` and an async `list`. The S3 implementation uses AWS SDK v3; consumers
do not receive SDK client types. Endpoint and credential resolution stay in the
factory, while previews, decompression and directory archives remain shared.
The existing native GCS path and public storage configuration are preserved.

Run the local HTTP protocol fixtures without cloud credentials:

```bash
cd frontend/server
npm test -- artifact-store.contract.test.ts
```

These fixtures exercise real AWS SDK serialization and signing against a local HTTP
server. They verify AWS S3 virtual-host addressing, GCS XML API, Cloudflare R2 and
IBM COS path-style endpoints, signing regions and session tokens. They also cover
unsigned access, encoded keys, pagination, cancellation, transient retries,
credential refresh, cached region correction within the retry budget without following
service-controlled Location headers, and truncated response bodies. They do not validate live cloud
authentication, TLS, provider policies, or complete provider compatibility.
Native GCS remains covered by its existing tests; Azure Blob is a different API
and is not implemented by this S3 adapter.

## Real SeaweedFS and MinIO

Use the repository's Node version, install the server dependencies with `npm ci`,
and have Docker running. The runner starts isolated, loopback-only services,
creates a unique test bucket, and cleans its processes, container, and temporary
data on exit. No cloud credentials are required. Credentials in the fixtures are
public test values.

Build the official MinIO source for the test fixture with Go installed:

```bash
git clone --depth 1 --branch RELEASE.2025-10-15T17-29-55Z \
  https://github.com/minio/minio.git /tmp/kfp-contract-minio-source
cd /tmp/kfp-contract-minio-source
git rev-parse HEAD # 9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a
go build -o /tmp/kfp-contract-minio .
```

From the KFP repository root:

```bash
MINIO_BINARY=/tmp/kfp-contract-minio bash frontend/scripts/test-artifact-store-contract.sh
```

The runner pins SeaweedFS `4.48`, matching the deployment manifest. The real-service
checks read keys containing Unicode, spaces, plus signs and literal percent
escapes; list 1,004 objects across the S3 page boundary; check sizes and missing
objects; cancel reads; and read public objects without signing. MinIO uses a mixed-case nondefault
`RegionOne` signing region while the adapter starts with no region configured. Both services must
run for this command to pass; neither service test is skipped. A deliberately wrong-region
SDK read must fail with `400 AuthorizationHeaderMalformed`, verifying the MinIO
region setting before testing adapter recovery. Region correction consumes an attempt,
so the adapter uses a three-attempt budget.

For services already running with the fixture credentials (and MinIO configured
with `MINIO_SITE_REGION=RegionOne`), run directly:

```bash
cd frontend/server
KFP_SEAWEEDFS_ENDPOINT=http://127.0.0.1:18333 \
KFP_MINIO_ENDPOINT=http://127.0.0.1:19000 \
  npm test -- artifact-store.contract.test.ts
```

Direct invocation skips real-service suites whose endpoint variable is absent.
Only use disposable services: the tests create and delete buckets and install a
public-read policy on each test bucket.
