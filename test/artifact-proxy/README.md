# Artifact authorization acceptance

These fixtures validate the proposed master/3.0 storage-authorization work in
[#14883](https://github.com/kubeflow/pipelines/pull/14883). They do not describe
shipping 2.18 adoption resources or certify the separate
[2.18 compatibility rollback](https://github.com/kubeflow/pipelines/pull/15044).
See [release scope and first adoption](../../docs/operator-guides/artifact-authorization.md#release-scope-and-first-adoption).

The `KFP E2E Pipeline tests` workflow automatically runs managed and external
artifact authorization acceptance for pull requests and pushes selected by its
path filters (and release-branch merge-queue events). It builds the tested revision
and deploys multi-user KFP on stock Kind. A focused manual run remains available:

```bash
gh workflow run e2e-test.yml --repo OWNER/pipelines --ref BRANCH \
  -f artifact_authorization_only=true
```

The `artifact-authorization-acceptance` artifact contains the check results and
cluster diagnostics. A successful image build alone is not an acceptance pass;
the **Artifact authorization full-cluster acceptance** job must succeed. Ordinary
manual dispatch skips this additional job unless `artifact_authorization_only=true`.
The managed fixture checks legacy `minio://mlpipeline/artifacts/` isolation through
both default `auto` and a current-image shared UI explicitly configured as `storage`,
while proving the stock namespace credentials can still read/list those old paths.

To use an existing **disposable** multi-user test cluster with the candidate UI
and profile controller deployed:

```bash
PYTHON=/path/to/python-with-boto3 \
  bash test/artifact-proxy/test-artifact-authorization.sh \
    --namespace kubeflow --output-dir /tmp/artifact-authorization
```

The fixture creates test Profiles and objects, temporarily pauses the profile
webhook, holds one receiver pending with a zero-Pod quota, and rotates a test
tenant's storage credentials. It also removes the managed HeadBucket permission
to verify recovery for existing namespaces while retaining every cross-tenant
Deny statement; cleanup restores the original bucket policy if recovery fails.
Do not run it against a production or shared development installation.

The external-storage lane also corrupts and repairs shared adoption configuration
before and after approval. Managed recovery must retain the same authorization
Secret and child-resource UIDs, Namespace ownership, and populated credential
Secret contents. A successful artifact read alone is insufficient: losing
ownership can break the next reconciliation. The lane verifies historical reads
within the approved store and rejects earlier bundled-store references after an
intentional switch to external storage.

The pinned CI Profile controller binds owners to `kubeflow-admin`, which this
minimal CI installation does not provide. The fixture therefore gives each test
User a namespace-scoped Role and RoleBinding granting only `get` on
`kubeflow.org/viewers` in that User's test namespace. Requests still go through
the running UI and real KFP API/SAR; the other test namespace remains unauthorized.
Port forwarding supplies the trusted user header normally set by ingress, so this
does not test ingress login or identity-header sanitization.

Master tests an actual 2.17.2 receiver followed by automatic replacement. The
deferred [#14884 prototype](https://github.com/kubeflow/pipelines/pull/14884) also
tested the 2.17.2 shared UI against a new receiver on a 2.18 base. That is
historical prototype evidence, not a shipping 2.18 capability. Master records that
old-UI check as inapplicable because it removed the V1 authorization API required
by that UI; receiver compatibility is checked separately. These checks do not
establish a complete 2.18-to-3.0 upgrade or eliminate the need to qualify first
adoption from a release without the new approval/readiness resources.

Keep the smaller `test-artifact-proxy.sh` workflow-output smoke test and
`test/seaweedfs/artifact_authorization_test.sh` storage/protocol suite. They cover
different boundaries and do not replace this cluster acceptance.
