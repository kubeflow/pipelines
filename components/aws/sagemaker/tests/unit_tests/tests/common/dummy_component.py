from tests.unit_tests.tests.common.dummy_spec import (
    DummyInputs,
    DummyOutputs,
    DummySpec,
)
from common.sagemaker_component import ComponentMetadata, SageMakerComponent


@ComponentMetadata(
    name="Dummy component", description="Dummy description", spec=DummySpec
)
class DummyComponent(SageMakerComponent):
    pass
