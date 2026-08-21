# Pipelines API Reference (v2beta1)

This document describes the API specification for the `v2beta1` Kubeflow Pipelines REST API.

## About the REST API

In most deployments of the [Kubeflow Community Distribution](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-community-distribution), the Kubeflow Pipelines REST API is available under the `/pipeline/` HTTP path.
For example, if you host Kubeflow at `https://kubeflow.example.com`, the API will be available at `https://kubeflow.example.com/pipeline/`.

:::{tip}
We recommend using the {doc}`Kubeflow Pipelines Python SDK </python-sdk>` as it provides a more user-friendly interface.
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

```{raw} html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css" crossorigin="anonymous">

<style>
#kfp-swagger-ui { background:#fff; border-radius:8px; padding:0.5rem 1.25rem; margin-top:0.75rem; }
#kfp-swagger-ui .information-container { display:none; }
#kfp-swagger-ui .scheme-container { display:none; }
</style>

<p style="margin-bottom:0.35rem;">Enter the base URL of your Kubeflow Pipelines API:</p>
<input id="kfp-api-base-url" type="url" value="https://kubeflow.example.com/pipeline/" disabled
       style="width:100%;box-sizing:border-box;padding:0.5rem;margin-bottom:1rem;font-family:monospace;">

<div id="kfp-swagger-ui"></div>

<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js" crossorigin="anonymous"></script>
<script>
(function () {
  var input = document.getElementById("kfp-api-base-url");
  var requestInterceptor = function (req) {
    if (req.loadSpec) return req;
    try {
      var base = new URL(input.value);
      var reqUrl = new URL(req.url);
      base.pathname = base.pathname.replace(/\/$/, "") + reqUrl.pathname;
      base.search = reqUrl.search;
      req.url = base.toString();
    } catch (e) {}
    return req;
  };
  window.addEventListener("load", function () {
    SwaggerUIBundle({
      url: "../../_static/kfp_api_single_file.swagger.json",
      dom_id: "#kfp-swagger-ui",
      presets: [SwaggerUIBundle.presets.apis],
      requestInterceptor: requestInterceptor,
      syntaxHighlight: { activated: true, theme: "idea" }
    });
    input.disabled = false;
    input.addEventListener("input", function () {
      try { new URL(input.value); input.style.background = "#f3ffef"; }
      catch (e) { input.style.background = "#ffefef"; }
    });
  });
})();
</script>
```
