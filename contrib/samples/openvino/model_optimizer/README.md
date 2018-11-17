# OpenVINO model optimizer pipeline

This is an example of a one step pipeline implementation of model optimization using OpenVINO toolkit

It performs graph optimization and generates Intermediate Representation model format which can be used 
later by the Inference Engine. 

Learn more about [OpenVINO model optimizer](https://software.intel.com/en-us/articles/OpenVINO-ModelOptimizer)

*Note:* Executing this pipeline required building the docker image according to the guidelines on 
[OpenVINO model converted doc](../../../contrib/components/openvino/model_convert). 
The image name pushed to the docker registry should be configured in the pipeline script `convert_model_pipeline.py` 

## Examples of the parameters
 
input_path - gs://tensorflow_model_path/resnet/1/saved_model.pb

mo_options - --saved_model_dir .

output_path - gs://tensorflow_model_path/resnet/1


All parameters for model optimizer options are described in the component
 [doc](../../../components/model_convert/README.md)
 
The model conversion component is copying the content of the input path to the current directory in the container.
It can include a single file or the complete folder. In the model optimizer options you should reference 
the the file using relative path from the input path folder. This way you could pass also any configuration file
needed by the model optimizer.
