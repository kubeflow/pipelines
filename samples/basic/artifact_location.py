#!/usr/bin/env python3
# Copyright 2019 Google LLC
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

from kfp import dsl
from kubernetes.client import V1SecretKeySelector


@dsl.pipeline(
    name="custom_artifact_location_pipeline",
    description="""A pipeline to demonstrate how to configure the artifact 
    location for all the ops in the pipeline.""",
)
def custom_artifact_location(
    tag: str, namespace: str = "kubeflow", bucket: str = "somebucket"
):

    # configures artifact location
    pipeline_artifact_location = dsl.ArtifactLocation.s3(
        bucket=bucket,
        endpoint="minio-service.%s:9000" % namespace,  # parameterize minio-service endpoint
        insecure=True,
        access_key_secret=V1SecretKeySelector(name="minio", key="accesskey"),
        secret_key_secret={"name": "minio", "key": "secretkey"},  # accepts dict also
    )

    # set pipeline level artifact location
    dsl.get_pipeline_conf().set_artifact_location(pipeline_artifact_location)

    # artifacts in this op are stored to endpoint `minio-service.<namespace>:9000`
    op = dsl.ContainerOp(name="foo", image="busybox:%s" % tag)
