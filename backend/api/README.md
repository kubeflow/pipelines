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

Use `make generate` command to generate clients:
```bash
make generate
```

**Default behavior (fast):** Uses pre-built image for faster development.

**When pre-built images may be outdated:** Force building from source:
```bash
USE_PREBUILT_IMAGE=false make generate
```

The source build automatically:
1. Builds the API generator Docker image from `backend/api/Dockerfile`
2. Runs the generator inside the container to create client libraries
3. Caches the built image for subsequent runs (using `.image-built` target)

Go client library will be placed into:

* `./${API_VERSION}/go_client`
* `./${API_VERSION}/go_http_client`
* `./${API_VERSION}/swagger`

> **Note**
> `./${API_VERSION}/swagger/pipeline.upload.swagger.json` is manually created, while the rest of `./${API_VERSION}/swagger/*.swagger.json` are compiled from `./${API_VERSION}/*.proto` files.

## Compiling Python client

To generate the Python client, use the `generate-kfp-server-api-package` Make target:

```bash
make generate-kfp-server-api-package
```

**Default behavior (fast):** Uses pre-built image for faster development.

**When pre-built images may be outdated:** Force building from source:
```bash
USE_PREBUILT_IMAGE=false make generate-kfp-server-api-package
```

The source build automatically:
1. Builds the API generator Docker image (with Java) from source if needed
2. Runs the Python client generation script inside the container
3. The image already includes Java required for the OpenAPI generator

Alternatively, you can run the script directly if you have Java and Python3 installed locally:

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

    Fov `v1beta1`:

   ```bash
   bootprint openapi ./v1beta1/swagger/kfp_api_single_file.swagger.json ./temp/v1
   html-inline ./temp/v1/index.html > ./temp/v1/kubeflow-pipeline-api-spec.html
   ```

   For `v2beta1`:

   ```bash
   bootprint openapi ./v2beta1/swagger/kfp_api_single_file.swagger.json ./temp/v2
   html-inline ./temp/v2/index.html > ./temp/v2/kubeflow-pipeline-api-spec.html
   ```

3. Use the above generated html file(s) to replace the relevant section(s) on kubeflow.org. When copying th content, make sure to **preserve the original headers**.
   - `v1beta1`: file [kubeflow-pipeline-api-spec.html](https://github.com/kubeflow/website/blob/master/content/en/docs/components/pipelines/v1/reference/api/kubeflow-pipeline-api-spec.html).
   - `v2beta1`: file [kubeflow-pipeline-api-spec.html](https://github.com/kubeflow/website/blob/master/content/en/docs/components/pipelines/v2/reference/api/kubeflow-pipeline-api-spec.html).

4. Create a PR with the changes in [kubeflow.org website repository](https://github.com/kubeflow/website). See an example [here](https://github.com/kubeflow/website/pull/3444).

## Local development and Docker image management

### Development Workflow Options

The API generation workflow supports two modes:

**1. Fast Development (default):** Uses pre-built images
```bash
make generate                           # Uses pre-built image (fast)
make generate-kfp-server-api-package   # Uses pre-built image (fast)
```

**2. Source Build (accurate):** Builds from current source
```bash
USE_PREBUILT_IMAGE=false make generate                           # Builds from source
USE_PREBUILT_IMAGE=false make generate-kfp-server-api-package   # Builds from source
```

**CI/Validation:** Always builds from source to ensure accuracy.

### When to Use Each Mode

- **Pre-built images**: Regular development, prototyping, testing
- **Source build**: When pre-built images may have outdated tool versions, when changing API generation tools, updating dependencies, or before committing API changes

### Legacy Targets

- `make generate-from-scratch`: Legacy target, always builds from source

### Docker Requirements

- **Docker**: Required for all API generation operations
- **Docker Buildx**: Automatically set up for improved caching performance

### Manual API Generator Image Publishing

The `build-tools-images.yml` CI workflow automatically publishes API generator images to GitHub Container Registry. Manual publishing is typically not needed, but if required:

1. Update the [Dockerfile](./Dockerfile) with your changes
2. The image will be built and published automatically on the next push to a tracked branch
3. For manual publishing, see the `build-tools-images.yml` workflow for the exact commands
