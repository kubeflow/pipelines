# Copyright 2023 The Kubeflow Authors
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

import argparse
from datetime import datetime
import random
import string

import constants
import kfp
import kfp.dsl as dsl


@dsl.container_component
def say_hello(name: str):
    return dsl.ContainerSpec(
        image='library/bash:4.4.23', command=['echo'], args=[f'Hello, {name}!'])


@dsl.pipeline(name='My first pipeline', description='A hello pipeline.')
def hello_pipeline(name: str):
    say_hello(name=name)


# Parsing the input arguments
def parse_arguments():
    """Parse command line arguments."""

    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--host', type=str, required=True, help='The host of kfp.')
    args = parser.parse_args()
    return args


def main():
    args = parse_arguments()

    ###### Initialization ######
    client = kfp.Client(args.host)
    print('host is {}'.format(args.host))

    ###### Create Experiment ######
    print('Creating experiment')
    experiment_name = 'kfp-functional-e2e-expriment-' + ''.join(
        random.choices(string.ascii_uppercase + string.digits, k=5))
    response = client.create_experiment(
        experiment_name, namespace=constants.DEFAULT_USER_NAMESPACE)
    experiment_id = response.experiment_id
    print('Experiment with id {} created'.format(experiment_id))
    try:
        ###### Create Run from Pipeline Func ######
        print('Creating Run from Pipeline Func')
        response = client.create_run_from_pipeline_func(
            hello_pipeline,
            arguments={'name': 'World'},
            experiment_name=experiment_name,
            namespace=constants.DEFAULT_USER_NAMESPACE)
        run_id = response.run_id
        print('Run {} created'.format(run_id))

        ###### Monitor Run ######
        start_time = datetime.now()
        response = client.wait_for_run_completion(run_id,
                                                  constants.RUN_TIMEOUT_SECONDS)
        success = (response.state.lower() == 'succeeded')
        end_time = datetime.now()
        elapsed_time = (end_time - start_time).seconds
        if success:
            print('Run succeeded in {} seconds'.format(elapsed_time))
        else:
            print("Run can't complete in {} seconds".format(elapsed_time))
    finally:
        ###### Archive Experiment ######
        print('Archiving experiment')
        client.archive_experiment(experiment_id)
        print('Archived experiment with id {}'.format(experiment_id))


if __name__ == '__main__':
    main()
