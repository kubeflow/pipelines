"""SageMaker component for training"""
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from abc import ABC, abstractmethod

from .sagemaker_training_spec import SageMakerTrainingSpec
from common.sagemaker_component import SageMakerComponent, ComponentMetadata


@ComponentMetadata(
    name="SageMaker - Training Job",
    description="Train Machine Learning and Deep Learning Models using SageMaker",
    spec=SageMakerTrainingSpec,
)
class SageMakerTrainingComponent(SageMakerComponent):
    """SageMaker component for training."""

    def __init__(self):
        """Initialize a new component."""
        super()

    def Do(self, spec):
        pass


if __name__ == "__main__":
    pass
