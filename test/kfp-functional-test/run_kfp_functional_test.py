import argparse
import os
import random
import string
from datetime import datetime

import kfp
import utils
from kfp import dsl
from kfp.containers._gcs_helper import GCSHelper


def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['echo "hello world"']
    )


@dsl.pipeline(
    name='My first pipeline',
    description='A hello world pipeline.'
)
def hello_world_pipeline():
    echo_task = echo_op()


# Parsing the input arguments
def parse_arguments():
    """Parse command line arguments."""

    parser = argparse.ArgumentParser()
    parser.add_argument('--result_dir',
                        type=str,
                        required=True,
                        help='The path of the test result that will be exported.')
    parser.add_argument('--host',
                        type=str,
                        required=True,
                        help='The host of kfp.')
    parser.add_argument('--gcs_dir',
                        type=str,
                        required=True,
                        help='The gcs dir to store the test result.')
    args = parser.parse_args()
    return args


def main():
    args = parse_arguments()
    test_cases = []
    test_name = 'KFP Functional Test'

    ###### Initialization ######
    client = kfp.Client(args.host)
    print('host is %s' % args.host)
    print('test results will be stored in %s' % args.result_dir)

    ###### Create Experiment ######
    experiment_name = 'kfp-functional-e2e-expriment-' + ''.join(random.choices(string.ascii_uppercase +
                                                                               string.digits, k=5))
    response = client.create_experiment(experiment_name)
    experiment_id = response.id
    utils.add_junit_test(test_cases, 'create experiment', True)

    ###### Create Run from Pipeline Func ######
    response = client.create_run_from_pipeline_func(hello_world_pipeline, arguments={}, experiment_name=experiment_name)
    utils.add_junit_test(test_cases, 'create run from pipeline func', True)
    run_id = response.run_id

    ###### Monitor Run ######
    start_time = datetime.now()
    response = client.wait_for_run_completion(run_id, 1800)
    succ = (response.run.status.lower() == 'succeeded')
    end_time = datetime.now()
    elapsed_time = (end_time - start_time).seconds
    utils.add_junit_test(test_cases, 'run completion', succ, 'waiting for run completion failure', elapsed_time)

    ###### Archive Experiment ######
    client.experiments.archive_experiment(experiment_id)
    utils.add_junit_test(test_cases, 'archive experiment', True)

    result = "junit_kfp_e2e%sOutput.xml" % experiment_name
    result_absolute_path = args.result_dir + "/" + result
    utils.write_junit_xml(test_name, result_absolute_path, test_cases)
    print('Copy the test results to GCS %s' % args.gcs_dir)
    GCSHelper.upload_gcs_file(result_absolute_path, os.path.join(args.gcs_dir, result))


if __name__ == "__main__":
    main()
