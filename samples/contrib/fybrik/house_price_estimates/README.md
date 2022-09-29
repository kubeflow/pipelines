# Fybrik Based Housing Price Estimate Pipeline Sample

## Pipeline Overview

This is an enhancement of the pipeline for [house price prediction](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/versioned-pipeline-ci-samples/kaggle-ci-sample).  We show how by integrating with [Fybrik](https://fybrik.io/v1.0/) the same pipeline benefits from transparent handling of non-functional requirements related to the reading and writing of data, such as the handling of credentials, automatic enforcement of data governance, and automatic allocation and cataloging of the new dataset containing the results.  

Please note that data is read/written via the arrow-flight protocol in this example.

A more detailed explanation of the goals, architecture, and demo screen shots can be found [here](https://drive.google.com/file/d/1xn7pGe5pEAEZxnIzDdom9r7K6s78alcP/view?usp=sharing).

The source code for this sample is in the [fybrik repository](https://github.com/fybrik/kfp-components/tree/main/samples/house_price_estimates).

## Prerequisits

* [get_data_endpoints prerequisits and setup](../../../../components/contrib/fybrik/get_data_endpoints/README.md#prerequisits)

## Usage

A detailed description of how to configure everything needed to run the sample may be found [here](https://github.com/fybrik/kfp-components/tree/main/samples/house_price_estimates#usage).

### Upload Pipeline

Download to your computer either the [pipeline-argo.yaml](https://github.com/fybrik/kfp-components/blob/main/samples/house_price_estimates/pipeline-argo.yaml) or [pipeline-tekton.yaml](https://github.com/fybrik/kfp-components/blob/main/samples/house_price_estimates/pipeline-tekton.yaml), and then upload it via the Kubeflow Pipeline GUI. Which you need depends on whether you installed kubeflow pipelines with Argo (default) or with Tekton Pipelines.

See slide 17 of the [demo presentation](https://drive.google.com/file/d/1xn7pGe5pEAEZxnIzDdom9r7K6s78alcP/view?usp=sharing).

### Run the Pipeline

Create a pipeline run via the Kubeflow Pipeline GUI, providing the following parameters:

* test_dataset_id: testpii-csv
* train_dataset_id:  trainpii-csv

See slide 19 of the [demo presentation](https://drive.google.com/file/d/1xn7pGe5pEAEZxnIzDdom9r7K6s78alcP/view?usp=sharing).

### View Pipeline Run Details

Click on each step in the Kubeflow Pipeline GUI and view its log to see the output.  

* get-data-endpoints prints and returns the virtual endpoints for the 3 dataset (test, train, results)
* visualize-table prints a preview of the data, and you can see that the personal information has been obfuscated
* train-model runs trains the machine learning model and then runs it on the test data, printing out a sample of the results
* submit-result prints out the data catalog asset ID of the newly created asset containing the results

See slides 21-24 of the [demo presentation](https://drive.google.com/file/d/1xn7pGe5pEAEZxnIzDdom9r7K6s78alcP/view?usp=sharing).

### Making Changes to the Pipeline

If you want to experiment with changing the pipeline, see [these instructions](https://github.com/fybrik/kfp-components/tree/main/samples/house_price_estimates#making-changes-to-the-pipeline).
