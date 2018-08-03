The sample pipelines are represented as Python3 functions (DSL code). To run these samples, for now you need to compile them into workflow yamls and then upload to the pipeline system. In the future, we will build the compiler into the pipeline system so these Python3 functions are directly deployable.

To compile these workflow yamls:

1. Make sure you have Python3 set up.
2. Clone the repo.
3. Run `pip3 install ./dsl/. --upgrade`. This is the library used to represent pipelines with Python code.
4. Run `pip3 install ./dsl-compiler/. --upgrade`. This is the compiler to compile DSL code into workflow yaml.
5. Run `dsl-compile --py [path/to/py/file] --output [path/to/output/yaml]`. For example: `dsl-compile --py ./samples/basic/sequential.py --output /tmp/sequential.yaml`.
6. Then you can upload the generated yaml in ML Pipelines system to run.

The directory also includes a compiled pipeline (sequential.yaml). The only parameter it expects is a GS URL to a text file. For example, you can use gs://bradley-playground/shakespeare1.txt.
