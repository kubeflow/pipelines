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
from typing import Dict, Union, Any

from argo.models import V1alpha1ArtifactLocation, V1alpha1S3Artifact, V1alpha1Artifact
from deprecated.sphinx import deprecated
from kubernetes.client.models import V1SecretKeySelector


def _dict_to_secret(
    value: Union[V1SecretKeySelector, Dict[str, Any]]
) -> V1SecretKeySelector:
    """Converts a dict to a kubernetes V1SecretKeySelector."""
    if isinstance(value, dict) and value.get("name") and value.get("key"):
        return V1SecretKeySelector(**value)
    return value or V1SecretKeySelector(key="", optional=True)


@deprecated(version='0.1.32', reason='ArtifactLocation is deprecated since SDK v0.1.32. Please configure the artifact location in the cluster configMap: https://github.com/argoproj/argo/blob/master/ARTIFACT_REPO.md#configure-the-default-artifact-repository .')
class ArtifactLocation:
    """
    ArtifactLocation describes a location for a single or multiple artifacts.
    It is used as single artifact in the context of inputs/outputs
    (e.g. outputs.artifacts.artname). It is also used to describe the location
    of multiple artifacts such as the archive location of a single workflow
    step, which the executor will use as a default location to store its files.
    """

    @staticmethod
    def s3(
        bucket: str = None,
        endpoint: str = None,
        insecure: bool = None,
        region: str = None,
        access_key_secret: Union[V1SecretKeySelector, Dict[str, Any]] = None,
        secret_key_secret: Union[V1SecretKeySelector, Dict[str, Any]] = None,
    ) -> V1alpha1ArtifactLocation:
        """
        Creates a new instance of V1alpha1ArtifactLocation with a s3 artifact
        backend.

        Example::

          from kubernetes.client.models import V1SecretKeySelector
          from kfp.dsl import ArtifactLocation


          artifact_location = ArtifactLocation(
            bucket="foo",
            endpoint="s3.amazonaws.com",
            insecure=False,
            region="ap-southeast-1",
            access_key_secret={"name": "s3-secret", "key": "accesskey"},
            secret_key_secret=V1SecretKeySelector(name="s3-secret", key="secretkey")
          )

        Args:
          bucket (str): name of the bucket.
          endpoint (str): hostname to the bucket endpoint.
          insecure (bool): use TLS if set to True.
          region (str): bucket region (for s3 buckets).
          access_key_secret (Union[V1SecretKeySelector, Dict[str, Any]]): k8s secret selector to access key.
          secret_key_secret (Union[V1SecretKeySelector, Dict[str, Any]]): k8s secret selector to secret key.

        Returns:
          V1alpha1ArtifactLocation: a new instance of V1alpha1ArtifactLocation.
        """
        return V1alpha1ArtifactLocation(
            s3=V1alpha1S3Artifact(
                bucket=bucket,
                endpoint=endpoint,
                insecure=insecure,
                region=region,
                access_key_secret=_dict_to_secret(access_key_secret),
                secret_key_secret=_dict_to_secret(secret_key_secret),
                key="",  # key is a required value for V1alpha1S3Artifact
            )
        )

    @staticmethod
    def create_artifact_for_s3(
        artifact_location: Union[V1alpha1ArtifactLocation, Dict[str, Any]],
        name: str,
        path: str,
        key: str,
        **kwargs
    ) -> V1alpha1Artifact:
        """
        Creates a s3-backed `V1alpha1Artifact` object using a
        `V1alpha1ArtifactLocation` object.

        Args:
          artifact_location (Union[V1alpha1ArtifactLocation, Dict[str, Any]]): `V1alpha1ArtifactLocation`
            object or a dict representing it.
          name (str): name of the artifact. must be unique within a template's
            inputs/outputs.
          path (str): container path to the artifact.
          key (str): key in bucket to store artifact.
          **kwargs: any other keyword arguments accepted by `V1alpha1Artifact`.

        Returns:
          V1alpha1Artifact: V1alpha1Artifact object.
        """
        if not artifact_location:
            return V1alpha1Artifact(
                name=name,
                path=path,
                **kwargs
            )

        # dict representation of artifact location
        if isinstance(artifact_location, dict) and artifact_location.get("s3"):
          s3_artifact = artifact_location.get("s3")
          return V1alpha1Artifact(
            name=name,
            path=path,
            s3=V1alpha1S3Artifact(
              bucket=s3_artifact.get("bucket"),
              endpoint=s3_artifact.get("endpoint"),
              insecure=s3_artifact.get("insecure"),
              region=s3_artifact.get("region"),
              access_key_secret=_dict_to_secret(s3_artifact.get("accessKeySecret")),
              secret_key_secret=_dict_to_secret(s3_artifact.get("secretKeySecret")),
              key=key
            )
          )

        if artifact_location.s3:
            return V1alpha1Artifact(
                name=name,
                path=path,
                s3=V1alpha1S3Artifact(
                    bucket=artifact_location.s3.bucket,
                    endpoint=artifact_location.s3.endpoint,
                    insecure=artifact_location.s3.insecure,
                    region=artifact_location.s3.region,
                    access_key_secret=artifact_location.s3.access_key_secret,
                    secret_key_secret=artifact_location.s3.secret_key_secret,
                    key=key,
                ),
                **kwargs
            )
        raise ValueError("artifact_location does not have s3 configuration.")
