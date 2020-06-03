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

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.  # noqa: E501

    The version of the OpenAPI document: 1.0.0-dev.1
    Contact: kubeflow-pipelines@google.com
    Generated by: https://openapi-generator.tech
"""


import pprint
import re  # noqa: F401

import six

from kfp_server_api.configuration import Configuration


class ApiReportRunMetricsRequest(object):
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
        'run_id': 'str',
        'metrics': 'list[ApiRunMetric]'
    }

    attribute_map = {
        'run_id': 'run_id',
        'metrics': 'metrics'
    }

    def __init__(self, run_id=None, metrics=None, local_vars_configuration=None):  # noqa: E501
        """ApiReportRunMetricsRequest - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._run_id = None
        self._metrics = None
        self.discriminator = None

        if run_id is not None:
            self.run_id = run_id
        if metrics is not None:
            self.metrics = metrics

    @property
    def run_id(self):
        """Gets the run_id of this ApiReportRunMetricsRequest.  # noqa: E501

        Required. The parent run ID of the metric.  # noqa: E501

        :return: The run_id of this ApiReportRunMetricsRequest.  # noqa: E501
        :rtype: str
        """
        return self._run_id

    @run_id.setter
    def run_id(self, run_id):
        """Sets the run_id of this ApiReportRunMetricsRequest.

        Required. The parent run ID of the metric.  # noqa: E501

        :param run_id: The run_id of this ApiReportRunMetricsRequest.  # noqa: E501
        :type: str
        """

        self._run_id = run_id

    @property
    def metrics(self):
        """Gets the metrics of this ApiReportRunMetricsRequest.  # noqa: E501

        List of metrics to report.  # noqa: E501

        :return: The metrics of this ApiReportRunMetricsRequest.  # noqa: E501
        :rtype: list[ApiRunMetric]
        """
        return self._metrics

    @metrics.setter
    def metrics(self, metrics):
        """Sets the metrics of this ApiReportRunMetricsRequest.

        List of metrics to report.  # noqa: E501

        :param metrics: The metrics of this ApiReportRunMetricsRequest.  # noqa: E501
        :type: list[ApiRunMetric]
        """

        self._metrics = metrics

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
        if not isinstance(other, ApiReportRunMetricsRequest):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, ApiReportRunMetricsRequest):
            return True

        return self.to_dict() != other.to_dict()
