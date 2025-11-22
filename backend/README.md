# Kubeflow Pipelines backend API

## Before you begin

Tools needed:

* [docker](https://docs.docker.com/get-docker/)
* [make](https://www.gnu.org/software/make/)
* [java](https://www.java.com/en/download/)
* [python3](https://www.python.org/downloads/)

Set the environment variable `API_VERSION` to the version that you want to generate. We use `v1beta1` as example here.

```bash
export API_VERSION="v2beta1"
```

## Compiling `.proto` files to Go client and swagger definitions

Use `make generate` command to generate clients using a pre-built api-generator image:
```bash
make generate
```

Go client library will be placed into:

* `./${API_VERSION}/go_client`
* `./${API_VERSION}/go_http_client`
* `./${API_VERSION}/swagger`

> **Note**
> `./${API_VERSION}/swagger/pipeline.upload.swagger.json` is manually created, while the rest of `./${API_VERSION}/swagger/*.swagger.json` are compiled from `./${API_VERSION}/*.proto` files.

## Compiling Python client

To generate the Python client, run the following bash script (requires `java` and `python3`).

```bash
./build_kfp_server_api_python_package.sh
```

Python client will be placed into `./${API_VERSION}/python_http_client`.

## Updating of API reference documentation

> **Note**
> Whenever the API definition changes (i.e., the file `kfp_api_single_file.swagger.json` changes), the API reference documentation needs to be updated.

API definitions in this folder are used to generate [`v1beta1`](https://www.kubeflow.org/docs/components/pipelines/v1/reference/api/kubeflow-pipeline-api-spec/) and [`v2beta1`](https://www.kubeflow.org/docs/components/pipelines/v2/reference/api/kubeflow-pipeline-api-spec/) API reference documentation on kubeflow.org. Follow the steps below to update the documentation:

1. Install [bootprint-openapi](https://github.com/bootprint/bootprint-monorepo/tree/master/packages/bootprint-openapi) and [html-inline](https://www.npmjs.com/package/html-inline) packages using `npm`:
   ```bash
   npm install -g bootprint
   npm install -g bootprint-openapi
   npm -g install html-inline
   ```

2. Generate *self-contained html file(s)* with API reference documentation from `./${API_VERSION}/swagger/kfp_api_single_file.swagger.json`:

   For `v1beta1`:

   ```bash
   bootprint openapi ./v1beta1/swagger/kfp_api_single_file.swagger.json ./temp/v1
   html-inline ./temp/v1/index.html > ./temp/v1/kubeflow-pipeline-api-spec.html
   ```

   For `v2beta1`:

   ```bash
   bootprint openapi ./v2beta1/swagger/kfp_api_single_file.swagger.json ./temp/v2
   html-inline ./temp/v2/index.html > ./temp/v2/kubeflow-pipeline-api-spec.html
   ```

3. Use the above generated html file(s) to replace the relevant section(s) on kubeflow.org. When copying the content, make sure to **preserve the original headers**.
   - `v1beta1`: file [kubeflow-pipeline-api-spec.html](https://github.com/kubeflow/website/blob/master/content/en/docs/components/pipelines/v1/reference/api/kubeflow-pipeline-api-spec.html).
   - `v2beta1`: file [kubeflow-pipeline-api-spec.html](https://github.com/kubeflow/website/blob/master/content/en/docs/components/pipelines/v2/reference/api/kubeflow-pipeline-api-spec.html).

4. Create a PR with the changes in [kubeflow.org website repository](https://github.com/kubeflow/website). See an example [here](https://github.com/kubeflow/website/pull/3444).

## Updating API generator image (Manual)

This is now automatic on pushes to GitHub branches, but the instructions are kept here in case you need to do it
manually.

API generator image is defined in [Dockerfile](`./Dockerfile`). If you need to update the container, follow these steps:

1. Login to GHCR container registry: `echo "<PAT>" | docker login ghcr.io -u <USERNAME> --password-stdin` 
   * Replace `<PAT>` with a GitHub Personal Access Token (PAT) with the write:packages and `read:packages` scopes, as well as `delete:packages` if needed. 
2. Update the [Dockerfile](`./Dockerfile`) and build the image by running `docker build -t ghcr.io/kubeflow/kfp-api-generator:$BRANCH .`
3. Push the new container by running `docker push ghcr.io/kubeflow/kfp-api-generator:$BRANCH`.
4. Update the `PREBUILT_REMOTE_IMAGE` variable in the [Makefile](./Makefile) to point to your new image.
5. Similarly, push a new version of the release tools image to `ghcr.io/kubeflow/kfp-release:$BRANCH` and run `make push` in [test/release/Makefile](../../test/release/Makefile).

## TLS Certificate Rotation (Pod-to-Pod TLS)

### Context

When pod-to-pod TLS is enabled (see PR #12082), backend components (API server, launcher, persistence agent, cache, metadata writer, etc.) use TLS certificates stored in Kubernetes Secret(s). These certificates expire and must be rotated periodically. Updating the Secret object does not automatically make running pods load the new certificate: a restart (rolling restart) of the affected deployments is required so that they mount/read the new secret.

This section documents a recommended, minimal process for renewing TLS certificates and applying them safely.

### Which secrets and components are typically impacted

TLS certificates are commonly stored in a Kubernetes TLS Secret (for example: kfp-pod-tls) in the KFP namespace (commonly kubeflow).

Which secrets and components are typically impacted:

* `pipelines-api-server`
* `pipelines-persistenceagent`
* `pipelines-metadata-writer`
* `pipelines-cache`
* `(Any other pipeline-related deployments that reference the TLS secret)`

> **Note**
> Confirm the exact secret name(s) and deployments used in your cluster before running commands:

   ```bash
   kubectl get secrets -n <namespace>
   kubectl get deploy -n <namespace> | grep -i pipeline
   ```

### Recommended rotation procedure (example)

Prepare/obtain new cert and key
Generate or obtain new cert files: `server.crt` and `server.key` (PEM encoded).

# Update the Kubernetes TLS secret

Replace <namespace> and <secret-name> with your cluster's values:

   ```bash
   kubectl create secret tls <secret-name> \
      --cert=server.crt \
      --key=server.key \
      --dry-run=client -o yaml | kubectl apply -f -
   ```

Example:

   ```bash
   kubectl create secret tls kfp-pod-tls \
      --cert=server.crt \
      --key=server.key \
      --dry-run=client -o yaml | kubectl apply -f -
   ```

# Restart affected deployments (rolling restart)

Restart the deployments so they mount/read the updated secret:

   ```bash
   kubectl rollout restart deploy -n <namespace> pipelines-api-server
   kubectl rollout restart deploy -n <namespace> pipelines-persistenceagent
   kubectl rollout restart deploy -n <namespace> pipelines-metadata-writer
   kubectl rollout restart deploy -n <namespace> pipelines-cache
   ```

Or restart all pipeline-related deployments discovered earlier.

# Verify rollout and pod readiness

   ```bash
   kubectl get pods -n <namespace>
   kubectl rollout status deploy/pipelines-api-server -n <namespace>
   ```

Check logs for errors:

   ```bash
   kubectl describe secret <secret-name> -n <namespace>
   ```

Optional: connect to the API server over TLS (from inside cluster or via port-forward) to confirm certificate in use.

## Best practices & notes

* Rotation interval: rotate certificates before they expire; recommended: track expiry via:

   ```bash
   openssl x509 -enddate -noout -in server.crt
   ```

* Automation: if using cert-manager, automate issuance and renewal, and consider automating rollout restarts (see below).
* Minimize disruption: perform rollouts during maintenance windows or low activity windows.
* Errors to expect: expired certs usually lead to TLS handshake failures; check both client and server logs for diagnosis.
* Auditing: record rotation times and certificate fingerprints for traceability.

## Optional: automate restart after secret update

Kubernetes does not automatically restart pods when a Secret changes. Options:

* Use an operator or controller that watches the Secret and triggers kubectl rollout restart on dependent deployments.
* Use cert-manager and the common checksum/annotation pattern to trigger a redeploy when the Secret changes:
   * Add an annotation to deployments like `kubectl.kubernetes.io/restartedAt` or a checksum of the secret in the deployment spec; update it after cert renewal to trigger a rolling update.