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
from kfp.dsl import get_pipeline_conf, S3Artifactory
from kubernetes.client.models.v1_secret_key_selector import V1SecretKeySelector


@dsl.pipeline(
    name='foo', description='A pipeline with parameterized output artifacts')
def foo_pipeline(bucket: str = 'foobucket'):
    """A pipeline with parameterized output artifacts."""

    # parameterized s3 artifactory
    s3_artifactory = (S3Artifactory()
                        .bucket(bucket)
                        .endpoint('s3.amazonaws.com')
                        .insecure(False)
                        .region('ap-southeast-1')
                        .access_key_secret(name='s3_credential', key='accesskey')
                        .secret_key_secret(name='s3_credential', key='secretkey'))

    get_pipeline_conf().set_s3_artifactory(s3_artifactory)

    # this op will upload metadata & metrics to input-variable:`bucket`
    op1 = dsl.ContainerOp("s3artifactory", image="busybox:latest")
    # this op will upload metadata & metrics to `barbucket`
    op2 = dsl.ContainerOp("bar",
                          image="busybox:latest",
                          command="echo '%s'" % op1.output,
                          s3_artifactory=S3Artifactory(
                              bucket="barbucket",
                              endpoint="minio-service.kubeflow:9000",
                              insecure=True,
                              region=None,
                              access_key_secret=V1SecretKeySelector(name='mlpipeline-minio-artifact', key='accesskey'),
                              secret_key_secret=V1SecretKeySelector(name='mlpipeline-minio-artifact', key='secretkey')
                              ))
