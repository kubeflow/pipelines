import unittest

from ..components import create_component_from_tfx_component


class ComponentFromTfxTestCase(unittest.TestCase):

    def test_create_component_from_tfx_with_file_io(self):
        try:
            from tfx.components import Trainer
        except ImportError:
            self.skipTest('tfx package is not installed')

        op = create_component_from_tfx_component(
            tfx_component_class=Trainer,
            use_uri_io=False,
            base_image='tensorflow/tfx:0.21.5',
        )
        component_spec = op.component_spec

        self.assertEquals(component_spec.name, 'Trainer')
        self.assertTrue(component_spec.inputs)
        self.assertTrue(component_spec.outputs)
        self.assertEquals(component_spec.implementation.container.image, 'tensorflow/tfx:0.21.5')
        
    def test_create_component_from_tfx_with_uri_io(self):
        try:
            from tfx.components import Trainer
        except ImportError:
            self.skipTest('tfx package is not installed')

        op = create_component_from_tfx_component(
            tfx_component_class=Trainer,
            use_uri_io=True,
            base_image='tensorflow/tfx:0.21.5',
        )
        component_spec = op.component_spec

        self.assertEquals(component_spec.name, 'Trainer')
        self.assertTrue(component_spec.inputs)
        self.assertTrue(component_spec.outputs)
        self.assertEquals(component_spec.implementation.container.image, 'tensorflow/tfx:0.21.5')

        self.assertIn('beam_pipeline_args', component_spec.inputs)
        self.assertTrue(all(output.name.endswith('_uri') for output in component_spec.outputs))
