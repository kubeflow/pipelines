# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import json

from fire import decorators
from ._client import DataprocClient
from kfp_component.core import KfpExecutionContext, display
from .. import common as gcp_common

@decorators.SetParseFns(image_version=str)
def create_cluster(project_id, region, name=None, name_prefix=None,
    initialization_actions=None, config_bucket=None, image_version=None,
    cluster=None, wait_interval=30):
    if not cluster:
        cluster = {}
    cluster['projectId'] = project_id
    cluster['config'] = {}
    if name:
        cluster['clusterName'] = name
    if initialization_actions:
        cluster['config']['initializationActions'] = list(
            map(lambda file: {
                'executableFile': file
            }, initialization_actions)
        )
    if config_bucket:
        cluster['config']['configBucket'] = config_bucket
    if image_version:
        if 'softwareConfig' not in cluster:
            cluster['softwareConfig'] = {}
        cluster['softwareConfig']['imageVersion'] = image_version

    return _create_cluster_internal(project_id, region, cluster, name_prefix,
        wait_interval)

def _create_cluster_internal(project_id, region, cluster, name_prefix, 
    wait_interval):
    client = DataprocClient()
    operation_name = None
    with KfpExecutionContext(
        on_cancel=lambda: client.cancel_operation(operation_name)) as ctx:
        _set_cluster_name(cluster, ctx.context_id(), name_prefix)
        _dump_metadata(cluster, region)
        operation = client.create_cluster(project_id, region, cluster, 
            request_id=ctx.context_id())
        operation_name = operation.get('name')
        operation = client.wait_for_operation_done(operation_name, 
            wait_interval)
        return _dump_cluster(operation.get('response'))

def _set_cluster_name(cluster, context_id, name_prefix):
    if 'clusterName' in cluster:
        return
    if not name_prefix:
        name_prefix = 'cluster'
    cluster['clusterName'] = name_prefix + '-' + context_id

def _dump_metadata(cluster, region):
    display.display(display.Link(
        'https://console.cloud.google.com/dataproc/clusters/{}?project={}&region={}'.format(
            cluster.get('clusterName'), cluster.get('projectId'), region),
        'Cluster Details'
    ))

def _dump_cluster(cluster):
    gcp_common.dump_file('/tmp/kfp/output/dataproc/cluster.json', 
        json.dumps(cluster))
    gcp_common.dump_file('/tmp/kfp/output/dataproc/cluster_name.txt',
        cluster.get('clusterName'))
    return cluster
