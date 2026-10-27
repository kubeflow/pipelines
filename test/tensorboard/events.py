# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Grow and rotate event objects through the candidate's storage clients."""
import io
import os
import sys
import time

from PIL import Image
from tensorboard.compat.proto import event_pb2
from tensorboard.compat.proto import graph_pb2
from tensorboard.compat.proto import node_def_pb2
from tensorboard.compat.proto import summary_pb2
from tensorboard.summary.writer.record_writer import RecordWriter

scheme, step = sys.argv[1], int(sys.argv[2])
buffer = io.BytesIO()
writer = RecordWriter(buffer)
if step != 2:
    writer.write(
        event_pb2.Event(file_version='brain.Event:2').SerializeToString())
if step == 1:
    graph = graph_pb2.GraphDef(
        node=[node_def_pb2.NodeDef(name='input', op='Placeholder')])
    writer.write(
        event_pb2.Event(
            wall_time=time.time(),
            graph_def=graph.SerializeToString()).SerializeToString())
    png = io.BytesIO()
    Image.new('RGB', (2, 2), 'blue').save(png, format='PNG')
    writer.write(
        event_pb2.Event(
            wall_time=time.time(),
            step=step,
            summary=summary_pb2.Summary(value=[
                summary_pb2.Summary.Value(
                    tag='preview',
                    image=summary_pb2.Summary.Image(
                        height=2,
                        width=2,
                        colorspace=3,
                        encoded_image_string=png.getvalue()))
            ])).SerializeToString())
writer.write(
    event_pb2.Event(
        wall_time=time.time(),
        step=step,
        summary=summary_pb2.Summary(value=[
            summary_pb2.Summary.Value(tag='accuracy', simple_value=step / 4)
        ])).SerializeToString())
writer.flush()
# Step2 grows the existing object; step3 rotates to a lexically newer file.
key = 'logs/legacy/events.out.tfevents.' + ('0000000002' if step == 3 else
                                            '0000000001') + '.fixture'
body = buffer.getvalue()
if scheme == 's3':
    import boto3
    session = boto3.Session()
    assert session.get_credentials().method == 'shared-credentials-file'
    client = session.client('s3', endpoint_url=os.environ['S3_ENDPOINT'])
    if step == 1:
        client.create_bucket(Bucket='tensorboard-fixtures')
    if step == 2:
        body = client.get_object(
            Bucket='tensorboard-fixtures', Key=key)['Body'].read() + body
    client.put_object(Bucket='tensorboard-fixtures', Key=key, Body=body)
elif scheme == 'gs':
    import gcsfs
    fs = gcsfs.GCSFileSystem(token='anon')
    if step == 1:
        fs.mkdir('tensorboard-fixtures')
    if step == 2:
        body = fs.cat('tensorboard-fixtures/' + key) + body
    fs.pipe('tensorboard-fixtures/' + key, body)
else:
    raise ValueError('Use s3 or gs for deployed event acceptance')
print(f'Uploaded {scheme} event step {step}', flush=True)
