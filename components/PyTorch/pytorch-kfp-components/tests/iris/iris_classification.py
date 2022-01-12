#!/usr/bin/env/python3
#
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

# pylint: disable=W0221
# pylint: disable=W0613
# pylint: disable=W0223
from argparse import ArgumentParser

import pytorch_lightning as pl
import torch
import torch.nn as nn
import torch.nn.functional as F
from torchmetrics import Accuracy


class IrisClassification(pl.LightningModule):

    def __init__(self, **kwargs):
        super(IrisClassification, self).__init__()

        self.train_acc = Accuracy()
        self.val_acc = Accuracy()
        self.test_acc = Accuracy()
        self.args = kwargs

        self.fc1 = nn.Linear(4, 10)
        self.fc2 = nn.Linear(10, 10)
        self.fc3 = nn.Linear(10, 3)
        self.cross_entropy_loss = nn.CrossEntropyLoss()

    def forward(self, x):
        x = F.relu(self.fc1(x))
        x = F.relu(self.fc2(x))
        x = F.relu(self.fc3(x))
        return x

    @staticmethod
    def add_model_specific_args(parent_parser):
        """
        Add model specific arguments like learning rate

        :param parent_parser: Application specific parser

        :return: Returns the augmented arugument parser
        """
        parser = ArgumentParser(parents=[parent_parser], add_help=False)
        parser.add_argument(
            "--lr",
            type=float,
            default=0.01,
            metavar="LR",
            help="learning rate (default: 0.001)",
        )
        return parser

    def configure_optimizers(self):
        return torch.optim.Adam(self.parameters(), self.args.get("lr", 0.01))

    def training_step(self, batch, batch_idx):
        x, y = batch
        logits = self.forward(x)
        _, y_hat = torch.max(logits, dim=1)
        loss = self.cross_entropy_loss(logits, y)
        self.train_acc(y_hat, y)
        self.log(
            "train_acc",
            self.train_acc.compute(),
            on_step=False,
            on_epoch=True,
        )
        self.log("train_loss", loss)
        return {"loss": loss}

    def validation_step(self, batch, batch_idx):
        x, y = batch
        logits = self.forward(x)
        _, y_hat = torch.max(logits, dim=1)
        loss = F.cross_entropy(logits, y)
        self.val_acc(y_hat, y)
        self.log("val_acc", self.val_acc.compute())
        self.log("val_loss", loss, sync_dist=True)

    def test_step(self, batch, batch_idx):
        x, y = batch
        logits = self.forward(x)
        _, y_hat = torch.max(logits, dim=1)
        self.test_acc(y_hat, y)
        self.log("test_acc", self.test_acc.compute())
