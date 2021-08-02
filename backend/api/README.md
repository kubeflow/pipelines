# Kubeflow Pipelines API

## Before You Start

Tools needed:

* Docker
* Make

## Auto-generation of Go client and swagger definitions

```bash
make
```

Code will be generated in:

* `./go_client`
* `./go_http_client`
* `./swagger`

## Auto-generation of Python client

```bash
./build_kfp_server_api_python_package.sh
```

Code will be generated in `./python_http_client`.

## Auto-generation of API reference documentation

This directory contains API definitions. They are used to generate [the API reference on kubeflow.org](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/).

* Use the tools [bootprint-openapi](https://github.com/bootprint/bootprint-monorepo/tree/master/packages/bootprint-openapi) and [html-inline](https://github.com/substack/html-inline) to generate the API reference from [kfp_api_single_file.swagger.json](https://github.com/kubeflow/pipelines/blob/master/backend/api/swagger/kfp_api_single_file.swagger.json). These [instructions](https://github.com/bootprint/bootprint-monorepo/tree/master/packages/bootprint-openapi#bootprint-openapi) have shown how to generate *a single self-contained html file* which is the API reference, from a json file.

* Use the above generated html to replace the html section, which is below the title section, in the file [kubeflow-pipeline-api-spec.html](https://github.com/kubeflow/website/blob/master/content/en/docs/pipelines/reference/api/kubeflow-pipeline-api-spec.html)

Note: whenever the API definition changes (i.e., the file [kfp_api_single_file.swagger.json](https://github.com/kubeflow/pipelines/blob/master/backend/api/swagger/kfp_api_single_file.swagger.json) changes), the API reference needs to be updated.

## Auto-generation of api generator image

```bash
make push
```

When you update the [Dockerfile](`./Dockerfile`), to make sure others are using the same image as you do:

1. push a new version of the api generator image to gcr.io/ml-pipeline-test/api-generator:latest.
2. update the PREBUILT_REMOTE_IMAGE var in Makefile to point to your new image.
3. push a new version of the release tools image to gcr.io/ml-pipeline-test/release:latest, run `make push` in [test/release/Makefile](../../test/release/Makefile).
