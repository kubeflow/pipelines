# Copyright 2020 Google LLC
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

# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.

    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


import pprint
import re  # noqa: F401

import six

from kfp_server_api.configuration import Configuration


class ApiPeriodicSchedule(object):
    """NOTE: This class is auto generated by OpenAPI Generator.
    Ref: https://openapi-generator.tech

    Do not edit the class manually.
    """

    """
    Attributes:
      openapi_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    openapi_types = {
        'start_time': 'datetime',
        'end_time': 'datetime',
        'interval_second': 'str'
    }

    attribute_map = {
        'start_time': 'start_time',
        'end_time': 'end_time',
        'interval_second': 'interval_second'
    }

    def __init__(self, start_time=None, end_time=None, interval_second=None, local_vars_configuration=None):  # noqa: E501
        """ApiPeriodicSchedule - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._start_time = None
        self._end_time = None
        self._interval_second = None
        self.discriminator = None

        if start_time is not None:
            self.start_time = start_time
        if end_time is not None:
            self.end_time = end_time
        if interval_second is not None:
            self.interval_second = interval_second

    @property
    def start_time(self):
        """Gets the start_time of this ApiPeriodicSchedule.  # noqa: E501


        :return: The start_time of this ApiPeriodicSchedule.  # noqa: E501
        :rtype: datetime
        """
        return self._start_time

    @start_time.setter
    def start_time(self, start_time):
        """Sets the start_time of this ApiPeriodicSchedule.


        :param start_time: The start_time of this ApiPeriodicSchedule.  # noqa: E501
        :type: datetime
        """

        self._start_time = start_time

    @property
    def end_time(self):
        """Gets the end_time of this ApiPeriodicSchedule.  # noqa: E501


        :return: The end_time of this ApiPeriodicSchedule.  # noqa: E501
        :rtype: datetime
        """
        return self._end_time

    @end_time.setter
    def end_time(self, end_time):
        """Sets the end_time of this ApiPeriodicSchedule.


        :param end_time: The end_time of this ApiPeriodicSchedule.  # noqa: E501
        :type: datetime
        """

        self._end_time = end_time

    @property
    def interval_second(self):
        """Gets the interval_second of this ApiPeriodicSchedule.  # noqa: E501


        :return: The interval_second of this ApiPeriodicSchedule.  # noqa: E501
        :rtype: str
        """
        return self._interval_second

    @interval_second.setter
    def interval_second(self, interval_second):
        """Sets the interval_second of this ApiPeriodicSchedule.


        :param interval_second: The interval_second of this ApiPeriodicSchedule.  # noqa: E501
        :type: str
        """

        self._interval_second = interval_second

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.openapi_types):
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

        return result

    def to_str(self):
        """Returns the string representation of the model"""
        return pprint.pformat(self.to_dict())

    def __repr__(self):
        """For `print` and `pprint`"""
        return self.to_str()

    def __eq__(self, other):
        """Returns true if both objects are equal"""
        if not isinstance(other, ApiPeriodicSchedule):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, ApiPeriodicSchedule):
            return True

        return self.to_dict() != other.to_dict()
