# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.

    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import unittest

import kfp_server_api_v1beta1
from kfp_server_api_v1beta1.api.pipeline_upload_service_api import PipelineUploadServiceApi  # noqa: E501
from kfp_server_api_v1beta1.rest import ApiException


class TestPipelineUploadServiceApi(unittest.TestCase):
    """PipelineUploadServiceApi unit test stubs"""

    def setUp(self):
        self.api = kfp_server_api_v1beta1.api.pipeline_upload_service_api.PipelineUploadServiceApi()  # noqa: E501

    def tearDown(self):
        pass

    def test_upload_pipeline(self):
        """Test case for upload_pipeline

        """
        pass

    def test_upload_pipeline_version(self):
        """Test case for upload_pipeline_version

        """
        pass


if __name__ == '__main__':
    unittest.main()
