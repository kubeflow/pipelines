import kfp
import kfp.dsl as dsl
from kfp import components

from conftest import runner


@components.create_component_from_func
def write_to_volume():
    with open("/mnt/file.txt", "w") as file:
        file.write("Hello world")


@dsl.pipeline(
    name="volumeop-basic",
    description="A Basic Example on VolumeOp Usage."
)
def volumeop_basic(size: str="1Gi"):
    vop = dsl.VolumeOp(
        name="create-pvc",
        resource_name="my-pvc",
        modes=dsl.VOLUME_MODE_RWO,
        size=size
    )
    
    write_to_volume().add_pvolumes({"/mnt'": vop.volume})

def test_volumeop_basic():
    runner(volumeop_basic)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(volumeop_basic, __file__ + '.yaml')
