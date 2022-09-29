# Fybrik KFP component

This is a component that can be used in Kubeflow Pipelines to read two input datasets (ex: test and training data), and to write the results of whatever
processing is done on that data **in a secure and governed fashion by leveraging [Fybrik](https://fybrik.io)**

It is assumed that Fybrik is installed together with the chosen Data Catalog and Data Governance engine, and that:

* training and testing datasets have been registered in the data catalog
* governance policies have been defined in the data governance engine

## Prerequisits

The prerequisits and links for installing them are found [here](https://github.com/fybrik/kfp-components/tree/main/get_data_endpoints#prerequisits)

## Setup for Using this Component

Please follow the directions [here](https://github.com/fybrik/kfp-components/tree/main/get_data_endpoints#setup-for-using-this-component) to configure the kubeflow pipeline environment to work with fybrik.

## Component Usage

This component receives the following parameters, all of which are strings.

Input:

* train_dataset_id -  data catalog ID of the dataset on which the ML model is trained
* test_dataset_id - data catalog ID of the dataset containing the testing data

Outputs:

* train_endpoint - virtual endpoint used to read the training data
* test_endpoint - virtual endpoint used to read the testing data
* result_endpoint - virtual endpoint used to write the results

### Example Pipeline Snippet
A full sample may be seen in the [house price estimates sample](../../../samples/contrib/fybrik/house_price_estimates).

```python
def pipeline(
    test_dataset_id: str,
    train_dataset_id: str
):
       
    # Where to store parameters passed between workflow steps
    result_name = "submission-" + st(run_name)

    # Default - could also be read from the environment
    namespace = kubeflow
    
    # Get the ID of the run.  Make sure it's lower case and starts with a letter 
    run_name = "run-" + dsl.RUN_ID_PLACEHOLDER.lower()

    getDataEndpointsOp = components.load_component_from_file(
        'https://github.com/fybrik/kfp-components/blob/master/get_data_endpoints/component.yaml') 

    getDataEndpointsStep = getDataEndpointsOp(
        train_dataset_id=train_dataset_id, 
        test_dataset_id=test_dataset_id, 
        namespace=namespace, 
        run_name=run_name, 
        result_name=result_name)

    #...

    trainModelOp = components.load_component_from_file(
        './train_model/component.yaml')
    trainModelStep = trainModelOp(
        train_endpoint_path='%s' % getDataEndpointsStep.outputs['train_endpoint'],
        test_endpoint_path='%s' % getDataEndpointsStep.outputs['test_endpoint'],
        result_name=result_name,
        result_endpoint_path='%s' % getDataEndpointsStep.outputs['result_endpoint'],
        train_dataset_id=train_dataset_id,
        test_dataset_id=test_dataset_id,
        namespace=namespace)
```

## Development

If you wish to enhance or contribute to this component, please note that it is written in python and packaged as a docker image.  The source code is in the [fybrik kfp component repository](https://github.com/fybrik/kfp-components/tree/main/get_data_endpoints).
