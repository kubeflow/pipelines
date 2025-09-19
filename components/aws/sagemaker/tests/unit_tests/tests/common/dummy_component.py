from common.sagemaker_component import ComponentMetadata
from common.sagemaker_component import SageMakerComponent
from tests.unit_tests.tests.common.dummy_spec import DummyInputs
from tests.unit_tests.tests.common.dummy_spec import DummyOutputs
from tests.unit_tests.tests.common.dummy_spec import DummySpec


@ComponentMetadata(
    name="Dummy component", description="Dummy description", spec=DummySpec
)
class DummyComponent(SageMakerComponent):
    pass
