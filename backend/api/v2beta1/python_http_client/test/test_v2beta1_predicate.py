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

import kfp_server_api
from kfp_server_api.models.v2beta1_predicate import V2beta1Predicate  # noqa: E501
from kfp_server_api.rest import ApiException

class TestV2beta1Predicate(unittest.TestCase):
    """V2beta1Predicate unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test V2beta1Predicate
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_server_api.models.v2beta1_predicate.V2beta1Predicate()  # noqa: E501
        if include_optional :
            return V2beta1Predicate(
                operation = 'OPERATION_UNSPECIFIED', 
                key = '0', 
                int_value = 56, 
                long_value = '0', 
                string_value = '0', 
                timestamp_value = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                int_values = kfp_server_api.models.predicate_int_values.PredicateIntValues(
                    values = [
                        56
                        ], ), 
                long_values = kfp_server_api.models.predicate_long_values.PredicateLongValues(
                    values = [
                        '0'
                        ], ), 
                string_values = kfp_server_api.models.predicate_string_values.PredicateStringValues(
                    values = [
                        '0'
                        ], )
            )
        else :
            return V2beta1Predicate(
        )

    def testV2beta1Predicate(self):
        """Test V2beta1Predicate"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
