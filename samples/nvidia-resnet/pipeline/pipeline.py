# Copyright (c) 2019, NVIDIA CORPORATION. All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#  * Redistributions of source code must retain the above copyright
#    notice, this list of conditions and the following disclaimer.
#  * Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
#  * Neither the name of NVIDIA CORPORATION nor the names of its
#    contributors may be used to endorse or promote products derived
#    from this software without specific prior written permission.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS ``AS IS'' AND ANY
# EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
# PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL THE COPYRIGHT OWNER OR
# CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
# EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
# PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
# PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
# OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
# OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


import kfp.dsl as dsl
import datetime
import os
from kubernetes import client as k8s_client


PIPELINE_NAME = 'resnet_cifar10_pipeline'


def preprocess_op(persistent_volume_path, input_dir, output_dir, step_name='preprocess'):
    return dsl.ContainerOp(
        name=step_name,
        image='nvcr.io/nvidian/sae/ananths:kubeflow-preprocess',
        command=['python'],
        arguments=[
            '/scripts/preprocess.py',
            '--input_dir', '%s/%s' % (persistent_volume_path, input_dir),
            '--output_dir', '%s/%s' % (persistent_volume_path, output_dir),
        ],
        file_outputs={}
    )


def train_op(persistent_volume_path, input_dir, output_dir, step_name='train'):
    return dsl.ContainerOp(
        name=step_name,
        image='nvcr.io/nvidian/sae/ananths:kubeflow-train',
        command=['python'],
        arguments=[
            '/scripts/train.py',
            '--input_dir', '%s/%s' % (persistent_volume_path, input_dir),
            '--output_dir', '%s/%s' % (persistent_volume_path, output_dir),
        ],
        file_outputs={}
    )


def serve_op(persistent_volume_path, input_dir, step_name='serve'):
    return dsl.ContainerOp(
        name=step_name,
        image='nvcr.io/nvidian/sae/ananths:kubeflow-serve',
        command=['python3'],
        arguments=[
            '/scripts/serve.py',
            '--input_dir', '%s/%s' % (persistent_volume_path, input_dir),
        ],
        file_outputs={}
    )


@dsl.pipeline(
    name=PIPELINE_NAME,
    description='Demonstrate the ResNet50 predict.'
)
def resnet_pipeline(
    raw_data_dir=dsl.PipelineParam(name='raw-data-dir', value='raw_data'),
    processed_data_dir=dsl.PipelineParam(name='processed-data-dir', value='processed_data'),
    model_dir=dsl.PipelineParam(name='saved-model-dir', value='saved_model')
):

    op_dict = {}

    persistent_volume_name = 'nvidia-workspace'
    persistent_volume_path = '/mnt/workspace/'

    op_dict['preprocess'] = preprocess_op(
        persistent_volume_path, raw_data_dir, processed_data_dir)

    op_dict['train'] = train_op(
        persistent_volume_path, processed_data_dir, model_dir)
    op_dict['train'].after(op_dict['preprocess'])

    op_dict['serve'] = serve_op(
        persistent_volume_path, model_dir)
    op_dict['serve'].after(op_dict['train'])

    for _, container_op in op_dict.items():
        container_op.add_volume(k8s_client.V1Volume(
            host_path=k8s_client.V1HostPathVolumeSource(
                path=persistent_volume_path),
            name=persistent_volume_name))
        container_op.add_volume_mount(k8s_client.V1VolumeMount(
            mount_path=persistent_volume_path, name=persistent_volume_name))


if __name__ == '__main__':
    import kfp.compiler as compiler
    compiler.Compiler().compile(resnet_pipeline, __file__ + '.tar.gz')
