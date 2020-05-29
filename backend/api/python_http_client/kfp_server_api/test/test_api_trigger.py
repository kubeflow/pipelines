# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.  # noqa: E501

    The version of the OpenAPI document: 1.0.0-dev.1
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import unittest
import datetime

import kfp_server_api
from kfp_server_api.models.api_trigger import ApiTrigger  # noqa: E501
from kfp_server_api.rest import ApiException

class TestApiTrigger(unittest.TestCase):
    """ApiTrigger unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test ApiTrigger
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = kfp_server_api.models.api_trigger.ApiTrigger()  # noqa: E501
        if include_optional :
            return ApiTrigger(
                cron_schedule = kfp_server_api.models.cron_schedule_allow_scheduling_the_job_with_unix_like_cron.CronSchedule allow scheduling the job with unix-like cron(
                    start_time = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                    end_time = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                    cron = '0', ), 
                periodic_schedule = kfp_server_api.models.periodic_schedule_allow_scheduling_the_job_periodically_with_certain_interval.PeriodicSchedule allow scheduling the job periodically with certain interval(
                    start_time = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                    end_time = datetime.datetime.strptime('2013-10-20 19:20:30.00', '%Y-%m-%d %H:%M:%S.%f'), 
                    interval_second = '0', )
            )
        else :
            return ApiTrigger(
        )

    def testApiTrigger(self):
        """Test ApiTrigger"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
