# Kubeflow Pipelines backend API

## Before you begin

Tools needed:

* [docker](https://docs.docker.com/get-docker/)
* [make](https://www.gnu.org/software/make/)
* [java](https://www.java.com/en/download/)
* [python3](https://www.python.org/downloads/)

Set the environment variable `API_VERSION` to the version that you want to generate. We use `v1beta1` as example here.

```bash
export API_VERSION="v1beta1"
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

## Updating API generator image

API generator image is defined in [Dockerfile](`./Dockerfile`). If you need to update the container, follow these steps:

1. Update the [Dockerfile](`./Dockerfile`) and build the image by running `docker build -t gcr.io/ml-pipeline-test/api-generator:latest .`
1. Push the new container by running `docker push gcr.io/ml-pipeline-test/api-generator:latest` (requires to be [authenticated](https://cloud.google.com/container-registry/docs/advanced-authentication)).
1. Update the `PREBUILT_REMOTE_IMAGE` variable in the [Makefile](./Makefile) to point to your new image.
1. Similarly, push a new version of the release tools image to `gcr.io/ml-pipeline-test/release:latest` and run `make push` in [test/release/Makefile](../../test/release/Makefile).
