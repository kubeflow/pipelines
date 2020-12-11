# Copyright 2020 Google LLC
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
import random
import string
from datetime import datetime

import kfp
from kfp import dsl
import constants


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
    parser.add_argument('--host',
                        type=str,
                        required=True,
                        help='The host of kfp.')
    args = parser.parse_args()
    return args


def main():
    args = parse_arguments()

    ###### Initialization ######
    client = kfp.Client(args.host)
    print("host is {}".format(args.host))

    ###### Create Experiment ######
    print("Creating experiment")
    experiment_name = "kfp-functional-e2e-expriment-" + "".join(random.choices(string.ascii_uppercase +
                                                                               string.digits, k=5))
    response = client.create_experiment(experiment_name)
    experiment_id = response.id
    print("Experiment with id {} created".format(experiment_id))
    try:
        ###### Create Run from Pipeline Func ######
        print("Creating Run from Pipeline Func")
        response = client.create_run_from_pipeline_func(hello_world_pipeline, arguments={}, experiment_name=experiment_name)
        run_id = response.run_id
        print("Run {} created".format(run_id))

        ###### Monitor Run ######
        start_time = datetime.now()
        response = client.wait_for_run_completion(run_id, constants.RUN_TIMEOUT_SECONDS)
        succ = (response.run.status.lower() == 'succeeded')
        end_time = datetime.now()
        elapsed_time = (end_time - start_time).seconds
        if succ:
            print("Run succeeded in {} seconds".format(elapsed_time))
        else:
            print("Run can't complete in {} seconds".format(elapsed_time))
    finally:
        ###### Archive Experiment ######
        print("Archiving experiment")
        client.experiments.archive_experiment(experiment_id)
        print("Archived experiment with id {}".format(experiment_id))


if __name__ == "__main__":
    main()
