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
    cluster=None, wait_interval=30,
    cluster_name_output_path='/tmp/kfp/output/dataproc/cluster_name.txt',
    cluster_object_output_path='/tmp/kfp/output/dataproc/cluster.json',
):
    """Creates a DataProc cluster under a project.

    Args:
        project_id (str): Required. The ID of the Google Cloud Platform project 
            that the cluster belongs to.
        region (str): Required. The Cloud Dataproc region in which to handle the 
            request.
        name (str): Optional. The cluster name. Cluster names within a project
            must be unique. Names of deleted clusters can be reused.
        name_prefix (str): Optional. The prefix of the cluster name.
        initialization_actions (list): Optional. List of GCS URIs of executables 
            to execute on each node after config is completed. By default,
            executables are run on master and all worker nodes. 
        config_bucket (str): Optional. A Google Cloud Storage bucket used to 
            stage job dependencies, config files, and job driver console output.
        image_version (str): Optional. The version of software inside the cluster.
        cluster (dict): Optional. The full cluster config. See [full details](
            https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#Cluster)
        wait_interval (int): The wait seconds between polling the operation. 
            Defaults to 30s.

    Returns:
        The created cluster object.

    Output Files:
        $KFP_OUTPUT_PATH/dataproc/cluster_name.txt: The cluster name of the 
            created cluster.
    """
    if not cluster:
        cluster = {}
    cluster['projectId'] = project_id
    if 'config' not in cluster:
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
        if 'softwareConfig' not in cluster['config']:
            cluster['config']['softwareConfig'] = {}
        cluster['config']['softwareConfig']['imageVersion'] = image_version

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
        cluster = operation.get('response')
        gcp_common.dump_file(cluster_object_output_path, json.dumps(cluster))
        gcp_common.dump_file(cluster_name_output_path, cluster.get('clusterName'))
        return cluster

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
