
# Name
Data preparation by executing an Apache Beam job in Cloud Dataflow

# Labels
GCP, Cloud Dataflow, Apache Beam, Python, Kubeflow

# Summary
A Kubeflow Pipeline component that prepares data by submitting an Apache Beam job (authored in Python) to Cloud Dataflow for execution. The Python Beam code is run with Cloud Dataflow Runner.

# Details
## Intended use

Use this component to run a Python Beam code to submit a Cloud Dataflow job as a step of a Kubeflow pipeline. 

## Runtime arguments
Name | Description | Optional |  Data type| Accepted values | Default |
:--- | :----------| :----------| :----------| :----------| :---------- |
python_file_path |  The path to the Cloud Storage bucket or local directory containing the Python file to be run. |  |  GCSPath |  |  |
project_id |  The ID of the Google Cloud Platform (GCP) project  containing the Cloud Dataflow job.| | GCPProjectID | | |
staging_dir  |   The path to the Cloud Storage directory where the staging files are stored. A random subdirectory will be created under the staging directory to keep the  job information.This is done so that you can resume the job in case of failure. `staging_dir` is passed as the command line arguments (`staging_location` and `temp_location`) of the Beam code. |   Yes  |   GCSPath  |   |   None  |
requirements_file_path |   The path to the Cloud Storage bucket or local directory containing the pip requirements file. | Yes | GCSPath |  | None |
args |  The list of arguments to pass to the Python file. | No |  List | A list of string arguments | None |
wait_interval |  The number of seconds to wait between calls to get the status of the job. | Yes | Integer  |  | 30 |

## Input data schema

Before you use the component, the following files must be ready in a Cloud Storage bucket:
- A Beam Python code file.
- A  `requirements.txt` file which includes a list of dependent packages.

The Beam Python code should follow the [Beam programming guide](https://beam.apache.org/documentation/programming-guide/) as well as the following additional requirements to be compatible with this component:
- It accepts the command line arguments `--project`, `--temp_location`, `--staging_location`, which are [standard Dataflow Runner options](https://cloud.google.com/dataflow/docs/guides/specifying-exec-params#setting-other-cloud-pipeline-options).
- It enables `info logging` before the start of a Cloud Dataflow job in the Python code. This is important to allow the component to track the status and ID of the job that is created. For example, calling `logging.getLogger().setLevel(logging.INFO)` before any other code.


## Output
Name | Description
:--- | :----------
job_id | The id of the Cloud Dataflow job that is created.

## Cautions & requirements
To use the components, the following requirements must be met:
- Cloud Dataflow API is enabled.
- The component is running under a secret Kubeflow user service account in a Kubeflow Pipeline cluster.  For example:
```
component_op(...).apply(gcp.use_gcp_secret('user-gcp-sa'))
```
The Kubeflow user service account is a member of:
- `roles/dataflow.developer` role of the project.
- `roles/storage.objectViewer` role of the Cloud Storage Objects `python_file_path` and `requirements_file_path`.
- `roles/storage.objectCreator` role of the Cloud Storage Object `staging_dir`. 

## Detailed description
The component does several things during the execution:
- Downloads `python_file_path` and `requirements_file_path` to local files.
- Starts a subprocess to launch the Python program.
- Monitors the logs produced from the subprocess to extract the Cloud Dataflow job information.
- Stores the Cloud Dataflow job information in `staging_dir` so the job can be resumed in case of failure.
- Waits for the job to finish.
The steps to use the component in a pipeline are:
1. Install the Kubeflow Pipelines SDK:



```python
%%capture --no-stderr

KFP_PACKAGE = 'https://storage.googleapis.com/ml-pipeline/release/0.1.14/kfp.tar.gz'
!pip3 install $KFP_PACKAGE --upgrade
```

2. Load the component using KFP SDK


```python
import kfp.components as comp

dataflow_python_op = comp.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/48dd338c8ab328084633c51704cda77db79ac8c2/components/gcp/dataflow/launch_python/component.yaml')
help(dataflow_python_op)
```

### Sample
Note: The following sample code works in an IPython notebook or directly in Python code. See the sample code below to learn how to execute the template.
In this sample, we run a wordcount sample code in a Kubeflow Pipeline. The output will be stored in a Cloud Storage bucket. Here is the sample code:


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

## References
* [Component python code](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/component_sdk/python/kfp_component/google/dataflow/_launch_python.py)
* [Component docker file](https://github.com/kubeflow/pipelines/blob/master/components/gcp/container/Dockerfile)
* [Sample notebook](https://github.com/kubeflow/pipelines/blob/master/components/gcp/dataflow/launch_python/sample.ipynb)
* [Dataflow Python Quickstart](https://cloud.google.com/dataflow/docs/quickstarts/quickstart-python)

## License
By deploying or using this software you agree to comply with the [AI Hub Terms of Service](https://aihub.cloud.google.com/u/0/aihub-tos) and the [Google APIs Terms of Service](https://developers.google.com/terms/). To the extent of a direct conflict of terms, the AI Hub Terms of Service will control.
