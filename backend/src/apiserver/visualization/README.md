# Python based visualizations guideline

This document describes the development guideline to contribute new predefined
visualizations to the Kubeflow Pipeline project. For information about using
Python based visualizations please visit the [documentation page](https://www.kubeflow.org/docs/pipelines/sdk/python-based-visualizations).
Please check the [developer guidelines](https://github.com/kubeflow/pipelines/blob/master/developer_guide.md)
for additional development guidelines.

## How to create predefined visualizations

1. Determine what the new visualization will be.
    * When determining if a visualization should become a predefined
    visualization, consider the following:
        * How often will it be used?
            * The frequency of which a visualization is used is a major factor
            for a predefined visualization. The more a visualization is used,
            the more likely it should be a predefine visualization.
        * How complex is it?
            * The complexity of a visualization can reduce its usability.
            Predefined visualizations are intended to be powerful and simple.
            Visualizations that require extensive or complex variables are not
            good candidates for predefined visualizations. 
2. Fork the Kubeflow Pipelines repository.
3. Add a new type for the visualization within the [visualization.proto](https://github.com/kubeflow/pipelines/blob/master/backend/api/visualization.proto#L78)
file that is within the `backend/api` directory.
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
            ...
            # Check if a variable is provided
            # You should always check if the desired variable is provided before
            # accessing it because accessing a key from a dict when it does not
            # exist will result in an exception being raised.
            if "key" in variables:
              # Variable of name key is provided.
              # Use the value of the specified variable to manipulate the way
              # a visualization is generated here.
              pass
            else:
              # Variable of name key is not provided.
              # You can provide a default operation here if a variable is not
              # provided or throw an error if the variable must be provided.
              pass
            ...
            # Accessing a variable
            key = variables["key"]
            # or
            key = variables.get("key")
            ...
            ```
            * Additional details about how this is implemented can be found in
            the [exporter.py](https://github.com/kubeflow/pipelines/blob/master/backend/src/apiserver/visualization/exporter.py#L93)
            file.
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