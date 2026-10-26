#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
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
import importlib.metadata
import io
import json
import os
import pathlib
import subprocess
import time
import urllib.request

from PIL import Image
from tensorboard.compat import tf
from tensorboard.compat.proto import event_pb2
from tensorboard.compat.proto import graph_pb2
from tensorboard.compat.proto import node_def_pb2
from tensorboard.compat.proto import summary_pb2
from tensorboard.summary.writer.event_file_writer import EventFileWriter

root = pathlib.Path('/tmp/tensorboard-acceptance')
root.mkdir(exist_ok=True)
writer = EventFileWriter(str(root / 'legacy'))
graph = graph_pb2.GraphDef(
    node=[node_def_pb2.NodeDef(name='input', op='Placeholder')])
writer.add_event(
    event_pb2.Event(
        wall_time=time.time(), step=1, graph_def=graph.SerializeToString()))
png = io.BytesIO()
Image.new('RGB', (2, 2), 'blue').save(png, format='PNG')
writer.add_event(
    event_pb2.Event(
        wall_time=time.time(),
        step=1,
        summary=summary_pb2.Summary(value=[
            summary_pb2.Summary.Value(tag='accuracy', simple_value=0.75),
            summary_pb2.Summary.Value(
                tag='preview',
                image=summary_pb2.Summary.Image(
                    height=2,
                    width=2,
                    colorspace=3,
                    encoded_image_string=png.getvalue())),
        ])))
writer.flush()
writer.close()
schemes = ['local']
assert os.environ.get('S3_ENDPOINT'), 'Set S3_ENDPOINT for runtime acceptance'
assert os.environ.get(
    'STORAGE_EMULATOR_HOST'), 'Set STORAGE_EMULATOR_HOST for runtime acceptance'
print(
    json.dumps({
        'versions': {
            name: importlib.metadata.version(name) for name in
            ['tensorboard', 'protobuf', 'Werkzeug', 'boto3', 'gcsfs']
        },
        'filesystem_module': tf.io.gfile.__name__
    }),
    flush=True)
# Exercise the same filesystem readers used by TensorBoard against actual
# S3/GCS protocol emulators, not merely import their libraries.
if os.environ.get('S3_ENDPOINT'):
    import boto3
    s3 = boto3.client('s3', endpoint_url=os.environ['S3_ENDPOINT'])
    s3.create_bucket(Bucket='tensorboard-fixtures')
    event = next((root / 'legacy').glob('events.*'))
    s3.put_object(
        Bucket='tensorboard-fixtures',
        Key='events/' + event.name,
        Body=event.read_bytes())
    with tf.io.gfile.GFile('s3://tensorboard-fixtures/events/' + event.name,
                           'rb') as stream:
        assert stream.read() == event.read_bytes()
    assert tf.io.gfile.listdir('s3://tensorboard-fixtures/events')
    schemes.append('s3')
    print('S3 event read/list passed', flush=True)
if os.environ.get('STORAGE_EMULATOR_HOST'):
    import gcsfs

    # This emulator must allow anonymous requests. gcsfs reads the emulator
    # endpoint from the environment and uses its normal credential fallback.
    fs = gcsfs.GCSFileSystem(token='anon')
    fs.mkdir('tensorboard-fixtures')
    event = next((root / 'legacy').glob('events.*'))
    fs.pipe('tensorboard-fixtures/events/' + event.name, event.read_bytes())
    with tf.io.gfile.GFile('gs://tensorboard-fixtures/events/' + event.name,
                           'rb') as stream:
        assert stream.read() == event.read_bytes()
    assert tf.io.gfile.listdir('gs://tensorboard-fixtures/events')
    schemes.append('gs')
    print('GCS event read/list passed', flush=True)
prefix = '/tensorboard/viewer-acceptance'
process = subprocess.Popen([
    'tensorboard', '--logdir=' + str(root), '--path_prefix=' + prefix + '/',
    '--bind_all', '--port=6006', '--reload_interval=1'
])
try:
    base = 'http://127.0.0.1:6006' + prefix

    def get(path):
        with urllib.request.urlopen(base + path, timeout=5) as response:
            return response.read()

    for attempt in range(90):
        try:
            plugins = json.loads(get('/data/plugins_listing'))
            if plugins['scalars']['enabled'] and plugins['images'][
                    'enabled'] and plugins['graphs']['enabled']:
                break
        except Exception:
            pass
        time.sleep(1)
    else:
        raise AssertionError(
            'TensorBoard did not ingest scalar, image and graph fixtures')
    assert 'legacy' in json.loads(get('/data/plugin/scalars/tags'))
    assert 'legacy' in json.loads(get('/data/plugin/graphs/info'))
    assert b'input' in get('/data/plugin/graphs/graph?run=legacy')
    assert get('/')
    print('Event ingestion and prefixed endpoints passed', flush=True)
    pathlib.Path('/tmp/tensorboard-ready').write_text(json.dumps(schemes))
    # Keep server alive for an external browser test of the historical prefix bug.
    process.wait()
finally:
    process.terminate()
