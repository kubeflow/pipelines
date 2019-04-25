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


"""Note that this sample is just to show the ResourceOp's usage.

It is not a good practice to put password as a pipeline argument, since it will
be visible on KFP UI.
"""

from kubernetes import client as k8s_client
import kfp.dsl as dsl


@dsl.pipeline(
    name="ResourceOp Basic",
    description="A Basic Example on ResourceOp Usage."
)
def resourceop_basic(username, password):
    secret_resource = k8s_client.V1Secret(
        api_version="v1",
        kind="Secret",
        metadata=k8s_client.V1ObjectMeta(generate_name="my-secret-"),
        type="Opaque",
        data={"username": username, "password": password}
    )
    rop = dsl.ResourceOp(
        name="create-my-secret",
        k8s_resource=secret_resource,
        attribute_outputs={"name": "{.metadata.name}"}
    )

    secret = k8s_client.V1Volume(
        name="my-secret",
        secret=k8s_client.V1SecretVolumeSource(secret_name=rop.output)
    )

    cop = dsl.ContainerOp(
        name="cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["ls /etc/secret-volume"],
        pvolumes={"/etc/secret-volume": secret}
    )


if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(resourceop_basic, __file__ + ".tar.gz")
