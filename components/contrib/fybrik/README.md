# Fybrik KFP Components

This repository contains components that enable [Kubeflow Pipeline](https://www.kubeflow.org/docs/components/pipelines/) users to focus on their business needs while letting [Fybrik](https://fybrik.io) handle the non-functional aspects of dealing with data.  This includes data governance, data plane optimization, and management of dataset credentials so that they do not have to be shared with the users.

## Components

### Get Data Endpoints

[get_data_endpoints](get_data_endpoints/README.md) is a kfp component that receives  catalog IDs of two input files, the training and test dataset IDs.

It returns the virtual endpoints of the two input datasets and a virtual endpoint for the result dataset, to be used by subsequent components in the pipeline for reading the input datasets and for writing the results to the result dataset.  It also catalogs the newly created result dataset, and ensures that data governance policies defined in the data governance engine are enforced during the read and write operations.

The [component's readme](get_data_endpoints/README.md) contains information about prerequisits and installations.

## Samples

### House Price Estimates

The [house price estimate sample](../../../samples/contrib/fybrik/house_price_estimates/) shows how to use the get-data-endpoints component in an example machine learning pipeline.