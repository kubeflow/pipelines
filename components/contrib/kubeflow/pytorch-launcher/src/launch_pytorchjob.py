import argparse
import datetime
from str2bool import str2bool
import logging
import yaml

from kubernetes import client as k8s_client
from kubernetes import config

from kubeflow.training import TrainingClient
from kubeflow.training import KubeflowOrgV1RunPolicy
from kubeflow.training import KubeflowOrgV1PyTorchJob
from kubeflow.training import KubeflowOrgV1PyTorchJobSpec


def yamlOrJsonStr(string):
    if string == "" or string is None:
        return None
    return yaml.safe_load(string)


def get_current_namespace():
    """Returns current namespace if available, else kubeflow"""
    try:
        namespace = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
        current_namespace = open(namespace).read()
    except FileNotFoundError:
        current_namespace = "kubeflow"
    return current_namespace



def get_arg_parser():
    parser = argparse.ArgumentParser(description='Kubeflow Job launcher')
    parser.add_argument('--name', type=str,
                        default="pytorchjob",
                        help='Job name.')
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
    parser.add_argument('--masterSpec', type=yamlOrJsonStr,
                        default={},
                        help='Job master replicaSpecs.')
    parser.add_argument('--workerSpec', type=yamlOrJsonStr,
                        default={},
                        help='Job worker replicaSpecs.')
    parser.add_argument('--deleteAfterDone', type=str2bool,
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
    return parser


def main(args):
    logging.getLogger(__name__).setLevel(logging.INFO)
    logging.info('Generating job template.')

    jobSpec = KubeflowOrgV1PyTorchJobSpec(
        pytorch_replica_specs={
            'Master': args.masterSpec,
            'Worker': args.workerSpec,
        },
        run_policy=KubeflowOrgV1RunPolicy(
            active_deadline_seconds=args.activeDeadlineSeconds,
            backoff_limit=args.backoffLimit,
            clean_pod_policy=args.cleanPodPolicy,
            ttl_seconds_after_finished=args.ttlSecondsAfterFinished,
        )
    )

    api_version = f"{args.jobGroup}/{args.version}"

    job = KubeflowOrgV1PyTorchJob(
        api_version=api_version,
        kind=args.kind,
        metadata=k8s_client.V1ObjectMeta(
            name=args.name,
            namespace=args.namespace,
        ),
        spec=jobSpec,
    )
    logging.info('Creating TrainingClient.')

    # remove one of these depending on where you are running this
    config.load_incluster_config()
    #config.load_kube_config()
    
    training_client = TrainingClient()

    logging.info(f"Creating PyTorchJob in namespace: {args.namespace}")
    training_client.create_job(job, namespace=args.namespace)

    expected_conditions = ["Succeeded", "Failed"]
    logging.info(
        f'Monitoring job until status is any of {expected_conditions}.'
    )
    training_client.wait_for_job_conditions(
        name=args.name,
        namespace=args.namespace,
        job_kind=args.kind,
        expected_conditions=set(expected_conditions),
        timeout=int(datetime.timedelta(minutes=args.jobTimeoutMinutes).total_seconds())
    )
    if args.deleteAfterDone:
        logging.info('Deleting job after completion.')
        training_client.delete_job(args.name, args.namespace)


if __name__ == "__main__":
    parser = get_arg_parser()
    args = parser.parse_args()
    main(args)
