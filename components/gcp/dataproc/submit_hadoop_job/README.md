
# Dataproc - Submit Hadoop Job

## Intended Use
A Kubeflow Pipeline component to submit a Apache Hadoop MapReduce job on Apache Hadoop YARN in Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
cluster_name | Required. The cluster to run the job.
main_jar_file_uri | The HCFS URI of the jar file containing the main class. Examples: `gs://foo-bucket/analytics-binaries/extract-useful-metrics-mr.jar` `hdfs:/tmp/test-samples/custom-wordcount.jar` `file:///home/usr/lib/hadoop-mapreduce/hadoop-mapreduce-examples.jar`
main_class | The name of the driver's main class. The jar file that contains the class must be in the default CLASSPATH or specified in jarFileUris. 
args | Optional. The arguments to pass to the driver. Do not include arguments, such as -libjars or -Dfoo=bar, that can be set as job properties, since a collision may occur that causes an incorrect job submission.
hadoop_job | Optional. The full payload of a [HadoopJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/HadoopJob).
job | Optional. The full payload of a [Dataproc job](https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs).
wait_interval | Optional. The wait seconds between polling the operation. Defaults to 30s.

## Output:
Name | Description
:--- | :----------
job_id | The ID of the created job.

## Sample

Note: the sample code below works in both IPython notebook or python code directly.

### Setup a Dataproc cluster
Follow the [guide](https://cloud.google.com/dataproc/docs/guides/create-cluster) to create a new Dataproc cluster or reuse an existing one.

### Prepare Hadoop job
Upload your Hadoop jar file to a Google Cloud Storage (GCS) bucket. In the sample, we will use a jar file that is pre-installed in the main cluster, so there is no need to provide the `main_jar_file_uri`. We only set `main_class` to be `org.apache.hadoop.examples.WordCount`.

Here is the [source code of example](https://github.com/apache/hadoop/blob/trunk/hadoop-mapreduce-project/hadoop-mapreduce-examples/src/main/java/org/apache/hadoop/examples/WordCount.java).

To package a self-contained Hadoop MapReduct application from source code, follow the [instructions](https://hadoop.apache.org/docs/current/hadoop-mapreduce-client/hadoop-mapreduce-client-core/MapReduceTutorial.html).

### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'
OUTPUT_GCS_PATH = '<Please put your output GCS path here>'
REGION = 'us-central1'
MAIN_CLASS = 'org.apache.hadoop.examples.WordCount'
INTPUT_GCS_PATH = 'gs://ml-pipeline-playground/shakespeare1.txt'
EXPERIMENT_NAME = 'Dataproc - Submit Hadoop Job'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/7622e57666c17088c94282ccbe26d6a52768c226/components/gcp/dataproc/submit_hadoop_job/component.yaml'
```

### Insepct Input Data
The input file is a simple text file:


```python
!gsutil cat $INTPUT_GCS_PATH
```

    With which he yoketh your rebellious necks Razeth your cities and subverts your towns And in a moment makes them desolate


### Clean up existing output files (Optional)
This is needed because the sample code requires the output folder to be a clean folder.
To continue to run the sample, make sure that the service account of the notebook server has access to the `OUTPUT_GCS_PATH`.

**CAUTION**: This will remove all blob files under `OUTPUT_GCS_PATH`.


```python
!gsutil rm $OUTPUT_GCS_PATH/**
```

    CommandException: No URLs matched: gs://hongyes-ml-tests/dataproc/hadoop/output/**


### Install KFP SDK
Install the SDK (Uncomment the code if the SDK is not installed before)


```python
# KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.12/kfp.tar.gz'
# !pip3 install $KFP_PACKAGE --upgrade
```

### Load component definitions


```python
import kfp.components as comp

dataproc_submit_hadoop_job_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataproc_submit_hadoop_job_op)
```


    <function dataproc_submit_hadoop_job(project_id, region, cluster_name, main_jar_file_uri='', main_class='', args='', hadoop_job='', job='', wait_interval='30')>


### Here is an illustrative pipeline that uses the component


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
    dataproc_submit_hadoop_job_op(project_id, region, cluster_name, main_jar_file_uri, main_class,
        args, hadoop_job, job, wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

### Compile the pipeline


```python
pipeline_func = dataproc_submit_hadoop_job_pipeline
pipeline_filename = pipeline_func.__name__ + '.pipeline.tar.gz'
import kfp.compiler as compiler
compiler.Compiler().compile(pipeline_func, pipeline_filename)
```

### Submit the pipeline for execution


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


Experiment link <a href="/pipeline/#/experiments/details/bc344edd-4cce-4535-8b11-b34a65e549e9" target="_blank" >here</a>



Run link <a href="/pipeline/#/runs/details/23edc062-46b1-11e9-8b9e-42010a800110" target="_blank" >here</a>


### Inspect the outputs

The sample in the notebook will count the words in the input text and output them in sharded files. Here is the command to inspect them:


```python
!gsutil cat $OUTPUT_GCS_PATH/*
```

    AccessDeniedException: 403 

