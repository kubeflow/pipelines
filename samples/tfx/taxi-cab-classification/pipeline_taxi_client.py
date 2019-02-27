# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""A client for the chicago_taxi demo."""

from __future__ import print_function

import argparse

from grpc.beta import implementations

import tensorflow as tf
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import prediction_service_pb2

_LOCAL_INFERENCE_TIMEOUT_SECONDS = 5.0


def _do_local_inference(host, port, examples, model_name):
    """Performs inference on a model hosted by the host:port server."""

    channel = implementations.insecure_channel(host, int(port))
    stub = prediction_service_pb2.beta_create_PredictionService_stub(channel)

    request = predict_pb2.PredictRequest()
    # request.model_spec.name = 'chicago_taxi'
    request.model_spec.name = model_name
    request.model_spec.signature_name = 'predict'

    tfproto = tf.contrib.util.make_tensor_proto([examples],
                                                shape=[len(examples)],
                                                dtype=tf.string)

    # request.inputs['examples'].CopyFrom(tfproto)
    request.inputs['csv_example'].CopyFrom(tfproto)
    print(stub.Predict(request, _LOCAL_INFERENCE_TIMEOUT_SECONDS))


def _do_inference(model_handle, examples_file, num_examples, model_name):
    """Sends requests to the model and prints the results."""

    input_file = open(examples_file, 'r')
    input_file.readline()  # skip header line

    examples = []
    for _ in range(num_examples):
        one_line = input_file.readline().rstrip()
        one_line_no_tips = ','.join(one_line.split(",")[:-1])
        examples.append(one_line_no_tips)
        if not one_line:
            print('End of example file reached')
            break

    parsed_model_handle = model_handle.split(':')
    _do_local_inference(
        host=parsed_model_handle[0],
        port=parsed_model_handle[1],
        examples=examples,
        model_name=model_name)


def main(_):
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--num_examples',
        help=('Number of examples to send to the server.'),
        default=1,
        type=int)
    parser.add_argument(
        '--server',
        help=('Prediction service host:port'),
        required=True)
    parser.add_argument(
        '--examples_file',
        help=('Path to csv file containing examples.'),
        required=True)
    parser.add_argument(
        '--model_name',
        help=('Model name.'),
        required=True)

    known_args, _ = parser.parse_known_args()
    _do_inference(known_args.server,
                  known_args.examples_file, known_args.num_examples,
                  known_args.model_name)


if __name__ == '__main__':
    tf.app.run()
