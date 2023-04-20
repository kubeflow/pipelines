# Copyright 2023 The Kubeflow Authors
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
"""A pipeline that passes a secret as a volume to a container."""
from kfp import dsl
from kfp import kubernetes
from kfp import compiler


@dsl.component
def comp():
    import os

    secret_key = "type"
    secret_path = os.path.join('/mnt/my_vol', secret_key)

    # Check if the secret exists
    if not os.path.exists(secret_path):
        raise Exception('Secret not found')

    # Open the secret
    with open(secret_path, 'rb') as secret_file:
        secret_data = secret_file.read()

    # Decode the secret
    secret_data = secret_data.decode('utf-8')

    # Print the secret
    print(secret_data)

    if secret_data == "service_account":
        print("Success")
        return 0
    else:
        sys.exit("Failure: cannot access secret as volume variable")


@dsl.pipeline
def pipeline_secret_volume():
    task = comp()
    kubernetes.use_secret_as_volume(
        task, secret_name='user-gcp-sa', mount_path='/mnt/my_vol')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_secret_volume, 
        package_path='pipeline_secret_volume.yaml')