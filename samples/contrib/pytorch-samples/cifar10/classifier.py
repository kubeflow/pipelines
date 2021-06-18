import os
import numpy as np
import torch
import torch.nn.functional as F
from torch import nn
from torchvision import models
import pytorch_lightning as pl

class CIFAR10CLASSIFIER(pl.LightningModule):
    def __init__(self, **kwargs):
        """
        Initializes the network, optimizer and scheduler
        """
        super(CIFAR10CLASSIFIER, self).__init__()
        self.model_conv = models.resnet50(pretrained=True)
        for param in self.model_conv.parameters():
            param.requires_grad = False
        num_ftrs = self.model_conv.fc.in_features
        num_classes = 10
        self.model_conv.fc = nn.Linear(num_ftrs, num_classes)

    def forward(self, x):
        out = self.model_conv(x)
        return out