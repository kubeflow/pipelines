
# Executing an Apache Beam Python job in Cloud Dataflow
A Kubeflow Pipeline component that submits an Apache Beam job (authored in Python) to Cloud Dataflow for execution. The Python Beam code is run with the Cloud Dataflow Runner.

## Intended Use
Use this component to run a Python Beam code to submit a Dataflow job as a step of a KFP pipeline. The component will wait until the job finishes.

## Runtime arguments
Name | Description | Type | Optional | Default
:--- | :---------- | :--- | :------- | :------
python_file_path |  The Cloud Storage or the local path to the python file being run. | String | No |
project_id |  The ID of the parent project of the Dataflow job. | GCPProjectID | No |
staging_dir | The Cloud Storage directory for keeping staging files. A random subdirectory will be created under the directory to keep job info for resuming the job in case of failure and it will be passed as `staging_location` and `temp_location` command line args of the beam code. | GCSPath | Yes | ` `
requirements_file_path |  The Cloud Storageor the local path to the pip requirements file. | String | Yes | ` `
args |  The list of arguments to pass to the python file. | List | Yes | `[]`
wait_interval |  The seconds to wait between calls to get the job status. | Integer | Yes | `30`

## Output:
Name | Description | Type
:--- | :---------- | :---
job_id | The id of the created dataflow job. | String

