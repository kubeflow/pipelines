from typing import Callable

from kubernetes.client.models.v1_secret_key_selector import V1SecretKeySelector
from argo.models.io_argoproj_workflow_v1alpha1_s3_artifact import IoArgoprojWorkflowV1alpha1S3Artifact

from . import _pipeline_param

# default k8s secrets
_access_key_secret = V1SecretKeySelector(
    name='mlpipeline-minio-artifact', key='accesskey')
_secret_key_secret = V1SecretKeySelector(
    name='mlpipeline-minio-artifact', key='secretkey')


class S3Artifactory(object):
    """Factory class to create `io.argoproj.workflow.v1alpha1.S3Artifact` 
    objects.
    """
    """
    Attributes:
      swagger_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    swagger_types = {'config': 'dict'}
    attribute_map = {'config': 'config'}

    def __init__(self,
                 bucket: str = 'mlpipeline',
                 endpoint: str = 'minio-service.kubeflow:9000',
                 insecure: bool = True,
                 region: str = None,
                 access_key_secret: V1SecretKeySelector = _access_key_secret,
                 secret_key_secret: V1SecretKeySelector = _secret_key_secret):
        """
        Creates a new instance of a S3ArtifactFactory.
        
        Args:
            bucket {str} -- bucket to store/retrieve artifacts (default: {'mlpipeline'})
            endpoint {str} -- api endpoint to call to store/retrieve artifacts (default: {'minio-service.kubeflow:9000'})
            insecure {bool} -- set to true for http (instead of https) api calls (default: {True})
            region {str} -- region for AWS s3 (default: {None})
            access_key_secret {V1SecretKeySelector} --  k8s secret for s3/minio access key (default: {name='mlpipeline-minio-artifact', key='accesskey'})
            secret_key_secret {V1SecretKeySelector} -- k8s secret for s3/minio access secret key (default: {name='mlpipeline-minio-artifact', key='secretkey'})
        """
        self._conf = dict(
            bucket=bucket,
            endpoint=endpoint,
            insecure=insecure,
            region=region,
            access_key_secret=access_key_secret,
            secret_key_secret=secret_key_secret)

    @property
    def config(self):
        """Configuration for S3Artifactory as a dict."""
        return self._conf

    def bucket(self, bucket: str):
        """bucket to store/retrieve artifacts."""
        self._conf['bucket'] = bucket
        return self

    def endpoint(self, endpoint: str):
        """endpoint to call when storing/retrieving artifacts."""
        self._conf['endpoint'] = endpoint
        return self

    def insecure(self, insecure: bool = True):
        """if set, http instead of https is used."""
        self._conf['insecure'] = insecure
        return self

    def region(self, region: str):
        """s3 aws region."""
        self._conf['region'] = region
        return self

    def access_key_secret(self,
                          key: str,
                          name: str = None,
                          optional: bool = None):
        """k8s secret to s3/minio access key."""
        self._conf['access_key_secret'] = V1SecretKeySelector(
            name=name, key=key, optional=optional)
        return self

    def secret_key_secret(self,
                          key: str,
                          name: str = None,
                          optional: bool = None):
        """k8s secret to s3/minio access secret key."""
        self._conf['secret_key_secret'] = V1SecretKeySelector(
            name=name, key=key, optional=optional)
        return self

    def create(self, key: str) -> IoArgoprojWorkflowV1alpha1S3Artifact:
        """Return a `io.argoproj.workflow.v1alpha1.S3Artifact` object based on
        the configured S3ArtifactFactory."""
        return IoArgoprojWorkflowV1alpha1S3Artifact(key=key, **self._conf)

    def process_pipelineparams(self, func: Callable):
        """Apply a func on any `PipelineParam` in the object"""
        self._conf = func(self._conf)
        return self