# Python based visualizations guideline

This document describes the architecture of python based visualizations,
development guidelines to contribute new predefined visualizations to the
Kubeflow Pipelines project, and current limitations. Python based visualizations
are a new method of generating visualizations within Kubeflow Pipelines that
allow for rapid development, experimentation, and customization when
visualizing results. For information about Python based visualizations and how
to use them, please visit the [documentation page](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations).

Please check the [developer guidelines](https://github.com/kubeflow/pipelines/blob/master/developer_guide.md)
for additional development guidelines.

## Architecture

Python based visualizations rely on three parts: the frontend, the API server,
and the Python visualization service. The frontend is responsible for creating
the visualization request and displaying the results of the created requests.
The API server is responsible for transposing the request provided by the
frontend to a request that is understandable by the python visualization
service, returning the result of the transposed request to the frontend, and
gracefully handling incorrectly formatted requests from the frontend and any
errors encountered with the Python visualization service. Finally, the Python
visualization service is responsible for generating a visualization from a
provided request.

## How to create predefined visualizations

1. Determine if the visualization should become a predefined visualization.
Consider the following:
    * How often will it be used?
        * Frequently used visualizations are a good candidate for predefine
        visualization.
    * How complex is it?
        * The complexity of a visualization can reduce its usability. Predefined
        visualizations are intended to be powerful and simple. Visualizations
        that require extensive or complex variables are not good candidates for
        predefined visualizations. 
2. Fork the Kubeflow Pipelines repository.
3. Add a new type for the visualization within the [visualization.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/visualization.proto#L78)
file in the `backend/api` directory.
    * The name of the visualization should be in screaming snake case (that is
    `VISUALIZATION_NAME`).
4. Run [`./generate_api.sh`](https://github.com/kubeflow/pipelines/blob/master/backend/api/generate_api.sh)
within the `backend/api` directory to generate the Swagger API definition for
the backend.
5. Download the [Swagger Codegen](https://swagger.io/tools/swagger-codegen/)
jar file.
    * Currently, version 2.3.1 of the Swagger Codegen jar file is used to
    generate the frontend API. Should this become out of date, the version can
    be checked within the [VERSION](https://github.com/kubeflow/pipelines/blob/master/frontend/src/apis/visualization/.swagger-codegen/VERSION)
    file for the visualization [Swagger Codegen directory](https://github.com/kubeflow/pipelines/tree/master/frontend/src/apis/visualization/.swagger-codegen).
    * This step is only required if the Swagger Codegen jar file is not present
    in the `frontend` directory. If you already have the jar file, you can skip
    steps 6 and 7.
6. Place the Swagger Codegen jar file in the `frontend` directory.
7. Rename the Swagger Codegen jar file to **swagger-codegen-cli.jar**.
8. Run `npm run apis:visualization` within the `frontend` directory to generate
the Swagger API definition for the frontend.
9. Create a new Python file that will be executed to generate a visualization.
    * Python 3 **MUST** be used.
    * The new Python file should be created within the
    `backend/src/apiserver/visualization` directory and it should have the same
    name as the type that was created earlier, use snake case instead of
    screaming snake case (that is `visualization_name.py`).
    * Dependency injection is used to pass variables from the Kubeflow Pipelines
    UI to a visualization.
        * To obtain a path or path pattern from the Kubeflow Pipelines UI, you
        can use the following syntax:

            ```python
            # The variable "source" will be injected to any visualization. The
            # value of "source" will be provided by the Kubeflow Pipelines UI
            # and will never be an empty string.
            ...
            # Open a file with a provided path or path pattern from the
            # Kubeflow Pipelines UI and append DataFrame to an array of
            # DataFrames
            dfs = []
            for f in file_io.get_matching_files(source):
                dfs.append(pd.read_csv(f))
            ...
            # Get a path from the Kubeflow Pipelines UI and create a DataFrame
            df = pd.read_csv(source)
            ...
            ```
            * Additional details about how this is implemented can be found in
            the [server.py](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/server.py#L127)
            file.
        * To obtain additional variables from the Pipelines UI, you can use
        the following syntax:

            ```python
            # Get a value for a specified key
            key = variables.get("key")
            # Get a value for a specified key with a default
            key = variables.get("key", "default_value")
            # Check if a value for a specified key exists
            if variables.get("key", "default_value") is "default_value":
                # Value for a specified key does not exist
                pass
            else:
                # Value for a specified key does exist
                pass
            ```
            * Additional details about how this is implemented can be found in
            the [exporter.py](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/exporter.py#L93)
            file and the [Python documentation](https://docs.python.org/3/library/stdtypes.html?highlight=dict#dict.get).
10. Add any new dependencies to the [requirements.txt](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/requirements.txt)
file in the `backend/src/apiserver/visualization` directory.
11. Add any new dependencies to the [third_party_licenses.csv](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/third_party_licenses.csv)
file.
    * The following format is used:
        ```csv
        package_name,url_to_package_license,license_name
        ```
    * The columns of the csv are as follows:
        * `package_name` is the name of the package on [pypi](https://pypi.org/).
        * `url_to_package_license` is the url where the license of the package
        can be downloaded from.
        * `license_name` is the name of package license.
    * Examples for all the columns can be found in the [third_party_licenses.csv](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/third_party_licenses.csv)
    file.
12. Submit these changes as a Pull Request or build docker image for usage
within your cluster.

## Known limitations

* Multiple visualizations cannot be generated concurrently.
    * This is because a single Python kernel is used to generate visualizations.
    * If visualizations are a major part of your workflow, it is recommended to
    increase the number of replicas within the [visualization deployment YAML](https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize/base/pipeline/ml-pipeline-visualization-deployment.yaml)
    file or within the visualization service deployment itself.
        * _Please note that this does not directly solve the issue, instead it
        decreases the likelihood of experiencing delays when generating
        visualizations._
* Visualizations that take longer than 30 seconds will fail to generate.
    * For visualizations where the 30 second timeout is reached, you can add the
    **TimeoutValue** header to the request made by the frontend, specifying a
    _positive integer as ASCII string of at most 8 digits_ for the length of
    time required to generate a visualization as specified by the
    [grpc documentation](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests).
    * For visualizations that take longer than 100 seconds, you will have to
    specify a **TimeoutValue** within the request headers **AND** change the
    default kernel timeout of the visualization service. To change the default
    kernel timeout of the visualization service, set the **KERNEL_TIMEOUT**
    environment variable of the visualization service deployment to be the new
    timeout length in seconds within the [visualization deployment YAML](https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize/base/pipeline/ml-pipeline-visualization-deployment.yaml)
    file or within the visualization service deployment itself.

    ```YAML
    - env:
      - name: KERNEL_TIMEOUT
        value: 100
    ```
* The HTML content of the generated visualizations cannot be larger than 4MB.
    * gRPC by default imposes a limit of 4MB as the maximum size that can be
    sent and received by a server. To allow for visualizations that are larger
    than 4MB in size to be generated, you must manually set
    **MaxCallRecvMsgSize** for gRPC. This can be done by editing the provided
    options given to the gRPC server within [main.go](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/main.go#L128)
    to 
    ```golang
    var maxCallRecvMsgSize = 4 * 1024 * 1024
	if serviceName == "Visualization" {
		// Only change the maxCallRecvMesSize if it is for visualizations
		maxCallRecvMsgSize = 50 * 1024 * 1024
    }
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize)),
		grpc.WithInsecure(),
	}
    ```

## Update python dependencies
1. Edit `requirements.in` with additional changes.
1. Run `./update_requirements.sh` to re-resolve dependencies.
1. Pinned dependencies are in `requirements.txt`.
