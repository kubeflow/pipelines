# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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

import json
import logging
import os
import time
from os import path
from google.api_core import gapic_v1
from google.cloud import aiplatform
from google_cloud_pipeline_components.experimental.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format

_POLLING_INTERVAL_IN_SECONDS = 20

def upload_model(
    type,
    project,
    location,
    payload,
    gcp_resources,
    output_model_artifact,
):
    """
  """
    logging.warning('***************** debug')
    logging.warning(output_model_artifact)
    logging.warning("************ debug done")
    client_options = {"api_endpoint": location + '-aiplatform.googleapis.com'}
    client_info = gapic_v1.client_info.ClientInfo(
        user_agent="google-cloud-pipeline-components",
    )

    lro_prefix = f"https://{client_options['api_endpoint']}/v1/"

    # Initialize client that will be used to create and send requests.
    model_client = aiplatform.gapic.ModelServiceClient(
        client_options=client_options, client_info=client_info
    )

    # Currently we don't check if operation already exists and continue from there
    # If this is desirable to the user and improves the reliability, we could do the following
    # ```
    # from google.api_core import operations_v1, grpc_helpers
    # channel = grpc_helpers.create_channel(location + '-aiplatform.googleapis.com')
    # api = operations_v1.OperationsClient(channel)
    # current_status = api.get_operation(upload_model_lro.operation.name)
    # ```

    parent = f"projects/{project}/locations/{location}"
    model_spec = json.loads(payload, strict=False)
    upload_model_lro = model_client.upload_model(parent=parent, model=model_spec)
    upload_model_lro_name = upload_model_lro.operation.name

    # Write the lro to the output
    long_running_operations = GcpResources()
    long_running_operation = long_running_operations.resources.add()
    long_running_operation.resource_type = "VertexLro"
    long_running_operation.resource_uri = f"{lro_prefix}{upload_model_lro_name}"

    with open(gcp_resources, 'w') as f:
        f.write(json_format.MessageToJson(long_running_operations))

    # Poll the LRO till done
    while not upload_model_lro.done():
        time.sleep(_POLLING_INTERVAL_IN_SECONDS)

    if upload_model_lro.operation.error.code:
        raise RuntimeError(
            "Failed to upload model. Error: {}".format(upload_model_lro.operation.error)
        )
    else:
        logging.info(
            'Upload model complete. %s.', upload_model_lro.result()
        )
        return
