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


class V2beta1RunMetric(object):
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
        'display_name': 'str',
        'node_id': 'str',
        'number_value': 'float',
        'format': 'RunMetricFormat'
    }

    attribute_map = {
        'display_name': 'display_name',
        'node_id': 'node_id',
        'number_value': 'number_value',
        'format': 'format'
    }

    def __init__(self, display_name=None, node_id=None, number_value=None, format=None, local_vars_configuration=None):  # noqa: E501
        """V2beta1RunMetric - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._display_name = None
        self._node_id = None
        self._number_value = None
        self._format = None
        self.discriminator = None

        if display_name is not None:
            self.display_name = display_name
        if node_id is not None:
            self.node_id = node_id
        if number_value is not None:
            self.number_value = number_value
        if format is not None:
            self.format = format

    @property
    def display_name(self):
        """Gets the display_name of this V2beta1RunMetric.  # noqa: E501

        Required input. The user defined name of the metric. It must be 1-63 characters long and must conform to the following regular expression: `[a-z]([-a-z0-9]*[a-z0-9])?`.  # noqa: E501

        :return: The display_name of this V2beta1RunMetric.  # noqa: E501
        :rtype: str
        """
        return self._display_name

    @display_name.setter
    def display_name(self, display_name):
        """Sets the display_name of this V2beta1RunMetric.

        Required input. The user defined name of the metric. It must be 1-63 characters long and must conform to the following regular expression: `[a-z]([-a-z0-9]*[a-z0-9])?`.  # noqa: E501

        :param display_name: The display_name of this V2beta1RunMetric.  # noqa: E501
        :type display_name: str
        """

        self._display_name = display_name

    @property
    def node_id(self):
        """Gets the node_id of this V2beta1RunMetric.  # noqa: E501

        Required input. The runtime node ID which reports the metric. The node ID  can be found in the RunDetail.workflow.Status. Metric with same  (node_id, name) are considerd as duplicate. Only the first reporting will  be recorded. Max length is 128.  # noqa: E501

        :return: The node_id of this V2beta1RunMetric.  # noqa: E501
        :rtype: str
        """
        return self._node_id

    @node_id.setter
    def node_id(self, node_id):
        """Sets the node_id of this V2beta1RunMetric.

        Required input. The runtime node ID which reports the metric. The node ID  can be found in the RunDetail.workflow.Status. Metric with same  (node_id, name) are considerd as duplicate. Only the first reporting will  be recorded. Max length is 128.  # noqa: E501

        :param node_id: The node_id of this V2beta1RunMetric.  # noqa: E501
        :type node_id: str
        """

        self._node_id = node_id

    @property
    def number_value(self):
        """Gets the number_value of this V2beta1RunMetric.  # noqa: E501

        The number value of the metric.  # noqa: E501

        :return: The number_value of this V2beta1RunMetric.  # noqa: E501
        :rtype: float
        """
        return self._number_value

    @number_value.setter
    def number_value(self, number_value):
        """Sets the number_value of this V2beta1RunMetric.

        The number value of the metric.  # noqa: E501

        :param number_value: The number_value of this V2beta1RunMetric.  # noqa: E501
        :type number_value: float
        """

        self._number_value = number_value

    @property
    def format(self):
        """Gets the format of this V2beta1RunMetric.  # noqa: E501


        :return: The format of this V2beta1RunMetric.  # noqa: E501
        :rtype: RunMetricFormat
        """
        return self._format

    @format.setter
    def format(self, format):
        """Sets the format of this V2beta1RunMetric.


        :param format: The format of this V2beta1RunMetric.  # noqa: E501
        :type format: RunMetricFormat
        """

        self._format = format

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
        if not isinstance(other, V2beta1RunMetric):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V2beta1RunMetric):
            return True

        return self.to_dict() != other.to_dict()
