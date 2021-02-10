import argparse
import datetime
from distutils.util import strtobool
import logging
import yaml

from kubernetes import client as k8s_client
from kubernetes import config

import launch_crd
from kubeflow.pytorchjob import V1PyTorchJob as V1PyTorchJob_original
from kubeflow.pytorchjob import V1PyTorchJobSpec as V1PyTorchJobSpec_original

def yamlOrJsonStr(str):
    if str == "" or str == None:
        return None
    return yaml.safe_load(str)

def get_current_namespace():
    current_namespace = open("/var/run/secrets/kubernetes.io/serviceaccount/namespace").read()
    return current_namespace

# # TODO: Needed if I just call k8s_client.ApiClient().sanitize_for_serialization() below?
# # Patch existing classes to fix API bugs
# import six
# def to_dict(self):
#     """Returns the model properties as a dict, respecting attribute_map renaming"""
#     result = {}

#     for attr, _ in six.iteritems(self.swagger_types):
#         if attr in self.attribute_map:
#             returned_attr = self.attribute_map[attr]
#         else:
#             returned_attr = attr

#         value = getattr(self, attr)
#         if isinstance(value, list):
#             result[returned_attr] = list(map(
#                 lambda x: x.to_dict() if hasattr(x, "to_dict") else x,
#                 value
#             ))
#         elif hasattr(value, "to_dict"):
#             result[returned_attr] = value.to_dict()
#         elif isinstance(value, dict):
#             result[returned_attr] = dict(map(
#                 lambda item: (item[0], item[1].to_dict())
#                 if hasattr(item[1], "to_dict") else item,
#                 value.items()
#             ))
#         else:
#             result[returned_attr] = value
#     # Class specific if.  Why do we need this?
# #         if issubclass(THIS_CLASS, dict):
# #             for key, value in self.items():
# #                 result[key] = value

#     return result

# Patch PyTorchJob APIs to align with k8s usage
class V1PyTorchJob(V1PyTorchJob_original):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.openapi_types = self.swagger_types

    # def to_dict(self):
    #     return to_dict(self)

class V1PyTorchJobSpec(V1PyTorchJobSpec_original):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.openapi_types = self.swagger_types

    # def to_dict(self):
        # return to_dict(self)

def main(argv=None):
    parser = argparse.ArgumentParser(description='Kubeflow Job launcher')
    parser.add_argument('--name', type=str,
                        help='Job name.')
    # TODO: Make default namespace the current namespace, not kubeflow?
    parser.add_argument('--namespace', type=str,
                        default=get_current_namespace(),
                        help='Job namespace.')
    parser.add_argument('--version', type=str,
                        default='v1',
                        help='Job version.')
    parser.add_argument('--activeDeadlineSeconds', type=int,
                        default=None,
                        help='Specifies the duration (in seconds) since startTime during which the job can remain active before it is terminated. Must be a positive integer. This setting applies only to pods where restartPolicy is OnFailure or Always.')
    parser.add_argument('--backoffLimit', type=int,
                        default=None,
                        help='Number of retries before marking this job as failed.')
    parser.add_argument('--cleanPodPolicy', type=str,
                        default="Running",
                        help='Defines the policy for cleaning up pods after the Job completes.')
    parser.add_argument('--ttlSecondsAfterFinished', type=int,
                        default=None,
                        help='Defines the TTL for cleaning up finished Jobs.')
    # TODO: Add k8s_client.ApiClient().sanitize_for_serialization(obj) to convert k8s api objects to k8s yaml
    parser.add_argument('--masterSpec', type=yamlOrJsonStr,
                        default={},
                        help='Job master replicaSpecs.')
    parser.add_argument('--workerSpec', type=yamlOrJsonStr,
                        default={},
                        help='Job worker replicaSpecs.')
    parser.add_argument('--deleteAfterDone', type=strtobool,
                        default=True,
                        help='When Job done, delete the Job automatically if it is True.')
    parser.add_argument('--jobTimeoutMinutes', type=int,
                        default=60*24,
                        help='Time in minutes to wait for the Job to reach end')

    # Options that likely wont be used, but left here for future use
    parser.add_argument('--jobGroup', type=str,
                        default="kubeflow.org",
                        help='Group for the CRD, ex: kubeflow.org')
    parser.add_argument('--jobPlural', type=str,
                        default="pytorchjobs",  # We could select a launcher here and populate these automatically
                        help='Plural name for the CRD, ex: pytorchjobs')
    parser.add_argument('--kind', type=str,
                        default='PyTorchJob',
                        help='CRD kind.')
    

    args = parser.parse_args()

    logging.getLogger(__name__).setLevel(logging.INFO)

    logging.info('Generating job template.')

    jobSpec = V1PyTorchJobSpec(
        pytorch_replica_specs={
            'Master': args.masterSpec,
            'Worker': args.workerSpec,
        },
        active_deadline_seconds=args.activeDeadlineSeconds,
        backoff_limit=args.backoffLimit,
        clean_pod_policy=args.cleanPodPolicy,
        ttl_seconds_after_finished=args.ttlSecondsAfterFinished,
    )

    api_version = f"{args.jobGroup}/{args.version}"

    job = V1PyTorchJob(
        api_version=api_version,
        kind=args.kind,
        metadata=k8s_client.V1ObjectMeta(
            name=args.name,
            namespace=args.namespace,
        ),
        spec=jobSpec,
    )

    serialized_job = k8s_client.ApiClient().sanitize_for_serialization(job)

    logging.info('Creating launcher client.')

    config.load_incluster_config()
    api_client = k8s_client.ApiClient()
    launcher_client = launch_crd.K8sCR(group=args.jobGroup, plural=args.jobPlural, version=args.version, client=api_client)
 
    logging.info('Submitting CR.')
    create_response = launcher_client.create(serialized_job)

    expected_conditions = ["Succeeded", "Failed"]
    logging.info(f'Monitoring job until status is any of {expected_conditions}.')
    launcher_client.wait_for_condition(
        args.namespace, args.name, expected_conditions,
        timeout=datetime.timedelta(minutes=args.jobTimeoutMinutes))
    if args.deleteAfterDone:
        logging.info(f'Deleting job.')
        launcher_client.delete(args.name, args.namespace)

if __name__== "__main__":
    main()
