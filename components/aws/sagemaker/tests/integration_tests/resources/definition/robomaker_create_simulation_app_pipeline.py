import kfp
from kfp import components
from kfp import dsl

robomaker_create_sim_app_op = components.load_component_from_file(
    "../../create_simulation_app/component.yaml"
)


@dsl.pipeline(
    name="RoboMaker Create Simulation App",
    description="RoboMaker Create Simulation App test pipeline",
)
def robomaker_create_simulation_app_test(
    region="",
    app_name="",
    sources="",
    simulation_software_name="",
    simulation_software_version="",
    robot_software_name="",
    robot_software_version="",
    rendering_engine_name="",
    rendering_engine_version="",
):

    robomaker_create_sim_app_op(
        region=region,
        app_name=app_name,
        sources=sources,
        simulation_software_name=simulation_software_name,
        simulation_software_version=simulation_software_version,
        robot_software_name=robot_software_name,
        robot_software_version=robot_software_version,
        rendering_engine_name=rendering_engine_name,
        rendering_engine_version=rendering_engine_version,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        robomaker_create_simulation_app_test, __file__ + ".yaml"
    )
