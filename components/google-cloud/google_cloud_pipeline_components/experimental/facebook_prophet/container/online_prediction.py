#!/usr/bin/env python3
# Copyright 2021 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""HTTP server for making predictions using a trained Facebook Prophet model."""

import json
import os
from flask import Flask, request
from google.cloud import storage
from prophet import Prophet
from prophet.serialize import model_from_json
import json
from functools import cache
import tempfile
from prediction import predict
from typing import Optional
from sys import stderr

storage_client = storage.Client()
artifact_blob = storage.Blob.from_string(os.environ['AIP_STORAGE_URI'])
model_blobs: dict[str, storage.Blob] = {}


def setup():
  # Discover available models
  for blob in storage_client.list_blobs(
      artifact_blob.bucket.name, prefix=artifact_blob.name):
    if blob.name.endswith('.model.json'):
      id = blob.name.removeprefix(artifact_blob.name).removesuffix('.model.json')
      id = id[1:] if id.startswith('/') else id
      model_blobs[id] = blob
  print(
      'Found {} models: {}'.format(
          len(model_blobs), ', '.join(model_blobs.keys())),
      file=stderr)


@cache
def get_model(id: str) -> Optional[Prophet]:
  if not id in model_blobs:
    return None
  with tempfile.TemporaryFile() as temp_file:
    storage_client.download_blob_to_file(model_blobs[id], temp_file)
    temp_file.seek(0)
    return model_from_json(json.load(temp_file))


app = Flask(__name__)


@app.route(os.environ['AIP_HEALTH_ROUTE'])
def health_check():
  return 'OK'


@app.route(os.environ['AIP_PREDICT_ROUTE'], methods=['POST'])
def prediction_endpoint():
  if request.json is None or 'instances' not in request.json or type(
      request.json['instances']) != list or not len(request.json['instances']):
    return 'Malformed request.', 400
  predictions = []
  for instance in request.json['instances']:
    if 'time_series' not in instance:
      return 'time_series missing from instance: ' + json.dumps(instance), 400
    if (m := get_model(instance['time_series'])) is None:
      return 'No model found for time series: {}. Options are: {}'.format(
          instance['time_series'], ', '.join(model_blobs.keys())), 400
    if 'periods' in instance:
      predictions.append(predict(m, periods=int(instance['periods'])).to_json())
    elif 'future_data_source' in instance:
      predictions.append(
          predict(m,
                  future_data_source=instance['future_data_source']).to_json())
    else:
      return 'neither periods nor future_data_source in instance: ' + json.dumps(
          instance), 400
  return json.dumps({'predictions': predictions})

if not 'ONLINE_PREDICTION_UNITTEST' in os.environ:
  setup()
