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

from kubernetes import client as k8s_client
import kfp.dsl as dsl


@dsl.pipeline(
    name="Seldon MNIST TF",
    description="The third example of the design doc."
)
def mnist_tf():

    secret = k8s_client.V1Volume(
        name="docker-config-secret",
        secret=k8s_client.V1SecretVolumeSource(secret_name='docker-config')
    )

    vop = dsl.VolumeOp(
        name="mypvc",
        resource_name="newpvc",
        size="50Mi",
        modes=dsl.VOLUME_MODE_RWO
    )

    clone = dsl.ContainerOp(
        name="clone",
        image="alpine/git:latest",
        command=["sh", "-c"],
        arguments=["git clone --depth 1 --branch master https://github.com/kubeflow/example-seldon.git; cp ./example-seldon/models/tf_mnist/train/* /workspace; ls /workspace/;"],
        pvolumes={"/workspace": vop.volume}
    )

    build = dsl.ContainerOp(
        name="build",
        image="gcr.io/kaniko-project/executor:latest",
        arguments=["--dockerfile","Dockerfile","--destination","ryandawsonuk/deepmnistclassifier_trainer:0.3"],
        pvolumes={"/workspace": clone.pvolume,"/root/.docker/": secret}
    )

    serve = dsl.ContainerOp(
        name="serve",
        image="library/bash:4.4.23",
        command=["echo", "This could become a serving step. Would first need to build serving image. Serve by deploying SeldonDeployment k8s resource."],
        pvolumes={"/workspace": vop.volume.after(build)}
    )


if __name__ == "__main__":
    import kfp.compiler as compiler
    compiler.Compiler().compile(mnist_tf, __file__ + ".tar.gz")
