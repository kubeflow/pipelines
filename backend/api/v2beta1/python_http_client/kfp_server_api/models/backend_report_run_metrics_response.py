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


class BackendReportRunMetricsResponse(object):
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
        'metric_name': 'str',
        'metric_node_id': 'str',
        'status': 'ReportRunMetricsResponseMetricStatus',
        'message': 'str'
    }

    attribute_map = {
        'metric_name': 'metric_name',
        'metric_node_id': 'metric_node_id',
        'status': 'status',
        'message': 'message'
    }

    def __init__(self, metric_name=None, metric_node_id=None, status=None, message=None, local_vars_configuration=None):  # noqa: E501
        """BackendReportRunMetricsResponse - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._metric_name = None
        self._metric_node_id = None
        self._status = None
        self._message = None
        self.discriminator = None

        if metric_name is not None:
            self.metric_name = metric_name
        if metric_node_id is not None:
            self.metric_node_id = metric_node_id
        if status is not None:
            self.status = status
        if message is not None:
            self.message = message

    @property
    def metric_name(self):
        """Gets the metric_name of this BackendReportRunMetricsResponse.  # noqa: E501

        Output. The name of the metric.  # noqa: E501

        :return: The metric_name of this BackendReportRunMetricsResponse.  # noqa: E501
        :rtype: str
        """
        return self._metric_name

    @metric_name.setter
    def metric_name(self, metric_name):
        """Sets the metric_name of this BackendReportRunMetricsResponse.

        Output. The name of the metric.  # noqa: E501

        :param metric_name: The metric_name of this BackendReportRunMetricsResponse.  # noqa: E501
        :type metric_name: str
        """

        self._metric_name = metric_name

    @property
    def metric_node_id(self):
        """Gets the metric_node_id of this BackendReportRunMetricsResponse.  # noqa: E501

        Output. The ID of the node which reports the metric.  # noqa: E501

        :return: The metric_node_id of this BackendReportRunMetricsResponse.  # noqa: E501
        :rtype: str
        """
        return self._metric_node_id

    @metric_node_id.setter
    def metric_node_id(self, metric_node_id):
        """Sets the metric_node_id of this BackendReportRunMetricsResponse.

        Output. The ID of the node which reports the metric.  # noqa: E501

        :param metric_node_id: The metric_node_id of this BackendReportRunMetricsResponse.  # noqa: E501
        :type metric_node_id: str
        """

        self._metric_node_id = metric_node_id

    @property
    def status(self):
        """Gets the status of this BackendReportRunMetricsResponse.  # noqa: E501


        :return: The status of this BackendReportRunMetricsResponse.  # noqa: E501
        :rtype: ReportRunMetricsResponseMetricStatus
        """
        return self._status

    @status.setter
    def status(self, status):
        """Sets the status of this BackendReportRunMetricsResponse.


        :param status: The status of this BackendReportRunMetricsResponse.  # noqa: E501
        :type status: ReportRunMetricsResponseMetricStatus
        """

        self._status = status

    @property
    def message(self):
        """Gets the message of this BackendReportRunMetricsResponse.  # noqa: E501

        Output. The detailed message of the error of the reporting.  # noqa: E501

        :return: The message of this BackendReportRunMetricsResponse.  # noqa: E501
        :rtype: str
        """
        return self._message

    @message.setter
    def message(self, message):
        """Sets the message of this BackendReportRunMetricsResponse.

        Output. The detailed message of the error of the reporting.  # noqa: E501

        :param message: The message of this BackendReportRunMetricsResponse.  # noqa: E501
        :type message: str
        """

        self._message = message

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
        if not isinstance(other, BackendReportRunMetricsResponse):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, BackendReportRunMetricsResponse):
            return True

        return self.to_dict() != other.to_dict()
