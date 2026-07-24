# Pipelines API Reference (v2beta1)

This document describes the API specification for the `v2beta1` Kubeflow Pipelines REST API.

## About the REST API

In most deployments of the [Kubeflow Platform](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-ai-reference-platform), the Kubeflow Pipelines REST API is available under the `/pipeline/` HTTP path.
For example, if you host Kubeflow at `https://kubeflow.example.com`, the API will be available at `https://kubeflow.example.com/pipeline/`.

:::{tip}
We recommend using the [Kubeflow Pipelines Python SDK](../sdk.md) as it provides a more user-friendly interface.
See the [Connect SDK to the API](../../user-guides/core-functions/connect-api.md) guide for more information.
:::

### Authentication

How requests are authenticated and authorized will depend on the distribution you are using.
Typically, you will need to provide a token or cookie in the request headers.

Please refer to the documentation of your [Kubeflow distribution](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-ai-reference-platform) for more information.

### Example Usage

To use the API, you will need to send HTTP requests to the appropriate endpoints.

For example, to list pipeline runs in the `team-1` namespace, send a `GET` request to the following URL:

```
https://kubeflow.example.com/pipeline/apis/v2beta1/runs?namespace=team-1
```

## Swagger UI

The API schema is maintained in the checked-in `v2beta1` specification linked below.

:::{note}
The _try it out_ feature of Swagger UI does not work due to authentication and CORS, but it can help you construct the correct API calls.
:::

Interactive Swagger UI embedding is not supported in this documentation. Open the [checked-in `v2beta1` API specification](https://github.com/kubeflow/pipelines/blob/master/backend/api/v2beta1/swagger/kfp_api_single_file.swagger.json) in [Swagger UI](https://github.com/swagger-api/swagger-ui) or another OpenAPI viewer.
