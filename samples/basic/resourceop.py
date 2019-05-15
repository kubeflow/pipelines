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
from kubernetes.client import V1Secret, V1ObjectMeta, V1Volume, V1SecretVolumeSource


@dsl.pipeline(
    name="ResourceOp Basic",
    description="""A pipeline to demonstrate how to use ResourceOp to create a 
    k8s resource and use in a pipeline."""
)
def resourceop_basic(username: str, password: str):
    # creates a k8s secret
    secret_resource = V1Secret(
        api_version="v1",
        kind="Secret",
        metadata=V1ObjectMeta(generate_name="my-secret-"),
        type="Opaque",
        data={"username": username, "password": password}
    )
    # creates a resource op
    rop = dsl.ResourceOp(
        name="create-my-secret",
        k8s_resource=secret_resource,
        attribute_outputs={"name": "{.metadata.name}"}
    )
    # creates a k8s volume
    secret = V1Volume(
        name="my-secret",
        secret=V1SecretVolumeSource(secret_name=rop.output)
    )
    # creates a container op
    cop = dsl.ContainerOp(
        name="cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["ls /etc/secret-volume"],
        pvolumes={"/etc/secret-volume": secret}
    )