from kfp import dsl
from kfp.kubernetes import mount_pvc
import pytest


class TestMountPVCFunctionError:

    def test(self):

        @dsl.component
        def identity(string: str) -> str:
            return string

        with pytest.raises(
                ValueError,
                match=r'Argument for \'pvc_name\' must be an instance of str or PipelineChannel\. Got unknown input type: <class \'int\'>\.',
        ):

            @dsl.pipeline
            def my_pipeline(string: str = 'string'):
                op1 = mount_pvc(
                    identity(string=string),
                    pvc_name=1,
                    mount_path='/path',
                )