## Cautions and requirements
To use the components, the following requirements must be met:
* Dataflow API is enabled.
* The component is running under a secret [Kubeflow user service account](https://www.kubeflow.org/docs/started/getting-started-gke/#gcp-service-accounts) in a KFP cluster. For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
* The Kubeflow user service account is a member of `roles/dataflow.developer` role of the project.
* The Kubeflow user service account is a member of `roles/storage.objectViewer` role of the Cloud Storage Objects `python_file_path` and `requirements_file_path`.
* The Kubeflow user service account is a member of `roles/storage.objectCreator` role of the Cloud Storage Object `staging_dir`.

## Detailed description
Before using the component, make sure the following files are prepared in a Cloud Storage bucket.
* A Beam Python code file.
* A `requirements.txt` file which includes a list of dependent packages.

The Beam Python code should follow [Beam programing model](https://beam.apache.org/documentation/programming-guide/) and the following additional requirements to be compatible with this component:
* It accepts command line arguments: `--project`, `--temp_location`, `--staging_location`, which are [standard Dataflow Runner options](https://cloud.google.com/dataflow/docs/guides/specifying-exec-params#setting-other-cloud-pipeline-options).
* Enable info logging before the start of a Dataflow job in the Python code. This is important to allow the component to track the status and ID of create job. For example: calling `logging.getLogger().setLevel(logging.INFO)` before any other code.

The component does several things during the execution:
* Download `python_file_path` and `requirements_file_path` to local files.
* Start a subprocess to launch the Python program.
* Monitor the logs produced from the subprocess to extract Dataflow job information.
* Store Dataflow job information in `staging_dir` so the job can be resumed in case of failure.
* Wait for the job to finish.

Here are the steps to use the component in a pipeline:
1. Install KFP SDK



```python
%%capture --no-stderr

KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.14/kfp.tar.gz'
!pip3 install $KFP_PACKAGE --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataflow_python_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/d2f5cc92a46012b9927209e2aaccab70961582dc/components/gcp/dataflow/launch_python/component.yaml')
help(dataflow_python_op)
```

For more information about the component, please checkout:
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/component_sdk/python/kfp_component/google/dataflow/_launch_python.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataflow/launch_python/sample.ipynb)
* [Dataflow Python Quickstart](https://cloud.google.com/dataflow/docs/quickstarts/quickstart-python)

### Sample

Note: the sample code below works in both IPython notebook or python code directly.

In this sample, we run a wordcount sample code in a KFP pipeline. The output will be stored in a Cloud Storage bucket. Here is the sample code:


```python
!gsutil cat gs://ml-pipeline-playground/samples/dataflow/wc/wc.py
```

    #
    # Licensed to the Apache Software Foundation (ASF) under one or more
    # contributor license agreements.  See the NOTICE file distributed with
    # this work for additional information regarding copyright ownership.
    # The ASF licenses this file to You under the Apache License, Version 2.0
    # (the "License"); you may not use this file except in compliance with
    # the License.  You may obtain a copy of the License at
    #
    #    http://www.apache.org/licenses/LICENSE-2.0
    #
    # Unless required by applicable law or agreed to in writing, software
    # distributed under the License is distributed on an "AS IS" BASIS,
    # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    # See the License for the specific language governing permissions and
    # limitations under the License.
    #
    
    """A minimalist word-counting workflow that counts words in Shakespeare.
    
    This is the first in a series of successively more detailed 'word count'
    examples.
    
    Next, see the wordcount pipeline, then the wordcount_debugging pipeline, for
    more detailed examples that introduce additional concepts.
    
    Concepts:
    
    1. Reading data from text files
    2. Specifying 'inline' transforms
    3. Counting a PCollection
    4. Writing data to Cloud Storage as text files
    
    To execute this pipeline locally, first edit the code to specify the output
    location. Output location could be a local file path or an output prefix
    on GCS. (Only update the output location marked with the first CHANGE comment.)
    
    To execute this pipeline remotely, first edit the code to set your project ID,
    runner type, the staging location, the temp location, and the output location.
    The specified GCS bucket(s) must already exist. (Update all the places marked
    with a CHANGE comment.)
    
    Then, run the pipeline as described in the README. It will be deployed and run
    using the Google Cloud Dataflow Service. No args are required to run the
    pipeline. You can see the results in your output bucket in the GCS browser.
    """
    
    from __future__ import absolute_import
    
    import argparse
    import logging
    import re
    
    from past.builtins import unicode
    
    import apache_beam as beam
    from apache_beam.io import ReadFromText
    from apache_beam.io import WriteToText
    from apache_beam.options.pipeline_options import PipelineOptions
    from apache_beam.options.pipeline_options import SetupOptions
    
    
    def run(argv=None):
      """Main entry point; defines and runs the wordcount pipeline."""
    
      parser = argparse.ArgumentParser()
      parser.add_argument('--input',
                          dest='input',
                          default='gs://dataflow-samples/shakespeare/kinglear.txt',
                          help='Input file to process.')
      parser.add_argument('--output',
                          dest='output',
                          # CHANGE 1/5: The Google Cloud Storage path is required
                          # for outputting the results.
                          default='gs://YOUR_OUTPUT_BUCKET/AND_OUTPUT_PREFIX',
                          help='Output file to write results to.')
      known_args, pipeline_args = parser.parse_known_args(argv)
      # pipeline_args.extend([
      #     # CHANGE 2/5: (OPTIONAL) Change this to DataflowRunner to
      #     # run your pipeline on the Google Cloud Dataflow Service.
      #     '--runner=DirectRunner',
      #     # CHANGE 3/5: Your project ID is required in order to run your pipeline on
      #     # the Google Cloud Dataflow Service.
      #     '--project=SET_YOUR_PROJECT_ID_HERE',
      #     # CHANGE 4/5: Your Google Cloud Storage path is required for staging local
      #     # files.
      #     '--staging_location=gs://YOUR_BUCKET_NAME/AND_STAGING_DIRECTORY',
      #     # CHANGE 5/5: Your Google Cloud Storage path is required for temporary
      #     # files.
      #     '--temp_location=gs://YOUR_BUCKET_NAME/AND_TEMP_DIRECTORY',
      #     '--job_name=your-wordcount-job',
      # ])
    
      # We use the save_main_session option because one or more DoFn's in this
      # workflow rely on global context (e.g., a module imported at module level).
      pipeline_options = PipelineOptions(pipeline_args)
      pipeline_options.view_as(SetupOptions).save_main_session = True
      with beam.Pipeline(options=pipeline_options) as p:
    
        # Read the text file[pattern] into a PCollection.
        lines = p | ReadFromText(known_args.input)
    
        # Count the occurrences of each word.
        counts = (
            lines
            | 'Split' >> (beam.FlatMap(lambda x: re.findall(r'[A-Za-z\']+', x))
                          .with_output_types(unicode))
            | 'PairWithOne' >> beam.Map(lambda x: (x, 1))
            | 'GroupAndSum' >> beam.CombinePerKey(sum))
    
        # Format the counts into a PCollection of strings.
        def format_result(word_count):
          (word, count) = word_count
          return '%s: %s' % (word, count)
    
        output = counts | 'Format' >> beam.Map(format_result)
    
        # Write the output using a "Write" transform that has side effects.
        # pylint: disable=expression-not-assigned
        output | WriteToText(known_args.output)
    
    
    if __name__ == '__main__':
      logging.getLogger().setLevel(logging.INFO)
      run()


#### Set sample parameters


```python
# Required Parameters
PROJECT_ID = '<Please put your project ID here>'
GCS_STAGING_DIR = 'gs://<Please put your GCS path here>' # No ending slash
```


```python
# Optional Parameters
EXPERIMENT_NAME = 'Dataflow - Launch Python'
OUTPUT_FILE = '{}/wc/wordcount.out'.format(GCS_STAGING_DIR)
```

#### Example pipeline that uses the component


```python
import kfp.dsl as dsl
import kfp.gcp as gcp
import json
@dsl.pipeline(
    name='Dataflow launch python pipeline',
    description='Dataflow launch python pipeline'
)
def pipeline(
    python_file_path = 'gs://ml-pipeline-playground/samples/dataflow/wc/wc.py',
    project_id = PROJECT_ID,
    staging_dir = GCS_STAGING_DIR,
    requirements_file_path = 'gs://ml-pipeline-playground/samples/dataflow/wc/requirements.txt',
    args = json.dumps([
        '--output', OUTPUT_FILE
    ]),
    wait_interval = 30
):
    dataflow_python_op(
        python_file_path = python_file_path, 
        project_id = project_id, 
        staging_dir = staging_dir, 
        requirements_file_path = requirements_file_path, 
        args = args,
        wait_interval = wait_interval).apply(gcp.use_gcp_secret('user-gcp-sa'))
```

#### Compile the pipeline


```python
pipeline_func = pipeline
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

#### Inspect the output


```python
!gsutil cat $OUTPUT_FILE
```
