#!/usr/bin/env python3

import kfp
from kfp import components
from kfp import dsl

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

    sagemaker_workteam_op(
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
