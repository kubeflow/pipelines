# Copyright 2021 Google LLC
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


class ApiPipeline(object):
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
        'parameters': 'list[ApiParameter]',
        'url': 'ApiUrl',
        'error': 'str',
        'default_version': 'ApiPipelineVersion'
    }

    attribute_map = {
        'id': 'id',
        'created_at': 'created_at',
        'name': 'name',
        'description': 'description',
        'parameters': 'parameters',
        'url': 'url',
        'error': 'error',
        'default_version': 'default_version'
    }

    def __init__(self, id=None, created_at=None, name=None, description=None, parameters=None, url=None, error=None, default_version=None, local_vars_configuration=None):  # noqa: E501
        """ApiPipeline - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._id = None
        self._created_at = None
        self._name = None
        self._description = None
        self._parameters = None
        self._url = None
        self._error = None
        self._default_version = None
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
        if url is not None:
            self.url = url
        if error is not None:
            self.error = error
        if default_version is not None:
            self.default_version = default_version

    @property
    def id(self):
        """Gets the id of this ApiPipeline.  # noqa: E501

        Output. Unique pipeline ID. Generated by API server.  # noqa: E501

        :return: The id of this ApiPipeline.  # noqa: E501
        :rtype: str
        """
        return self._id

    @id.setter
    def id(self, id):
        """Sets the id of this ApiPipeline.

        Output. Unique pipeline ID. Generated by API server.  # noqa: E501

        :param id: The id of this ApiPipeline.  # noqa: E501
        :type id: str
        """

        self._id = id

    @property
    def created_at(self):
        """Gets the created_at of this ApiPipeline.  # noqa: E501

        Output. The time this pipeline is created.  # noqa: E501

        :return: The created_at of this ApiPipeline.  # noqa: E501
        :rtype: datetime
        """
        return self._created_at

    @created_at.setter
    def created_at(self, created_at):
        """Sets the created_at of this ApiPipeline.

        Output. The time this pipeline is created.  # noqa: E501

        :param created_at: The created_at of this ApiPipeline.  # noqa: E501
        :type created_at: datetime
        """

        self._created_at = created_at

    @property
    def name(self):
        """Gets the name of this ApiPipeline.  # noqa: E501

        Optional input field. Pipeline name provided by user. If not specified, file name is used as pipeline name.  # noqa: E501

        :return: The name of this ApiPipeline.  # noqa: E501
        :rtype: str
        """
        return self._name

    @name.setter
    def name(self, name):
        """Sets the name of this ApiPipeline.

        Optional input field. Pipeline name provided by user. If not specified, file name is used as pipeline name.  # noqa: E501

        :param name: The name of this ApiPipeline.  # noqa: E501
        :type name: str
        """

        self._name = name

    @property
    def description(self):
        """Gets the description of this ApiPipeline.  # noqa: E501

        Optional input field. Describing the purpose of the job.  # noqa: E501

        :return: The description of this ApiPipeline.  # noqa: E501
        :rtype: str
        """
        return self._description

    @description.setter
    def description(self, description):
        """Sets the description of this ApiPipeline.

        Optional input field. Describing the purpose of the job.  # noqa: E501

        :param description: The description of this ApiPipeline.  # noqa: E501
        :type description: str
        """

        self._description = description

    @property
    def parameters(self):
        """Gets the parameters of this ApiPipeline.  # noqa: E501

        Output. The input parameters for this pipeline. TODO(jingzhang36): replace this parameters field with the parameters field inside PipelineVersion when all usage of the former has been changed to use the latter.  # noqa: E501

        :return: The parameters of this ApiPipeline.  # noqa: E501
        :rtype: list[ApiParameter]
        """
        return self._parameters

    @parameters.setter
    def parameters(self, parameters):
        """Sets the parameters of this ApiPipeline.

        Output. The input parameters for this pipeline. TODO(jingzhang36): replace this parameters field with the parameters field inside PipelineVersion when all usage of the former has been changed to use the latter.  # noqa: E501

        :param parameters: The parameters of this ApiPipeline.  # noqa: E501
        :type parameters: list[ApiParameter]
        """

        self._parameters = parameters

    @property
    def url(self):
        """Gets the url of this ApiPipeline.  # noqa: E501


        :return: The url of this ApiPipeline.  # noqa: E501
        :rtype: ApiUrl
        """
        return self._url

    @url.setter
    def url(self, url):
        """Sets the url of this ApiPipeline.


        :param url: The url of this ApiPipeline.  # noqa: E501
        :type url: ApiUrl
        """

        self._url = url

    @property
    def error(self):
        """Gets the error of this ApiPipeline.  # noqa: E501

        In case any error happens retrieving a pipeline field, only pipeline ID and the error message is returned. Client has the flexibility of choosing how to handle error. This is especially useful during listing call.  # noqa: E501

        :return: The error of this ApiPipeline.  # noqa: E501
        :rtype: str
        """
        return self._error

    @error.setter
    def error(self, error):
        """Sets the error of this ApiPipeline.

        In case any error happens retrieving a pipeline field, only pipeline ID and the error message is returned. Client has the flexibility of choosing how to handle error. This is especially useful during listing call.  # noqa: E501

        :param error: The error of this ApiPipeline.  # noqa: E501
        :type error: str
        """

        self._error = error

    @property
    def default_version(self):
        """Gets the default_version of this ApiPipeline.  # noqa: E501


        :return: The default_version of this ApiPipeline.  # noqa: E501
        :rtype: ApiPipelineVersion
        """
        return self._default_version

    @default_version.setter
    def default_version(self, default_version):
        """Sets the default_version of this ApiPipeline.


        :param default_version: The default_version of this ApiPipeline.  # noqa: E501
        :type default_version: ApiPipelineVersion
        """

        self._default_version = default_version

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
        if not isinstance(other, ApiPipeline):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, ApiPipeline):
            return True

        return self.to_dict() != other.to_dict()
