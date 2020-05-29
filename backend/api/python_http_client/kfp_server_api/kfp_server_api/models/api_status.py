# coding: utf-8

"""
    Kubeflow Pipelines API

    This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition.  # noqa: E501

    The version of the OpenAPI document: 1.0.0-dev.1
    Generated by: https://openapi-generator.tech
"""


import pprint
import re  # noqa: F401

import six

from kfp_server_api.configuration import Configuration


class ApiStatus(object):
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
        'error': 'str',
        'code': 'int',
        'details': 'list[ProtobufAny]'
    }

    attribute_map = {
        'error': 'error',
        'code': 'code',
        'details': 'details'
    }

    def __init__(self, error=None, code=None, details=None, local_vars_configuration=None):  # noqa: E501
        """ApiStatus - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._error = None
        self._code = None
        self._details = None
        self.discriminator = None

        if error is not None:
            self.error = error
        if code is not None:
            self.code = code
        if details is not None:
            self.details = details

    @property
    def error(self):
        """Gets the error of this ApiStatus.  # noqa: E501


        :return: The error of this ApiStatus.  # noqa: E501
        :rtype: str
        """
        return self._error

    @error.setter
    def error(self, error):
        """Sets the error of this ApiStatus.


        :param error: The error of this ApiStatus.  # noqa: E501
        :type: str
        """

        self._error = error

    @property
    def code(self):
        """Gets the code of this ApiStatus.  # noqa: E501


        :return: The code of this ApiStatus.  # noqa: E501
        :rtype: int
        """
        return self._code

    @code.setter
    def code(self, code):
        """Sets the code of this ApiStatus.


        :param code: The code of this ApiStatus.  # noqa: E501
        :type: int
        """

        self._code = code

    @property
    def details(self):
        """Gets the details of this ApiStatus.  # noqa: E501


        :return: The details of this ApiStatus.  # noqa: E501
        :rtype: list[ProtobufAny]
        """
        return self._details

    @details.setter
    def details(self, details):
        """Sets the details of this ApiStatus.


        :param details: The details of this ApiStatus.  # noqa: E501
        :type: list[ProtobufAny]
        """

        self._details = details

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
        if not isinstance(other, ApiStatus):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, ApiStatus):
            return True

        return self.to_dict() != other.to_dict()
