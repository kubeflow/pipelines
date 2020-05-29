# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.  # noqa: E501

    OpenAPI spec version: 1.0.0-dev.1
    
    Generated by: https://github.com/swagger-api/swagger-codegen.git
"""


import pprint
import re  # noqa: F401

import six

from kfp_server_api.models.api_cron_schedule import ApiCronSchedule  # noqa: F401,E501
from kfp_server_api.models.api_periodic_schedule import ApiPeriodicSchedule  # noqa: F401,E501


class ApiTrigger(object):
    """NOTE: This class is auto generated by the swagger code generator program.

    Do not edit the class manually.
    """

    """
    Attributes:
      swagger_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    swagger_types = {
        'cron_schedule': 'ApiCronSchedule',
        'periodic_schedule': 'ApiPeriodicSchedule'
    }

    attribute_map = {
        'cron_schedule': 'cron_schedule',
        'periodic_schedule': 'periodic_schedule'
    }

    def __init__(self, cron_schedule=None, periodic_schedule=None):  # noqa: E501
        """ApiTrigger - a model defined in Swagger"""  # noqa: E501

        self._cron_schedule = None
        self._periodic_schedule = None
        self.discriminator = None

        if cron_schedule is not None:
            self.cron_schedule = cron_schedule
        if periodic_schedule is not None:
            self.periodic_schedule = periodic_schedule

    @property
    def cron_schedule(self):
        """Gets the cron_schedule of this ApiTrigger.  # noqa: E501


        :return: The cron_schedule of this ApiTrigger.  # noqa: E501
        :rtype: ApiCronSchedule
        """
        return self._cron_schedule

    @cron_schedule.setter
    def cron_schedule(self, cron_schedule):
        """Sets the cron_schedule of this ApiTrigger.


        :param cron_schedule: The cron_schedule of this ApiTrigger.  # noqa: E501
        :type: ApiCronSchedule
        """

        self._cron_schedule = cron_schedule

    @property
    def periodic_schedule(self):
        """Gets the periodic_schedule of this ApiTrigger.  # noqa: E501


        :return: The periodic_schedule of this ApiTrigger.  # noqa: E501
        :rtype: ApiPeriodicSchedule
        """
        return self._periodic_schedule

    @periodic_schedule.setter
    def periodic_schedule(self, periodic_schedule):
        """Sets the periodic_schedule of this ApiTrigger.


        :param periodic_schedule: The periodic_schedule of this ApiTrigger.  # noqa: E501
        :type: ApiPeriodicSchedule
        """

        self._periodic_schedule = periodic_schedule

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.swagger_types):
            value = getattr(self, attr)
            if isinstance(value, list):
                result[attr] = list(map(
                    lambda x: x.to_dict() if hasattr(x, "to_dict") else x,
                    value
                ))
            elif hasattr(value, "to_dict"):
                result[attr] = value.to_dict()
            elif isinstance(value, dict):
                result[attr] = dict(map(
                    lambda item: (item[0], item[1].to_dict())
                    if hasattr(item[1], "to_dict") else item,
                    value.items()
                ))
            else:
                result[attr] = value
        if issubclass(ApiTrigger, dict):
            for key, value in self.items():
                result[key] = value

        return result

    def to_str(self):
        """Returns the string representation of the model"""
        return pprint.pformat(self.to_dict())

    def __repr__(self):
        """For `print` and `pprint`"""
        return self.to_str()

    def __eq__(self, other):
        """Returns true if both objects are equal"""
        if not isinstance(other, ApiTrigger):
            return False

        return self.__dict__ == other.__dict__

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        return not self == other
