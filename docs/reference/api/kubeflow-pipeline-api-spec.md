# Pipelines API Reference (v2beta1)

This document describes the API specification for the `v2beta1` Kubeflow Pipelines REST API.

## About the REST API

In most deployments of the [Kubeflow Community Distribution](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-community-distribution), the Kubeflow Pipelines REST API is available under the `/pipeline/` HTTP path.
For example, if you host Kubeflow at `https://kubeflow.example.com`, the API will be available at `https://kubeflow.example.com/pipeline/`.

:::{tip}
We recommend using the [Kubeflow Pipelines Python SDK](../sdk.md) as it provides a more user-friendly interface.
See the [Connect SDK to the API](../../user-guides/core-functions/connect-api.md) guide for more information.
:::

### Authentication

How requests are authenticated and authorized will depend on the distribution you are using.
Typically, you will need to provide a token or cookie in the request headers.

Please refer to the documentation of your [Kubeflow distribution](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-distributions) for more information.

### Example Usage

To use the API, you will need to send HTTP requests to the appropriate endpoints.

For example, to list pipeline runs in the `team-1` namespace, send a `GET` request to the following URL:

```
https://kubeflow.example.com/pipeline/apis/v2beta1/runs?namespace=team-1
```

## Swagger UI

The API reference is automatically generated from the [`2.17.0`](https://github.com/kubeflow/pipelines/releases/tag/2.17.0) version of Kubeflow Pipelines for the [`v2beta1` REST API](https://github.com/kubeflow/pipelines/blob/2.17.0/backend/api/v2beta1/swagger/kfp_api_single_file.swagger.json).

:::{note}
The _try it out_ feature of Swagger UI does not work due to authentication and CORS, but it can help you construct the correct API calls.
:::
