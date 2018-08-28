## Setup
* Create a python3 envionrment.
 
* Clone the repo. 
Download the latest [release](https://github.com/googleprivate/ml/releases) and unarchive it.
 
* Install mlp library and DSL compiler
 
```bash
cd ML_REPO_DIRECTORY
pip install ./dsl/ --upgrade # This is the library used to represent pipelines with Python code.
pip install ./dsl-compiler/ --upgrade # This is the compiler to compile DSL codes into workflow yaml.
 ```
After successful installation "dsl-compile" should be added to your PATH.

## Compile the samples
The sample pipelines are represented as python code. To run these samples, one needs to compile them into 
workflow yamls and then upload to the pipeline web console. 
<!--- 
In the future, we will build the compiler into the pipeline system such that these python files are immediately deployable.
--->

```bash
dsl-compile --py [path/to/py/file] --output [path/to/output/yaml]
```

For example:

```bash
dsl-compile --py ./samples/basic/sequential.py --output ~/Desktop/sequential.yaml
```

## Deploy the samples
Upload the generated yaml file through the ML pipeline web console. Here is the 
[instructions](https://github.com/googleprivate/ml/blob/master/README.md) to set up Pipelines System.

## Optional for advanced users: design customized DSL

### Requiremnt:
Install [docker](https://www.docker.com/get-docker).

### Step One: Create A Container For Each Component
In most cases, you need to create your own container image that includes your program. You can find container 
building examples from [here](https://github.com/googleprivate/ml/blob/master/components)(in the directory, go to any subdirectory and then go to “containers” directory).

If your component creates some outputs to be feeded as inputs to the downstream components, each output has 
to be a string and needs to be written to a separate local text file by the container image. 
For example, if a trainer component needs to output the trained model path, it writes the path into a 
local file “/output.txt”. In the python class (in step three), you have the chance to indicate how to map the contents 
of local files to component outputs.

<!---[TODO]: Add how to produce UI metadata.--->

### Step Two: Create A Python Class For Your Component
The python classes describe the interactions with the docker container image created in step one. 
For example, a component to create confusion matrix data from prediction results is like:

```python
class ConfusionMatrixOp(mlp.ContainerOp):

  def __init__(self, name, predictions, output_path):
    super(ConfusionMatrixOp, self).__init__(
      name=name,
      image='gcr.io/project-id/ml-pipeline-local-confusion-matrix:v1',
      command=['python', '/ml/confusion_matrix.py'],
      arguments=[
        '--output', '%s/{{workflow.name}}/confusionmatrix' % output_path,
        '--predictions', predictions
     ],
     file_outputs={'output': '/output.txt'})

```

Note:
* Each component needs to inherit from mlp.ContainerOp.
* If you already defined ENTRYPOINT in the container image, you don’t have to provide “command” unless you want to override it.
* In the init arguments, there can be python native types (such as str, int), and there can be “mlp.PipelineParam” 
types. Each mlp.PipelineParam represents a parameter whose value is only known at run time. It might be a pipeline 
parameter whose value is provided at pipeline run time, or it can be an output from an upstream component. 
In the above case, “predictions” and “output_path” are mlp.PipelineParams.
* Although value of each PipelineParam is only available at runtime, you can still use them inline the 
argument (note the “%s”). It means at run time the argument will contain the value of the param inline.
“File_outputs” lists a map between labels and local file paths. The outputs of the components will be referenced such as:  

```python
op = ConfusionMatrixOp(...)
op.outputs['label']
```

If there is only one output then you can do “op.output”.

* Inputs have to be inline arguments in entrypoints.
* Outputs have to be saved in local text files. Can be multiple. The file paths need to be in sync with DSL code.
* UI metadata needs to go to a GCS location by contention.

<!---[TODO]: Add Sample Link--->

### Step Three: Create Your Workflow as a python function
Each pipeline is identified as a python function. For example:

```python
@mlp.pipeline(
  name='TFMA Trainer',
  description='A trainer that does end-to-end training for TFMA models.'
)
def train(
    output_path,
    train_data=mlp.PipelineParam('train-data',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv'),
    eval_data=mlp.PipelineParam('eval-data',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv'),
    schema=mlp.PipelineParam('schema',
        value='gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json'),
    target=mlp.PipelineParam('target', value='tips'),
    learning_rate=mlp.PipelineParam('learning-rate', value=0.1),
    hidden_layer_size=mlp.PipelineParam('hidden-layer-size', value='100,50'),
    steps=mlp.PipelineParam('steps', value=1000),
    slice_columns=mlp.PipelineParam('slice-columns', value='trip_start_hour'),
    true_class=mlp.PipelineParam('true-class', value='true'),
    need_analysis=mlp.PipelineParam('need-analysis', value='true'),
)
```

Note:

* **@mlp.pipeline** is a required decoration including “name” and "description" properties.
* Input arguments will show up as pipeline parameters in Pipeline UI. As a python rule, positional 
args go first and keyword args go next.
* Each function argument is internally interpreted as mlp.PipelineParam type. The defaults values 
should all be of that type. The default values will show up in the Pipeline UI.

We recommend starting from simple examples. Note that in the [samples](https://github.com/googleprivate/ml/blob/master/samples), Step two (Create A Python Class For Your Component) is skipped. 
The pipeline creates ops from mlp.ContainerOp directly. This works if you don’t expect others to reuse your component. 
Otherwise, it is better to create python classes for a more friendly component interface.

<!---[TODO: Add a link to a real world example]--->
