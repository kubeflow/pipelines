# Inference component with OpenVINO inference engine

This component takes the following parameters:
* path to the model in Intermediate Representation format ( xml and bin files)
* numpy file with the input dataset. Input shape should fit to the used model definition.
* path to the folder where the inference results in numpy format should be uploaded

In the component logs are included inference performance details.

It is a generic component which can be used to process arbitrary data and any OpenVINO model.
It can be also considered as an example how to create more customized version.

```bash
python3 predict.py --help
usage: predict.py [-h] [--model_bin MODEL_BIN] [--model_xml MODEL_XML]
                  [--input_numpy_file INPUT_NUMPY_FILE]
                  [--output_folder OUTPUT_FOLDER]

Component executing inference operation

optional arguments:
  -h, --help            show this help message and exit
  --model_bin MODEL_BIN
                        GCS or local path to model weights file (.bin)
  --model_xml MODEL_XML
                        GCS or local path to model graph (.xml)
  --input_numpy_file INPUT_NUMPY_FILE
                        GCS or local path to input dataset numpy file
  --output_folder OUTPUT_FOLDER
                        GCS or local path to results upload folder
```


## building docker image


```bash
docker build --build-arg http_proxy=$http_proxy --build-arg https_proxy=$https_proxy .
```

## testing the image locally

```bash
COMMAND = python3 predict.py \
--model_bin gs://<path>/model.bin \
--model_xml gs://<path>/model.xml \
--input_numpy_file gs://<path>/datasets/imgs.npy \
--output_folder gs://<path>/outputs
docker run --rm -it -e GOOGLE_APPLICATION_CREDENTIALS=/etc/credentials/gcp_key.json \
-v ${PWD}/key.json:/etc/credentials/gcp_key.json <image name> $COMMAND
```