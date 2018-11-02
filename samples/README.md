# ML Pipeline Services - Authoring Guideline

## Setup
* Create a python3 envionronment. 

**Python 3.5 or above is required** and if you don't have Python3 set up, we suggest the following steps
to install [Miniconda](https://conda.io/miniconda.html).
 
In Debian/Ubuntu/[Cloud shell](https://console.cloud.google.com/cloudshell) environment:   
```bash
apt-get update; apt-get install -y wget bzip2
wget https://repo.continuum.io/miniconda/Miniconda3-latest-Linux-x86_64.sh
bash Miniconda3-latest-Linux-x86_64.sh
```
In Windows environment, download the [installer](https://repo.continuum.io/miniconda/Miniconda3-latest-Windows-x86_64.exe) and 
remember to select "*Add Miniconda to my PATH environment variable*" option during the installation.

In Mac environment, download the [installer](https://repo.continuum.io/miniconda/Miniconda3-latest-MacOSX-x86_64.sh) and
run the following command:

```bash
bash Miniconda3-latest-MacOSX-x86_64.sh
```

Then, create a clean python3 envionrment
 
```bash
conda create --name mlpipeline python=3.6
source activate mlpipeline
```
 
If `conda` command is not found, be sure to add the Miniconda path:
 
```bash
export PATH=MINICONDA_PATH/bin:$PATH

```

* Clone the repo. 

* Install DSL library and DSL compiler
 
```bash
pip install https://storage.googleapis.com/ml-pipeline/release/0.0.26/kfp-0.0.26.tar.gz --upgrade
 ```
After successful installation the command "dsl-compile" should be added to your PATH.

## Compile the samples
The sample pipelines are represented as Python code. To run these samples, you need to compile them, and then upload the output to the Pipeline system from web UI. 
<!--- 
In the future, we will build the compiler into the pipeline system such that these python files are immediately deployable.
--->

```bash
dsl-compile --py [path/to/py/file] --output [path/to/output/tar.gz]
```

For example:

```bash
dsl-compile --py [ML_REPO_DIRECTORY]/samples/basic/sequential.py --output [ML_REPO_DIRECTORY]/samples/basic/sequential.tar.gz
```

## Deploy the samples
Upload the generated .tar.gz file through the ML pipeline UI.

## Optional for advanced users: Building Your Own Components

### Requirement
Install [docker](https://www.docker.com/get-docker).

### Step One: Create A Container For Each Component
In most cases, you need to create your own container image that includes your program. You can find container 
building examples from [here](https://github.com/googleprivate/ml/blob/master/components)(in the directory, go to any subdirectory and then go to “containers” directory).

If your component creates some outputs to be fed as inputs to the downstream components, each output has 
to be a string and needs to be written to a separate local text file by the container image. 
For example, if a trainer component needs to output the trained model path, it writes the path into a 
local file “/output.txt”. In the python class (in step three), you have the chance to specify how to map the content 
of local files to component outputs.

<!---[TODO]: Add how to produce UI metadata.--->

### Step Two: Create A Python Class For Your Component
The python classes describe the interactions with the docker container image created in step one. 
For example, a component to create confusion matrix data from prediction results is like:

```python
class ConfusionMatrixOp(kfp.dsl.ContainerOp):

  def __init__(self, name, predictions, output_path):
    super(ConfusionMatrixOp, self).__init__(
      name=name,
      image='gcr.io/project-id/ml-pipeline-local-confusion-matrix:v1',
      command=['python', '/ml/confusion_matrix.py'],
      arguments=[
        '--output', '%s/{{workflow.name}}/confusionmatrix' % output_path,
        '--predictions', predictions
     ],
     file_outputs={'label': '/output.txt'})

```

Note:
* Each component needs to inherit from kfp.dsl.ContainerOp.
* If you already defined ENTRYPOINT in the container image, you don’t have to provide “command” unless you want to override it.
* In the init arguments, there can be python native types (such as str, int) and “kfp.dsl.PipelineParam” 
types. Each kfp.dsl.PipelineParam represents a parameter whose value is usually only known at run time. It might be a pipeline 
parameter whose value is provided at pipeline run time by user, or can be an output from an upstream component. 
In the above case, “predictions” and “output_path” are kfp.dsl.PipelineParams.
* Although value of each PipelineParam is only available at runtime, you can still use them inline the 
argument (note the “%s”). It means at run time the argument will contain the value of the param inline.
* “File_outputs” lists a map between labels and local file paths. In the above case, the content of '/output.txt' is gathered as a string output of the operator. To reference the output in code:

```python
op = ConfusionMatrixOp(...)
op.outputs['label']
```

If there is only one output then you can also do “op.output”.


### Step Three: Create Your Workflow as a python function
Each pipeline is identified as a python function. For example:

```python
@kfp.dsl.pipeline(
  name='TFMA Trainer',
  description='A trainer that does end-to-end training for TFMA models.'
)
def train(
    output_path,
    train_data=kfp.dsl.PipelineParam('train-data',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv'),
    eval_data=kfp.dsl.PipelineParam('eval-data',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv'),
    schema=kfp.dsl.PipelineParam('schema',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json'),
    target=kfp.dsl.PipelineParam('target', value='tips'),
    learning_rate=kfp.dsl.PipelineParam('learning-rate', value=0.1),
    hidden_layer_size=kfp.dsl.PipelineParam('hidden-layer-size', value='100,50'),
    steps=kfp.dsl.PipelineParam('steps', value=1000),
    slice_columns=kfp.dsl.PipelineParam('slice-columns', value='trip_start_hour'),
    true_class=kfp.dsl.PipelineParam('true-class', value='true'),
    need_analysis=kfp.dsl.PipelineParam('need-analysis', value='true'),
)
```

Note:

* **@kfp.dsl.pipeline** is a required decoration including “name” and "description" properties.
* Input arguments will show up as pipeline parameters in the Pipeline system web UI. As a python rule, positional 
args go first and keyword args go next.
* Each function argument is of type kfp.dsl.PipelineParam. The default values 
should all be of that type. The default values will show up in the Pipeline UI but can be overwritten.


See an example [here](https://github.com/googleprivate/ml/blob/master/samples/xgboost-spark/xgboost-training-cm.py).
