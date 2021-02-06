import kfp
from kfp import components
from kfp import dsl

robomaker_sim_job_op = components.load_component_from_file(
    "../../simulation_job/component.yaml"
)


@dsl.pipeline(
    name="Run RoboMaker Simulation Job",
    description="RoboMaker Simulation Job test pipeline",
)
def robomaker_simulation_job_test(
    region="",
    role="",
    output_bucket="",
    output_path="",
    max_run="",
    failure_behavior="",
    sim_app_arn="",
    sim_app_launch_config="",
    robot_app_arn="",
    robot_app_launch_config="",
):

    robomaker_sim_job_op(
        region=region,
        role=role,
        output_bucket=output_bucket,
        output_path=output_path,
        max_run=max_run,
        failure_behavior=failure_behavior,
        sim_app_arn=sim_app_arn,
        sim_app_launch_config=sim_app_launch_config,
        robot_app_arn=robot_app_arn,
        robot_app_launch_config=robot_app_launch_config,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(robomaker_simulation_job_test, __file__ + ".yaml")
