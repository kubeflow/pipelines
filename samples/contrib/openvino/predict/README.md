# OpenVINO predict pipeline

This is an example of simple one step pipeline implementation including OpenVINO predict component.

It can execute predict operation for a dataset in numpy format and provided model in Intermediate Representation format.

This format of models can be generated based on trained model from various frameworks like TensorFlow, Caffe, MXNET and Kaldi. 

Dataset in the numpy file needs to match the shape of the provided model input. 

This pipeline execute the predict operation and sends the results for each model output a numpy file with the name 
representing the output tensor name. 

*Note:* Executing this pipeline required building the docker image according to the guidelines on 
[OpenVINO predict component doc](../../../contrib/components/openvino/predict). 
The image name pushed to the docker registry should be configured in the pipeline script `numpy_predict` 

## Examples of the parameters

model_bin - gs://<path>/model.bin 

model_xml - gs://<path>/model.xml 

input_numpy_file - gs://<path>/datasets/imgs.npy 

output_folder - gs://<path>/outputs

