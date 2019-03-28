# Model optimization component

This component is executing model optimization process using OpenVINO Toolkit and generate as output the model in 
Intermediate Representation format.

Component takes the following arguments:
* model input GCS path
* model optimizer parameters
* model output GCS path

```bash
usage: convert_model.py [-h] [--input_path INPUT_PATH]
                        [--mo_options MO_OPTIONS] [--output_path OUTPUT_PATH]

Model converter to OpenVINO Intermediate Representation format

optional arguments:
  -h, --help            show this help message and exit
  --input_path INPUT_PATH
                        GCS path of input model file or folder
  --mo_options MO_OPTIONS
                        OpenVINO Model Optimizer options
  --output_path OUTPUT_PATH
                        GCS path of output folder
                        
```

## Component parameters

It takes as input GCS path to the input model in any of the OpenVINO supported frameworks:
* Tensorflow
* Caffe
* MXNET
* Kaldi
* ONNX

Input model path can be a folder or an individual file which will be copied to a component working directory

Model optimizer options can include any of the parameters supported by OpenVINO toolkit model optimizer.

Refer to OpenVINO [documentation](https://software.intel.com/en-us/articles/OpenVINO-ModelOptimizer) for details. 
```bash
mo.py --help
usage: mo.py [-h] [--framework {tf,caffe,mxnet,kaldi,onnx}]
             [--input_model INPUT_MODEL] [--model_name MODEL_NAME]
             [--output_dir OUTPUT_DIR] [--input_shape INPUT_SHAPE]
             [--scale SCALE] [--reverse_input_channels]
             [--log_level {CRITICAL,ERROR,WARN,WARNING,INFO,DEBUG,NOTSET}]
             [--input INPUT] [--output OUTPUT] [--mean_values MEAN_VALUES]
             [--scale_values SCALE_VALUES]
             [--data_type {FP16,FP32,half,float}] [--disable_fusing]
             [--disable_resnet_optimization]
             [--finegrain_fusing FINEGRAIN_FUSING] [--disable_gfusing]
             [--move_to_preprocess] [--extensions EXTENSIONS] [--batch BATCH]
             [--version] [--silent]
             [--freeze_placeholder_with_value FREEZE_PLACEHOLDER_WITH_VALUE]
             [--generate_deprecated_IR_V2] [--input_model_is_text]
             [--input_checkpoint INPUT_CHECKPOINT]
             [--input_meta_graph INPUT_META_GRAPH]
             [--saved_model_dir SAVED_MODEL_DIR]
             [--saved_model_tags SAVED_MODEL_TAGS]
             [--offload_unsupported_operations_to_tf]
             [--tensorflow_subgraph_patterns TENSORFLOW_SUBGRAPH_PATTERNS]
             [--tensorflow_operation_patterns TENSORFLOW_OPERATION_PATTERNS]
             [--tensorflow_custom_operations_config_update TENSORFLOW_CUSTOM_OPERATIONS_CONFIG_UPDATE]
             [--tensorflow_use_custom_operations_config TENSORFLOW_USE_CUSTOM_OPERATIONS_CONFIG]
             [--tensorflow_object_detection_api_pipeline_config TENSORFLOW_OBJECT_DETECTION_API_PIPELINE_CONFIG]
             [--tensorboard_logdir TENSORBOARD_LOGDIR]
             [--tensorflow_custom_layer_libraries TENSORFLOW_CUSTOM_LAYER_LIBRARIES]
             [--disable_nhwc_to_nchw] [--input_proto INPUT_PROTO] [-k K]
             [--mean_file MEAN_FILE] [--mean_file_offsets MEAN_FILE_OFFSETS]
             [--disable_omitting_optional] [--enable_flattening_nested_params]
             [--input_symbol INPUT_SYMBOL] [--nd_prefix_name ND_PREFIX_NAME]
             [--pretrained_model_name PRETRAINED_MODEL_NAME]
             [--save_params_from_nd] [--legacy_mxnet_model] [--counts COUNTS]
             [--remove_output_softmax]

optional arguments:
  -h, --help            show this help message and exit
  --framework {tf,caffe,mxnet,kaldi,onnx}
                        Name of the framework used to train the input model.

Framework-agnostic parameters:
  --input_model INPUT_MODEL, -w INPUT_MODEL, -m INPUT_MODEL
                        Tensorflow*: a file with a pre-trained model (binary
                        or text .pb file after freezing). Caffe*: a model
                        proto file with model weights
  --model_name MODEL_NAME, -n MODEL_NAME
                        Model_name parameter passed to the final create_ir
                        transform. This parameter is used to name a network in
                        a generated IR and output .xml/.bin files.
  --output_dir OUTPUT_DIR, -o OUTPUT_DIR
                        Directory that stores the generated IR. By default, it
                        is the directory from where the Model Optimizer is
                        launched.
  --input_shape INPUT_SHAPE
                        Input shape(s) that should be fed to an input node(s)
                        of the model. Shape is defined as a comma-separated
                        list of integer numbers enclosed in parentheses or
                        square brackets, for example [1,3,227,227] or
                        (1,227,227,3), where the order of dimensions depends
                        on the framework input layout of the model. For
                        example, [N,C,H,W] is used for Caffe* models and
                        [N,H,W,C] for TensorFlow* models. Model Optimizer
                        performs necessary transformations to convert the
                        shape to the layout required by Inference Engine
                        (N,C,H,W). The shape should not contain undefined
                        dimensions (? or -1) and should fit the dimensions
                        defined in the input operation of the graph. If there
                        are multiple inputs in the model, --input_shape should
                        contain definition of shape for each input separated
                        by a comma, for example: [1,3,227,227],[2,4] for a
                        model with two inputs with 4D and 2D shapes.
  --scale SCALE, -s SCALE
                        All input values coming from original network inputs
                        will be divided by this value. When a list of inputs
                        is overridden by the --input parameter, this scale is
                        not applied for any input that does not match with the
                        original input of the model.
  --reverse_input_channels
                        Switch the input channels order from RGB to BGR (or
                        vice versa). Applied to original inputs of the model
                        if and only if a number of channels equals 3. Applied
                        after application of --mean_values and --scale_values
                        options, so numbers in --mean_values and
                        --scale_values go in the order of channels used in the
                        original model.
  --log_level {CRITICAL,ERROR,WARN,WARNING,INFO,DEBUG,NOTSET}
                        Logger level
  --input INPUT         The name of the input operation of the given model.
                        Usually this is a name of the input placeholder of the
                        model.
  --output OUTPUT       The name of the output operation of the model. For
                        TensorFlow*, do not add :0 to this name.
  --mean_values MEAN_VALUES, -ms MEAN_VALUES
                        Mean values to be used for the input image per
                        channel. Values to be provided in the (R,G,B) or
                        [R,G,B] format. Can be defined for desired input of
                        the model, for example: "--mean_values
                        data[255,255,255],info[255,255,255]". The exact
                        meaning and order of channels depend on how the
                        original model was trained.
  --scale_values SCALE_VALUES
                        Scale values to be used for the input image per
                        channel. Values are provided in the (R,G,B) or [R,G,B]
                        format. Can be defined for desired input of the model,
                        for example: "--scale_values
                        data[255,255,255],info[255,255,255]". The exact
                        meaning and order of channels depend on how the
                        original model was trained.
  --data_type {FP16,FP32,half,float}
                        Data type for all intermediate tensors and weights. If
                        original model is in FP32 and --data_type=FP16 is
                        specified, all model weights and biases are quantized
                        to FP16.
  --disable_fusing      Turn off fusing of linear operations to Convolution
  --disable_resnet_optimization
                        Turn off resnet optimization
  --finegrain_fusing FINEGRAIN_FUSING
                        Regex for layers/operations that won't be fused.
                        Example: --finegrain_fusing Convolution1,.*Scale.*
  --disable_gfusing     Turn off fusing of grouped convolutions
  --move_to_preprocess  Move mean values to IR preprocess section
  --extensions EXTENSIONS
                        Directory or a comma separated list of directories
                        with extensions. To disable all extensions including
                        those that are placed at the default location, pass an
                        empty string.
  --batch BATCH, -b BATCH
                        Input batch size
  --version             Version of Model Optimizer
  --silent              Prevent any output messages except those that
                        correspond to log level equals ERROR, that can be set
                        with the following option: --log_level. By default,
                        log level is already ERROR.
  --freeze_placeholder_with_value FREEZE_PLACEHOLDER_WITH_VALUE
                        Replaces input layer with constant node with provided
                        value, e.g.: "node_name->True"
  --generate_deprecated_IR_V2
                        Force to generate legacy/deprecated IR V2 to work with
                        previous versions of the Inference Engine. The
                        resulting IR may or may not be correctly loaded by
                        Inference Engine API (including the most recent and
                        old versions of Inference Engine) and provided as a
                        partially-validated backup option for specific
                        deployment scenarios. Use it at your own discretion.
                        By default, without this option, the Model Optimizer
                        generates IR V3.

TensorFlow*-specific parameters:
  --input_model_is_text
                        TensorFlow*: treat the input model file as a text
                        protobuf format. If not specified, the Model Optimizer
                        treats it as a binary file by default.
  --input_checkpoint INPUT_CHECKPOINT
                        TensorFlow*: variables file to load.
  --input_meta_graph INPUT_META_GRAPH
                        Tensorflow*: a file with a meta-graph of the model
                        before freezing
  --saved_model_dir SAVED_MODEL_DIR
                        TensorFlow*: directory representing non frozen model
  --saved_model_tags SAVED_MODEL_TAGS
                        Group of tag(s) of the MetaGraphDef to load, in string
                        format, separated by ','. For tag-set contains
                        multiple tags, all tags must be passed in.
  --offload_unsupported_operations_to_tf
                        TensorFlow*: automatically offload unsupported
                        operations to TensorFlow*
  --tensorflow_subgraph_patterns TENSORFLOW_SUBGRAPH_PATTERNS
                        TensorFlow*: a list of comma separated patterns that
                        will be applied to TensorFlow* node names to infer a
                        part of the graph using TensorFlow*.
  --tensorflow_operation_patterns TENSORFLOW_OPERATION_PATTERNS
                        TensorFlow*: a list of comma separated patterns that
                        will be applied to TensorFlow* node type (ops) to
                        infer these operations using TensorFlow*.
  --tensorflow_custom_operations_config_update TENSORFLOW_CUSTOM_OPERATIONS_CONFIG_UPDATE
                        TensorFlow*: update the configuration file with node
                        name patterns with input/output nodes information.
  --tensorflow_use_custom_operations_config TENSORFLOW_USE_CUSTOM_OPERATIONS_CONFIG
                        TensorFlow*: use the configuration file with custom
                        operation description.
  --tensorflow_object_detection_api_pipeline_config TENSORFLOW_OBJECT_DETECTION_API_PIPELINE_CONFIG
                        TensorFlow*: path to the pipeline configuration file
                        used to generate model created with help of Object
                        Detection API.
  --tensorboard_logdir TENSORBOARD_LOGDIR
                        TensorFlow*: dump the input graph to a given directory
                        that should be used with TensorBoard.
  --tensorflow_custom_layer_libraries TENSORFLOW_CUSTOM_LAYER_LIBRARIES
                        TensorFlow*: comma separated list of shared libraries
                        with TensorFlow* custom operations implementation.
  --disable_nhwc_to_nchw
                        Disables default translation from NHWC to NCHW

Caffe*-specific parameters:
  --input_proto INPUT_PROTO, -d INPUT_PROTO
                        Deploy-ready prototxt file that contains a topology
                        structure and layer attributes
  -k K                  Path to CustomLayersMapping.xml to register custom
                        layers
  --mean_file MEAN_FILE, -mf MEAN_FILE
                        Mean image to be used for the input. Should be a
                        binaryproto file
  --mean_file_offsets MEAN_FILE_OFFSETS, -mo MEAN_FILE_OFFSETS
                        Mean image offsets to be used for the input
                        binaryproto file. When the mean image is bigger than
                        the expected input, it is cropped. By default, centers
                        of the input image and the mean image are the same and
                        the mean image is cropped by dimensions of the input
                        image. The format to pass this option is the
                        following: "-mo (x,y)". In this case, the mean file is
                        cropped by dimensions of the input image with offset
                        (x,y) from the upper left corner of the mean image
  --disable_omitting_optional
                        Disable omitting optional attributes to be used for
                        custom layers. Use this option if you want to transfer
                        all attributes of a custom layer to IR. Default
                        behavior is to transfer the attributes with default
                        values and the attributes defined by the user to IR.
  --enable_flattening_nested_params
                        Enable flattening optional params to be used for
                        custom layers. Use this option if you want to transfer
                        attributes of a custom layer to IR with flattened
                        nested parameters. Default behavior is to transfer the
                        attributes without flattening nested parameters.

Mxnet-specific parameters:
  --input_symbol INPUT_SYMBOL
                        Symbol file (for example, model-symbol.json) that
                        contains a topology structure and layer attributes
  --nd_prefix_name ND_PREFIX_NAME
                        Prefix name for args.nd and argx.nd files.
  --pretrained_model_name PRETRAINED_MODEL_NAME
                        Name of a pretrained MXNet model without extension and
                        epoch number. This model will be merged with args.nd
                        and argx.nd files
  --save_params_from_nd
                        Enable saving built parameters file from .nd files
  --legacy_mxnet_model  Enable MXNet loader to make a model compatible with
                        the latest MXNet version. Use only if your model was
                        trained with MXNet version lower than 1.0.0

Kaldi-specific parameters:
  --counts COUNTS       Path to the counts file
  --remove_output_softmax
                        Removes the SoftMax layer that is the output layer
```

The output folder specify then should be uploaded the generated model file in IR format with .bin and .xml
extensions.

The component also creates 3 files including the paths to generated model:
- `/tmp/output.txt` - GSC path to the folder including the generated model files.
- `/tmp/bin_path.txt` -  GSC path to weights model file 
- `/tmp/xml_path.txt` - GSC path to graph model file 
They can be used as parameters to be passed to other jobs in ML pipelines.

## Examples

Input path - gs://tensorflow_model_path/resnet/1/saved_model.pb<br />
MO options - --saved_model_dir .<br />
Output path - gs://tensorflow_model_path/resnet/1

Input path - gs://tensorflow_model_path/resnet/1<br />
MO options - --saved_model_dir 1<br />
Output path - gs://tensorflow_model_path/resnet/dldt/1<br />


## Building docker image

```bash
docker build --build-arg http_proxy=$http_proxy --build-arg https_proxy=$https_proxy .
```

## Starting and testing the component locally 

This component requires GCP authentication token in json format generated for the service account,
which has access to GCS location. In the example below it is in key.json in the current path.

```bash
COMMAND="convert_model.py --mo_options  \"--saved_model_dir .\" --input_path gs://tensorflow_model_path/resnet/1/saved_model.pb --output_path gs://tensorflow_model_path/resnet/1"
docker run --rm -it -v $(pwd)/key.json:/etc/credentials/gcp-key.json \
-e GOOGLE_APPLICATION_CREDENTIALS=/etc/credentials/gcp-key.json <image_name> $COMMAND

```

