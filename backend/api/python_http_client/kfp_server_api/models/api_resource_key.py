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


class ApiResourceKey(object):
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
        'type': 'ApiResourceType',
        'id': 'str'
    }

    attribute_map = {
        'type': 'type',
        'id': 'id'
    }

    def __init__(self, type=None, id=None, local_vars_configuration=None):  # noqa: E501
        """ApiResourceKey - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._type = None
        self._id = None
        self.discriminator = None

        if type is not None:
            self.type = type
        if id is not None:
            self.id = id

    @property
    def type(self):
        """Gets the type of this ApiResourceKey.  # noqa: E501


        :return: The type of this ApiResourceKey.  # noqa: E501
        :rtype: ApiResourceType
        """
        return self._type

    @type.setter
    def type(self, type):
        """Sets the type of this ApiResourceKey.


        :param type: The type of this ApiResourceKey.  # noqa: E501
        :type type: ApiResourceType
        """

        self._type = type

    @property
    def id(self):
        """Gets the id of this ApiResourceKey.  # noqa: E501

        The ID of the resource that referred to.  # noqa: E501

        :return: The id of this ApiResourceKey.  # noqa: E501
        :rtype: str
        """
        return self._id

    @id.setter
    def id(self, id):
        """Sets the id of this ApiResourceKey.

        The ID of the resource that referred to.  # noqa: E501

        :param id: The id of this ApiResourceKey.  # noqa: E501
        :type id: str
        """

        self._id = id

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
        if not isinstance(other, ApiResourceKey):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, ApiResourceKey):
            return True

        return self.to_dict() != other.to_dict()
