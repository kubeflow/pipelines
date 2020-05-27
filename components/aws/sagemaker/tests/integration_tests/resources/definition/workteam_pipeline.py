#!/usr/bin/env python3

# Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You
# may not use this file except in compliance with the License. A copy of
# the License is located at
#
# http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is
# distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF
# ANY KIND, either express or implied. See the License for the specific
# language governing permissions and limitations under the License.

import kfp
import json
import copy
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_workteam_op = components.load_component_from_file(
    "../../workteam/component.yaml"
)


@dsl.pipeline(
    name="SageMaker WorkTeam test pipeline",
    description="SageMaker WorkTeam test pipeline",
)
def workteam_test(
    region="", team_name="", description="", user_pool="", user_groups="", client_id=""
):

    workteam = sagemaker_workteam_op(
        region=region,
        team_name=team_name,
        description=description,
        user_pool=user_pool,
        user_groups=user_groups,
        client_id=client_id,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        workteam_test, "SageMaker_WorkTeam_Pipelines" + ".yaml"
    )
