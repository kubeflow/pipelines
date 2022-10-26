# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.

    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import unittest
import datetime

import kfp_server_api_v1beta1
from kfp_server_api_v1beta1.models.v1beta1_run_storage_state import V1beta1RunStorageState  # noqa: E501
from kfp_server_api_v1beta1.rest import ApiException

class TestV1beta1RunStorageState(unittest.TestCase):
    """V1beta1RunStorageState unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V1beta1RunStorageState
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_server_api_v1beta1.models.v1beta1_run_storage_state.V1beta1RunStorageState()  # noqa: E501
        if include_optional :
            return V1beta1RunStorageState(
            )
        else :
            return V1beta1RunStorageState(
        )

    def testV1beta1RunStorageState(self):
        """Test V1beta1RunStorageState"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
