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


class ApiPipelineVersion(object):
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
        'name': 'str',
        'created_at': 'datetime',
        'parameters': 'list[ApiParameter]',
        'code_source_url': 'str',
        'package_url': 'ApiUrl',
        'resource_references': 'list[ApiResourceReference]'
    }

    attribute_map = {
        'id': 'id',
        'name': 'name',
        'created_at': 'created_at',
        'parameters': 'parameters',
        'code_source_url': 'code_source_url',
        'package_url': 'package_url',
        'resource_references': 'resource_references'
    }

    def __init__(self, id=None, name=None, created_at=None, parameters=None, code_source_url=None, package_url=None, resource_references=None, local_vars_configuration=None):  # noqa: E501
        """ApiPipelineVersion - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._id = None
        self._name = None
        self._created_at = None
        self._parameters = None
        self._code_source_url = None
        self._package_url = None
        self._resource_references = None
        self.discriminator = None

        if id is not None:
            self.id = id
        if name is not None:
            self.name = name
        if created_at is not None:
            self.created_at = created_at
        if parameters is not None:
            self.parameters = parameters
        if code_source_url is not None:
            self.code_source_url = code_source_url
        if package_url is not None:
            self.package_url = package_url
        if resource_references is not None:
            self.resource_references = resource_references

    @property
    def id(self):
        """Gets the id of this ApiPipelineVersion.  # noqa: E501

        Output. Unique version ID. Generated by API server.  # noqa: E501

        :return: The id of this ApiPipelineVersion.  # noqa: E501
        :rtype: str
        """
        return self._id

    @id.setter
    def id(self, id):
        """Sets the id of this ApiPipelineVersion.

        Output. Unique version ID. Generated by API server.  # noqa: E501

        :param id: The id of this ApiPipelineVersion.  # noqa: E501
        :type id: str
        """

        self._id = id

    @property
    def name(self):
        """Gets the name of this ApiPipelineVersion.  # noqa: E501

        Optional input field. Version name provided by user.  # noqa: E501

        :return: The name of this ApiPipelineVersion.  # noqa: E501
        :rtype: str
        """
        return self._name

    @name.setter
    def name(self, name):
        """Sets the name of this ApiPipelineVersion.

        Optional input field. Version name provided by user.  # noqa: E501

        :param name: The name of this ApiPipelineVersion.  # noqa: E501
        :type name: str
        """

        self._name = name

    @property
    def created_at(self):
        """Gets the created_at of this ApiPipelineVersion.  # noqa: E501

        Output. The time this pipeline version is created.  # noqa: E501

        :return: The created_at of this ApiPipelineVersion.  # noqa: E501
        :rtype: datetime
        """
        return self._created_at

    @created_at.setter
    def created_at(self, created_at):
        """Sets the created_at of this ApiPipelineVersion.

        Output. The time this pipeline version is created.  # noqa: E501

        :param created_at: The created_at of this ApiPipelineVersion.  # noqa: E501
        :type created_at: datetime
        """

        self._created_at = created_at

    @property
    def parameters(self):
        """Gets the parameters of this ApiPipelineVersion.  # noqa: E501

        Output. The input parameters for this pipeline.  # noqa: E501

        :return: The parameters of this ApiPipelineVersion.  # noqa: E501
        :rtype: list[ApiParameter]
        """
        return self._parameters

    @parameters.setter
    def parameters(self, parameters):
        """Sets the parameters of this ApiPipelineVersion.

        Output. The input parameters for this pipeline.  # noqa: E501

        :param parameters: The parameters of this ApiPipelineVersion.  # noqa: E501
        :type parameters: list[ApiParameter]
        """

        self._parameters = parameters

    @property
    def code_source_url(self):
        """Gets the code_source_url of this ApiPipelineVersion.  # noqa: E501

        Input. Optional. Pipeline version code source.  # noqa: E501

        :return: The code_source_url of this ApiPipelineVersion.  # noqa: E501
        :rtype: str
        """
        return self._code_source_url

    @code_source_url.setter
    def code_source_url(self, code_source_url):
        """Sets the code_source_url of this ApiPipelineVersion.

        Input. Optional. Pipeline version code source.  # noqa: E501

        :param code_source_url: The code_source_url of this ApiPipelineVersion.  # noqa: E501
        :type code_source_url: str
        """

        self._code_source_url = code_source_url

    @property
    def package_url(self):
        """Gets the package_url of this ApiPipelineVersion.  # noqa: E501


        :return: The package_url of this ApiPipelineVersion.  # noqa: E501
        :rtype: ApiUrl
        """
        return self._package_url

    @package_url.setter
    def package_url(self, package_url):
        """Sets the package_url of this ApiPipelineVersion.


        :param package_url: The package_url of this ApiPipelineVersion.  # noqa: E501
        :type package_url: ApiUrl
        """

        self._package_url = package_url

    @property
    def resource_references(self):
        """Gets the resource_references of this ApiPipelineVersion.  # noqa: E501

        Input. Required. E.g., specify which pipeline this pipeline version belongs to.  # noqa: E501

        :return: The resource_references of this ApiPipelineVersion.  # noqa: E501
        :rtype: list[ApiResourceReference]
        """
        return self._resource_references

    @resource_references.setter
    def resource_references(self, resource_references):
        """Sets the resource_references of this ApiPipelineVersion.

        Input. Required. E.g., specify which pipeline this pipeline version belongs to.  # noqa: E501

        :param resource_references: The resource_references of this ApiPipelineVersion.  # noqa: E501
        :type resource_references: list[ApiResourceReference]
        """

        self._resource_references = resource_references

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
        if not isinstance(other, ApiPipelineVersion):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, ApiPipelineVersion):
            return True

        return self.to_dict() != other.to_dict()
