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

from kfp_server_api_v1beta1.configuration import Configuration


class V1beta1ListJobsResponse(object):
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
        'jobs': 'list[V1beta1Job]',
        'total_size': 'int',
        'next_page_token': 'str'
    }

    attribute_map = {
        'jobs': 'jobs',
        'total_size': 'total_size',
        'next_page_token': 'next_page_token'
    }

    def __init__(self, jobs=None, total_size=None, next_page_token=None, local_vars_configuration=None):  # noqa: E501
        """V1beta1ListJobsResponse - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._jobs = None
        self._total_size = None
        self._next_page_token = None
        self.discriminator = None

        if jobs is not None:
            self.jobs = jobs
        if total_size is not None:
            self.total_size = total_size
        if next_page_token is not None:
            self.next_page_token = next_page_token

    @property
    def jobs(self):
        """Gets the jobs of this V1beta1ListJobsResponse.  # noqa: E501

        A list of jobs returned.  # noqa: E501

        :return: The jobs of this V1beta1ListJobsResponse.  # noqa: E501
        :rtype: list[V1beta1Job]
        """
        return self._jobs

    @jobs.setter
    def jobs(self, jobs):
        """Sets the jobs of this V1beta1ListJobsResponse.

        A list of jobs returned.  # noqa: E501

        :param jobs: The jobs of this V1beta1ListJobsResponse.  # noqa: E501
        :type jobs: list[V1beta1Job]
        """

        self._jobs = jobs

    @property
    def total_size(self):
        """Gets the total_size of this V1beta1ListJobsResponse.  # noqa: E501

        The total number of jobs for the given query.  # noqa: E501

        :return: The total_size of this V1beta1ListJobsResponse.  # noqa: E501
        :rtype: int
        """
        return self._total_size

    @total_size.setter
    def total_size(self, total_size):
        """Sets the total_size of this V1beta1ListJobsResponse.

        The total number of jobs for the given query.  # noqa: E501

        :param total_size: The total_size of this V1beta1ListJobsResponse.  # noqa: E501
        :type total_size: int
        """

        self._total_size = total_size

    @property
    def next_page_token(self):
        """Gets the next_page_token of this V1beta1ListJobsResponse.  # noqa: E501

        The token to list the next page of jobs.  # noqa: E501

        :return: The next_page_token of this V1beta1ListJobsResponse.  # noqa: E501
        :rtype: str
        """
        return self._next_page_token

    @next_page_token.setter
    def next_page_token(self, next_page_token):
        """Sets the next_page_token of this V1beta1ListJobsResponse.

        The token to list the next page of jobs.  # noqa: E501

        :param next_page_token: The next_page_token of this V1beta1ListJobsResponse.  # noqa: E501
        :type next_page_token: str
        """

        self._next_page_token = next_page_token

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
        if not isinstance(other, V1beta1ListJobsResponse):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, V1beta1ListJobsResponse):
            return True

        return self.to_dict() != other.to_dict()
