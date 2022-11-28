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


class V1beta1Pipeline(object):
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
        'id': 'str',
        'created_at': 'datetime',
        'name': 'str',
        'description': 'str',
        'parameters': 'list[V1beta1Parameter]',
        'error': 'str'
    }

    attribute_map = {
        'id': 'id',
        'created_at': 'created_at',
        'name': 'name',
        'description': 'description',
        'parameters': 'parameters',
        'error': 'error'
    }

    def __init__(self, id=None, created_at=None, name=None, description=None, parameters=None, error=None, local_vars_configuration=None):  # noqa: E501
        """V1beta1Pipeline - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._id = None
        self._created_at = None
        self._name = None
        self._description = None
        self._parameters = None
        self._error = None
        self.discriminator = None

        if id is not None:
            self.id = id
        if created_at is not None:
            self.created_at = created_at
        if name is not None:
            self.name = name
        if description is not None:
            self.description = description
        if parameters is not None:
            self.parameters = parameters
        if error is not None:
            self.error = error

    @property
    def id(self):
        """Gets the id of this V1beta1Pipeline.  # noqa: E501


        :return: The id of this V1beta1Pipeline.  # noqa: E501
        :rtype: str
        """
        return self._id

    @id.setter
    def id(self, id):
        """Sets the id of this V1beta1Pipeline.


        :param id: The id of this V1beta1Pipeline.  # noqa: E501
        :type id: str
        """

        self._id = id

    @property
    def created_at(self):
        """Gets the created_at of this V1beta1Pipeline.  # noqa: E501


        :return: The created_at of this V1beta1Pipeline.  # noqa: E501
        :rtype: datetime
        """
        return self._created_at

    @created_at.setter
    def created_at(self, created_at):
        """Sets the created_at of this V1beta1Pipeline.


        :param created_at: The created_at of this V1beta1Pipeline.  # noqa: E501
        :type created_at: datetime
        """

        self._created_at = created_at

    @property
    def name(self):
        """Gets the name of this V1beta1Pipeline.  # noqa: E501


        :return: The name of this V1beta1Pipeline.  # noqa: E501
        :rtype: str
        """
        return self._name

    @name.setter
    def name(self, name):
        """Sets the name of this V1beta1Pipeline.


        :param name: The name of this V1beta1Pipeline.  # noqa: E501
        :type name: str
        """

        self._name = name

    @property
    def description(self):
        """Gets the description of this V1beta1Pipeline.  # noqa: E501


        :return: The description of this V1beta1Pipeline.  # noqa: E501
        :rtype: str
        """
        return self._description

    @description.setter
    def description(self, description):
        """Sets the description of this V1beta1Pipeline.


        :param description: The description of this V1beta1Pipeline.  # noqa: E501
        :type description: str
        """

        self._description = description

    @property
    def parameters(self):
        """Gets the parameters of this V1beta1Pipeline.  # noqa: E501


        :return: The parameters of this V1beta1Pipeline.  # noqa: E501
        :rtype: list[V1beta1Parameter]
        """
        return self._parameters

    @parameters.setter
    def parameters(self, parameters):
        """Sets the parameters of this V1beta1Pipeline.


        :param parameters: The parameters of this V1beta1Pipeline.  # noqa: E501
        :type parameters: list[V1beta1Parameter]
        """

        self._parameters = parameters

    @property
    def error(self):
        """Gets the error of this V1beta1Pipeline.  # noqa: E501

        In case any error happens retrieving a pipeline field, only pipeline ID and the error message is returned. Client has the flexibility of choosing how to handle error. This is especially useful during listing call.  # noqa: E501

        :return: The error of this V1beta1Pipeline.  # noqa: E501
        :rtype: str
        """
        return self._error

    @error.setter
    def error(self, error):
        """Sets the error of this V1beta1Pipeline.

        In case any error happens retrieving a pipeline field, only pipeline ID and the error message is returned. Client has the flexibility of choosing how to handle error. This is especially useful during listing call.  # noqa: E501

        :param error: The error of this V1beta1Pipeline.  # noqa: E501
        :type error: str
        """

        self._error = error

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
        if not isinstance(other, V1beta1Pipeline):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1beta1Pipeline):
            return True

        return self.to_dict() != other.to_dict()
