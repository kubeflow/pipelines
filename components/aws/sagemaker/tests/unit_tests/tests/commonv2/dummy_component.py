from tests.unit_tests.tests.commonv2.dummy_spec import (
    DummySpec,
)
from commonv2.sagemaker_component import ComponentMetadata, SageMakerComponent


@ComponentMetadata(
    name="Dummy component", description="Dummy description", spec=DummySpec
)
class DummyComponent(SageMakerComponent):
    pass
