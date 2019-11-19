#!/usr/bin/env python
import grpc
import numpy as np
import tensorflow.contrib.util as tf_contrib_util
import datetime
import argparse
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc
from urllib.parse import urlparse
import requests
import cv2
import os
import json
import classes


def crop_resize(img,cropx,cropy):
    y,x,c = img.shape
    if y < cropy:
        img = cv2.resize(img, (x, cropy))
        y = cropy
    if x < cropx:
        img = cv2.resize(img, (cropx,y))
        x = cropx
    startx = x//2-(cropx//2)
    starty = y//2-(cropy//2)
    return img[starty:starty+cropy,startx:startx+cropx,:]

def get_file_content(source_path):
    parsed_path = urlparse(source_path)
    if parsed_path.scheme == "http" or parsed_path.scheme == "https":
        try:
            response = requests.get(source_path, stream=True)
            content = response.content
        except requests.exceptions.RequestException as e:
            print(e)
            content = None
    elif parsed_path.scheme == "":
        if os.path.isfile(source_path):
            with open(input_images) as f:
                content = f.readlines()
            f.close
        else:
            print("file " + source_path + "is not accessible")
            content = None
    return content

def getJpeg(path, size, path_prefix):
    print(os.path.join(path_prefix,path))
    content = get_file_content(os.path.join(path_prefix,path))

    if content:
        try:
            img = np.frombuffer(content, dtype=np.uint8)
            img = cv2.imdecode(img, cv2.IMREAD_COLOR)  # BGR format
            # retrived array has BGR format and 0-255 normalization
            # add image preprocessing if needed by the model
            img = crop_resize(img, size, size)
            img = img.astype('float32')
            img = img.transpose(2,0,1).reshape(1,3,size,size)
            print(path, img.shape, "; data range:",np.amin(img),":",np.amax(img))
        except Exception as e:
            print("Can not read the image file", e)
            img = None
    else:
        print("Can not open ", os.path(path_prefix,path))
        img = None
    return img

parser = argparse.ArgumentParser(description='Sends requests to OVMS and TF Serving using images in numpy format')
parser.add_argument('--images_list', required=False, default='input_images.txt', help='Path to a file with a list of labeled images. It should include in every line a path to the image file and a numerical label separate by space.')
parser.add_argument('--grpc_endpoint',required=False, default='localhost:9000',  help='Specify endpoint of grpc service. default:localhost:9000')
parser.add_argument('--input_name',required=False, default='input', help='Specify input tensor name. default: input')
parser.add_argument('--output_name',required=False, default='resnet_v1_50/predictions/Reshape_1', help='Specify output name. default: output')
parser.add_argument('--model_name', default='resnet', help='Define model name, must be same as is in service. default: resnet',
                    dest='model_name')
parser.add_argument('--size',required=False, default=224, type=int, help='The size of the image in the model')
parser.add_argument('--image_path_prefix',required=False, default="", type=str, help='Path prefix to be added to every image in the list')
args = vars(parser.parse_args())

channel = grpc.insecure_channel(args['grpc_endpoint'])
stub = prediction_service_pb2_grpc.PredictionServiceStub(channel)
input_images = args.get('images_list')
size = args.get('size')

input_list_content = get_file_content(input_images)
if input_list_content is None:
    print("Can not open input images file", input_images)
    exit(1)
else:
    lines = input_list_content.decode().split("\n")
print(lines)
print('Start processing:')
print('\tModel name: {}'.format(args.get('model_name')))
print('\tImages list file: {}'.format(args.get('images_list')))

i = 0
matched = 0
processing_times = np.zeros((0),int)
imgs = np.zeros((0,3,size, size), np.dtype('<f'))
lbs = np.zeros((0), int)

for line in lines:
    path, label = line.strip().split(" ")
    img = getJpeg(path, size, args.get('image_path_prefix'))
    if img is not None:
        request = predict_pb2.PredictRequest()
        request.model_spec.name = args.get('model_name')
        request.inputs[args['input_name']].CopyFrom(tf_contrib_util.make_tensor_proto(img, shape=(img.shape)))
        start_time = datetime.datetime.now()
        result = stub.Predict(request, 10.0) # result includes a dictionary with all model outputs
        end_time = datetime.datetime.now()
        if args['output_name'] not in result.outputs:
            print("Invalid output name", args['output_name'])
            print("Available outputs:")
            for Y in result.outputs:
                print(Y)
            exit(1)
        duration = (end_time - start_time).total_seconds() * 1000
        processing_times = np.append(processing_times,np.array([int(duration)]))
        output = tf_contrib_util.make_ndarray(result.outputs[args['output_name']])
        nu = np.array(output)
        # for object classification models show imagenet class
        print('Processing time: {:.2f} ms; speed {:.2f} fps'.format(round(duration), 2),
          round(1000 / duration, 2)
          )
        ma = np.argmax(nu)
        if int(label) == ma:
            matched += 1
        i += 1
        print("Detected: {} - {} ; Should be: {} - {}".format(ma,classes.imagenet_classes[int(ma)],label,classes.imagenet_classes[int(label)]))

accuracy = matched/i
latency = np.average(processing_times)
metrics = {'metrics': [{'name': 'accuracy-score','numberValue':  accuracy,'format': "PERCENTAGE"},
                       {'name': 'latency','numberValue':  latency,'format': "RAW"}]}
with open('/mlpipeline-metrics.json', 'w') as f:
    json.dump(metrics, f)
f.close
print("\nOverall accuracy=",matched/i*100,"%")
print("Average latency=",latency,"ms")
