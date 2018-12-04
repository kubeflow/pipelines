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

def use_gcp_secret(secret_name='user-gcp-sa', secret_file_path_in_volume='/user-gcp-sa.json', volume_name='gcp-credentials', secret_volume_mount_path='/secret/gcp-credentials'):
    """An operator that configures the container to use GCP service account.

        The user-gcp-sa secret is created as part of the kubeflow deployment that
        stores the access token for kubeflow user service account.

        With this service account, the container has a range of GCP APIs to
        access to. This service account is automatically created as part of the
        kubeflow deployment.

        For the list of the GCP APIs this service account can access to, check
        https://github.com/kubeflow/kubeflow/blob/7b0db0d92d65c0746ac52b000cbc290dac7c62b1/deployment/gke/deployment_manager_configs/iam_bindings_template.yaml#L18

        If you want to call the GCP APIs in a different project, grant the kf-user
        service account access permission.
    """

    def _use_gcp_secret(task):
        from kubernetes import client as k8s_client
        return (
            task
                .add_volume(
                    k8s_client.V1Volume(
                        name=volume_name,
                        secret=k8s_client.V1SecretVolumeSource(
                            secret_name=secret_name,
                        )
                    )
                )
                .add_volume_mount(
                    k8s_client.V1VolumeMount(
                        name=volume_name,
                        mount_path=secret_volume_mount_path,
                    )
                )
                .add_env_variable(
                    k8s_client.V1EnvVar(
                        name='GOOGLE_APPLICATION_CREDENTIALS',
                        value=secret_volume_mount_path + secret_file_path_in_volume,
                    )
                )
        )
    
    return _use_gcp_secret
