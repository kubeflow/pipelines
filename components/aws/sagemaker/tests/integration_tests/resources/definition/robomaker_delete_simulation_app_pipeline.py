import kfp
from kfp import components
from kfp import dsl

robomaker_delete_sim_app_op = components.load_component_from_file(
    "../../delete_simulation_app/component.yaml"
)


@dsl.pipeline(
    name="RoboMaker Delete Simulation App",
    description="RoboMaker Delete Simulation App test pipeline",
)
def robomaker_delete_simulation_app_test(
    region="", arn="",
):

    robomaker_delete_sim_app_op(
        region=region, arn=arn,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        robomaker_delete_simulation_app_test, __file__ + ".yaml"
    )
