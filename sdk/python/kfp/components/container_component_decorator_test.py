import unittest

from kfp.components.container_component import ContainerComponent
from kfp.components.structures import ContainerSpec
from kfp.components.container_component_decorator import container_component


class TestContainerComponentDecorator(unittest.TestCase):

    def test_func_with_no_arg(self):

        @container_component
        def hello_world() -> str:
            """Hello world component."""
            return ContainerSpec(
                image='python3.7',
                command=['echo', 'hello world'],
                args=[],
            )

        self.assertIsInstance(hello_world, ContainerComponent)
        self.assertIsNone(hello_world.component_spec.inputs)
