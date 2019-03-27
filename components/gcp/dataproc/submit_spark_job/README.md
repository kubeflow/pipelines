
# Dataproc - Submit Spark Job

## Intended Use
A Kubeflow Pipeline component to submit a Spark job on YARN in Google Cloud Dataproc service. 

## Run-Time Parameters:
Name | Description
:--- | :----------
project_id | Required. The ID of the Google Cloud Platform project that the cluster belongs to.
region | Required. The Cloud Dataproc region in which to handle the request.
cluster_name | Required. The cluster to run the job.
main_jar_file_uri | The HCFS URI of the jar file that contains the main class.
main_class | The name of the driver's main class. The jar file that contains the class must be in the default CLASSPATH or specified in jarFileUris. 
args | Optional. The arguments to pass to the driver. Do not include arguments, such as --conf, that can be set as job properties, since a collision may occur that causes an incorrect job submission.
spark_job | Optional. The full payload of a [SparkJob](https://cloud.google.com/dataproc/docs/reference/rest/v1/SparkJob).
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

### Prepare Spark job
Upload your Spark jar file to a Google Cloud Storage (GCS) bucket. In the sample, we will use a jar file that is pre-installed in the main cluster `file:///usr/lib/spark/examples/jars/spark-examples.jar`. 

Here is the [source code of example](https://github.com/apache/spark/blob/master/examples/src/main/java/org/apache/spark/examples/JavaSparkPi.java).

To package a self-contained spark application, follow the [instructions](https://spark.apache.org/docs/latest/quick-start.html#self-contained-applications).

### Set sample parameters


```python
PROJECT_ID = '<Please put your project ID here>'
CLUSTER_NAME = '<Please put your existing cluster name here>'
REGION = 'us-central1'
SPARK_FILE_URI = 'file:///usr/lib/spark/examples/jars/spark-examples.jar'
MAIN_CLASS = 'org.apache.spark.examples.SparkPi'
ARGS = ['1000']
EXPERIMENT_NAME = 'Dataproc - Submit Spark Job'
COMPONENT_SPEC_URI = 'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataproc/submit_spark_job/component.yaml'
```

### Install KFP SDK
Install the SDK (Uncomment the code if the SDK is not installed before)


```python
# KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.12/kfp.tar.gz'
# !pip3 install $KFP_PACKAGE --upgrade
```

### Load component definitions


```python
import kfp.components as comp

dataproc_submit_spark_job_op = comp.load_component_from_url(COMPONENT_SPEC_URI)
display(dataproc_submit_spark_job_op)
```

### Here is an illustrative pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataproc submit Spark job pipeline',
    description='Dataproc submit Spark job pipeline'
)
def dataproc_submit_spark_job_pipeline(
    project_id = PROJECT_ID, 
    region = REGION,
    cluster_name = CLUSTER_NAME,
    main_jar_file_uri = '',
    main_class = MAIN_CLASS,
    args = json.dumps(ARGS), 
    spark_job=json.dumps({ 'jarFileUris': [ SPARK_FILE_URI ] }), 
    job='{}', 
    wait_interval='30'
):
    dataproc_submit_spark_job_op(project_id, region, cluster_name, main_jar_file_uri, main_class,
        args, spark_job, job, wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
    
```

### Compile the pipeline


```python
pipeline_func = dataproc_submit_spark_job_pipeline
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
