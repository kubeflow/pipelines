# Artifact authorization acceptance

The focused `KFP E2E Pipeline tests` dispatch builds the selected revision and
deploys multi-user KFP on stock Kind. Run it on a branch containing the fixture:

```bash
gh workflow run e2e-test.yml --repo OWNER/pipelines --ref BRANCH \
  -f artifact_authorization_only=true
```

The `artifact-authorization-acceptance` artifact contains the check results and
cluster diagnostics. A successful image build alone is not an acceptance pass;
the **Artifact authorization full-cluster acceptance** job must succeed.

To use an existing **disposable** multi-user test cluster with the candidate UI
and profile controller deployed:

```bash
PYTHON=/path/to/python-with-boto3 \
  bash test/artifact-proxy/test-artifact-authorization.sh \
    --namespace kubeflow --output-dir /tmp/artifact-authorization
```

The fixture creates test Profiles and objects, temporarily pauses the profile
webhook, holds one receiver pending with a zero-Pod quota, and rotates a test tenant's storage credentials. Do not run it against a
production or shared development installation. Requests go through the running
UI and real API authorization; port forwarding supplies the trusted user header
normally set by ingress, so this does not test ingress login.

Both branches test an actual 2.17.2 receiver followed by automatic replacement.
The 2.18 branch also tests the 2.17.2 shared UI against a new receiver. Master
records that old-UI check as inapplicable because it removed the V1 authorization
API required by that UI; receiver compatibility is checked separately.

Keep the smaller `test-artifact-proxy.sh` workflow-output smoke test and
`test/seaweedfs/artifact_authorization_test.sh` storage/protocol suite. They cover
different boundaries and do not replace this cluster acceptance.
