# Auto-generation of API reference documentation

This directory contains API definitions. They are used to generate [the API reference on kubeflow.org](https://www.kubeflow.org/docs/pipelines/reference/api/kubeflow-pipeline-api-spec/).

- Use the tools [bootprint-openapi](https://github.com/bootprint/bootprint-monorepo/tree/master/packages/bootprint-openapi) and [html-inline](https://github.com/substack/html-inline) to generate the API reference from [kfp_api_single_file.swagger.json](https://github.com/kubeflow/pipelines/blob/master/backend/api/swagger/kfp_api_single_file.swagger.json). These [instructions](https://github.com/bootprint/bootprint-monorepo/tree/master/packages/bootprint-openapi#bootprint-openapi) have shown how to generate *a single self-contained html file* which is the API reference, from a json file.

- Use the above generated html to replace the html section, which is below the title section, in the file [kubeflow-pipeline-api-spec.html](https://github.com/kubeflow/website/blob/master/content/en/docs/pipelines/reference/api/kubeflow-pipeline-api-spec.html)

Note: whenever the API definition changes (i.e., the file [kfp_api_single_file.swagger.json](https://github.com/kubeflow/pipelines/blob/master/backend/api/swagger/kfp_api_single_file.swagger.json) changes), the API reference needs to be updated.