# Slim models generator

This component is automating implementation of [slim models](https://github.com/tensorflow/models/blob/master/research/slim).
It can create a graph from slim models zoo, load the variables pre-trained checkpoint and export the model in the form 
of Tensorflow `frozen graph` and `saved model`.

The results of the component can be saved in a local path or in GCS cloud storage. The can be used to other ML pipeline
components like OpenVINO model optimizer, OpenVINO predict or OpenVINO Model Server.

## Building

```bash
docker build --build-arg http_proxy=$http_proxy --build-arg https_proxy=$https_proxy .
```


## Using the component

```bash
python slim_model.py --help
usage: slim_model.py [-h] [--model_name MODEL_NAME] [--export_dir EXPORT_DIR]
                     [--batch_size BATCH_SIZE]
                     [--checkpoint_url CHECKPOINT_URL]
                     [--num_classes NUM_CLASSES]

Slim model generator

optional arguments:
  -h, --help            show this help message and exit
  --model_name MODEL_NAME
  --export_dir EXPORT_DIR
                        GCS or local path to save the generated model
  --batch_size BATCH_SIZE
                        batch size to be used in the exported model
  --checkpoint_url CHECKPOINT_URL
                        URL to the pretrained compressed checkpoint
  --num_classes NUM_CLASSES
                        number of model classes
```

*Model name* can be any model defined in the slim repository. The naming convention needs to match the key name from 
[net_factory.py]()https://github.com/tensorflow/models/blob/master/research/slim/nets/nets_factory.py#L39)

*export dir* can be a local path in the container or it might be GCS path to store generated files:
- model graph file in pb format
- frozen graph including weights from the provided checkpoint
- event file which can be imported in tensorboard
- saved model which will be stored in subfolder called `1`.

*batch size* represent the batch used in the exported models. It can be natural number to represent fixed batch size
or `-1` value can be set for dynamic batch size.

*checkpoint_url* is the URL to a pre-trained checkpoint https://github.com/tensorflow/models/tree/master/research/slim#pre-trained-models
It must match the model specified in model_name parameter.

*num classes* should include model specific number of classes in the outputs. For slim models it should be a value 
of `1000` or `1001`. It must match the number of classes used in the requested model name.


## Examples

```
python slim_model.py --model_name mobilenet_v1_050 --export_dir /tmp/mobilnet 
--batch_size 1 --num_classes=1001 \
--checkpoint_url http://download.tensorflow.org/models/mobilenet_v1_2018_02_22/mobilenet_v1_0.5_160.tgz

python slim_model.py --model_name resnet_v1_50 --export_dir gs://<bucket_name>/resnet \
--batch_size -1 --num_classes=1000 \
--checkpoint_url http://download.tensorflow.org/models/resnet_v1_50_2016_08_28.tar.gz

python slim_model.py --model_name inception_v4 --export_dir gs://<bucket_name>/inception \
--batch_size -1 --num_classes=1001 \
--checkpoint_url http://download.tensorflow.org/models/inception_v4_2016_09_09.tar.gz

python slim_model.py --model_name vgg_19 --export_dir /tmp/vgg \
--batch_size 1 --num_classes=1000 \
--checkpoint_url http://download.tensorflow.org/models/vgg_19_2016_08_28.tar.gz
```

