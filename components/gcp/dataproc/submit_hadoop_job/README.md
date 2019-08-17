
# Name
Data preparation using Hadoop MapReduce on YARN with Cloud Dataproc

# Label
Cloud Dataproc, GCP, Cloud Storage, Hadoop, YARN, Apache, MapReduce


# Summary
A Kubeflow Pipeline component to prepare data by submitting an Apache Hadoop MapReduce job on Apache Hadoop YARN to Cloud Dataproc.

# Details
## Intended use
Use the component to run an Apache Hadoop MapReduce job as one preprocessing step in a Kubeflow  Pipeline. 

## Runtime arguments
| Argument | Description | Optional | Data type | Accepted values | Default |
|----------|-------------|----------|-----------|-----------------|---------|
| project_id | The Google Cloud Platform (GCP) project ID that the cluster belongs to. | No | GCPProjectID |  |  |
| region | The Dataproc region to handle the request. | No | GCPRegion |  |  |
| cluster_name | The name of the cluster to run the job. | No | String |  |  |
| main_jar_file_uri | The Hadoop Compatible Filesystem (HCFS) URI of the JAR file containing the main class to execute. | No | List |  |  |
| main_class | The name of the driver's main class. The JAR file that contains the class must be either in the default CLASSPATH or specified in `hadoop_job.jarFileUris`. | No | String |  |  |
| args | The arguments to pass to the driver. Do not include arguments, such as -libjars or -Dfoo=bar, that can be set as job properties, since a collision may occur that causes an incorrect job submission. | Yes | List |  | None |
| hadoop_job | The payload of a [HadoopJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/HadoopJob). | Yes | Dict |  | None |
| job | The payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs). | Yes | Dict |  | None |
| wait_interval | The number of seconds to pause between polling the operation. | Yes | Integer |  | 30 |

Note: 
`main_jar_file_uri`: The examples for the files are : 
- `gs://foo-bucket/analytics-binaries/extract-useful-metrics-mr.jar` 
- `hdfs:/tmp/test-samples/custom-wordcount.jarfile:///home/usr/lib/hadoop-mapreduce/hadoop-mapreduce-examples.jar`


## Output
Name | Description | Type
:--- | :---------- | :---
job_id | The ID of the created job. | String

## Cautions & requirements
To use the component, you must:
*   Set up a GCP project by following this [guide](https://cloud.google.com/dataproc/docs/guides/setup-project).
*   [Create a new cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster).
*   Run the component under a secret [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a Kubeflow cluster. For example:

    ```python
    component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
    ```
*   Grant the Kubeflow user service account the role `roles/dataproc.editor` on the project.

## Detailed description

This component creates a Hadoop job from [Dataproc submit job REST API](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs/submit).

Follow these steps to use the component in a pipeline:

1.  Install the Kubeflow Pipeline SDK:



```python
%%capture --no-stderr

KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.14/kfp.tar.gz'
!pip3 install $KFP_PACKAGE --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataproc_submit_hadoop_job_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/48dd338c8ab328084633c51704cda77db79ac8c2/components/gcp/dataproc/submit_hadoop_job/component.yaml')
help(dataproc_submit_hadoop_job_op)
```

## Sample
Note: The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.


### Setup a Dataproc cluster
[Create a new Dataproc cluster](https://cloud.google.com/dataproc/docs/guides/create-cluster) (or reuse an existing one) before running the sample code.


### Prepare a Hadoop job
Upload your Hadoop JAR file to a Cloud Storage bucket. In the sample, we will use a JAR file that is preinstalled in the main cluster, so there is no need to provide `main_jar_file_uri`. 

Here is the [WordCount example source code](https://github.com/apache/hadoop/blob/trunk/hadoop-mapreduce-project/hadoop-mapreduce-examples/src/main/java/org/apache/hadoop/examples/WordCount.java).

To package a self-contained Hadoop MapReduce application from the source code, follow the [MapReduce Tutorial](https://hadoop.apache.org/docs/current/hadoop-mapreduce-client/hadoop-mapreduce-client-core/MapReduceTutorial.html).


### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'
OUTPUT_GCS_PATH = '<Please put your output GCS path here>'
REGION = 'us-central1'
MAIN_CLASS = 'org.apache.hadoop.examples.WordCount'
INTPUT_GCS_PATH = 'gs://ml-pipeline-playground/shakespeare1.txt'
EXPERIMENT_NAME = 'Dataproc - Submit Hadoop Job'
```

#### Insepct Input Data
The input file is a simple text file:


```python
!gsutil cat $INTPUT_GCS_PATH
```

### Clean up the existing output files (optional)
This is needed because the sample code requires the output folder to be a clean folder. To continue to run the sample, make sure that the service account of the notebook server has access to the `OUTPUT_GCS_PATH`.

CAUTION: This will remove all blob files under `OUTPUT_GCS_PATH`.


```python
!gsutil rm $OUTPUT_GCS_PATH/**
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc submit Hadoop job pipeline',
    description='Dataproc submit Hadoop job pipeline'
)
def dataproc_submit_hadoop_job_pipeline(
    project_id = PROJECT_ID, 
    region = REGION,
    cluster_name = CLUSTER_NAME,
    main_jar_file_uri = '',
    main_class = MAIN_CLASS,
    args = json.dumps([
        INTPUT_GCS_PATH,
        OUTPUT_GCS_PATH
    ]), 
    hadoop_job='', 
    job='{}', 
    wait_interval='30'
):
    dataproc_submit_hadoop_job_op(
        project_id=project_id, 
        region=region, 
        cluster_name=cluster_name, 
        main_jar_file_uri=main_jar_file_uri, 
        main_class=main_class,
        args=args, 
        hadoop_job=hadoop_job, 
        job=job, 
        wait_interval=wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

#### Compile the pipeline


```python
pipeline_func = dataproc_submit_hadoop_job_pipeline
pipeline_filename = pipeline_func.__name__ + '.zip'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

#### Submit the pipeline for execution


```python
#Specify pipeline argument values
arguments = {}

#Get or create an experiment and submit a pipeline run
import kfp
client = kfp.Client()
experiment = client.create_experiment(EXPERIMENT_NAME)

#Submit a pipeline run
run_name = pipeline_func.__name__ + ' run'
run_result = client.run_pipeline(experiment.id, run_name, pipeline_filename, arguments)
```

### Inspect the output
The sample in the notebook will count the words in the input text and save them in sharded files. The command to inspect the output is:


```python
!gsutil cat $OUTPUT_GCS_PATH/*
```

## References
*   [Component Python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataproc/_submit_hadoop_job.py)
*   [Component Docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
*   [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataproc/submit_hadoop_job/sample.ipynb)
*   [Dataproc HadoopJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/HadoopJob)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
