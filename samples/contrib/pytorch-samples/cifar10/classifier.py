# !/usr/bin/env/python3
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
""" This is the model wrapper """

from torch import nn
from torchvision import models
import pytorch_lightning as pl


class CIFAR10CLASSIFIER(pl.LightningModule):  # pylint: disable=too-many-ancestors
    """
    model wrapper for cifar10 classification
    """

    def __init__(self, **kwargs):
        """
        Initializes the network, optimizer and scheduler
        """
        super().__init__()
        self.model_conv = models.resnet50(pretrained=True)
        for param in self.model_conv.parameters():
            param.requires_grad = False
        num_ftrs = self.model_conv.fc.in_features
        num_classes = 10
        self.model_conv.fc = nn.Linear(num_ftrs, num_classes)

    def forward(self, x):  # pylint: disable=arguments-differ
        out = self.model_conv(x)
        return out
