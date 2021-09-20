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
# pylint: disable=arguments-differ
# pylint: disable=unused-argument
# pylint: disable=abstract-method
"""Bert Training Script."""
import pytorch_lightning as pl
import torch
import torch.nn.functional as F
from pytorch_lightning.metrics import Accuracy
from sklearn.metrics import accuracy_score
from torch import nn
from transformers import AdamW, BertModel


class BertNewsClassifier(pl.LightningModule):  #pylint: disable=too-many-ancestors,too-many-instance-attributes
    """Bert Model Class."""

    def __init__(self, **kwargs):
        """Initializes the network, optimizer and scheduler."""
        super(BertNewsClassifier, self).__init__()  #pylint: disable=super-with-arguments
        self.pre_trained_model_name = "bert-base-uncased"  #pylint: disable=invalid-name
        self.bert_model = BertModel.from_pretrained(self.pre_trained_model_name)
        for param in self.bert_model.parameters():
            param.requires_grad = False
        self.drop = nn.Dropout(p=0.2)
        # assigning labels
        self.class_names = ["World", "Sports", "Business", "Sci/Tech"]
        n_classes = len(self.class_names)

        self.fc1 = nn.Linear(self.bert_model.config.hidden_size, 512)
        self.out = nn.Linear(512, n_classes)
        # self.bert_model.embedding = self.bert_model.embeddings
        # self.embedding = self.bert_model.embeddings

        self.scheduler = None
        self.optimizer = None
        self.args = kwargs

        self.train_acc = Accuracy()
        self.val_acc = Accuracy()
        self.test_acc = Accuracy()

        self.preds = []
        self.target = []

    def compute_bert_outputs(  #pylint: disable=no-self-use
        self, model_bert, embedding_input, attention_mask=None, head_mask=None
    ):
        """Computes Bert Outputs.

        Args:
            model_bert : the bert model
            embedding_input : input for bert embeddings.
            attention_mask : attention  mask
            head_mask : head mask
        Returns:
            output : the bert output
        """
        if attention_mask is None:
            attention_mask = torch.ones(  #pylint: disable=no-member
                embedding_input.shape[0], embedding_input.shape[1]
            ).to(embedding_input)

        extended_attention_mask = attention_mask.unsqueeze(1).unsqueeze(2)

        extended_attention_mask = extended_attention_mask.to(
            dtype=next(model_bert.parameters()).dtype
        )  # fp16 compatibility
        extended_attention_mask = (1.0 - extended_attention_mask) * -10000.0

        if head_mask is not None:
            if head_mask.dim() == 1:
                head_mask = head_mask.unsqueeze(0).unsqueeze(0).unsqueeze(
                    -1
                ).unsqueeze(-1)
                head_mask = head_mask.expand(
                    model_bert.config.num_hidden_layers, -1, -1, -1, -1
                )
            elif head_mask.dim() == 2:
                head_mask = (
                    head_mask.unsqueeze(1).unsqueeze(-1).unsqueeze(-1)
                )  # We can specify head_mask for each layer
            head_mask = head_mask.to(
                dtype=next(model_bert.parameters()).dtype
            )  # switch to fload if need + fp16 compatibility
        else:
            head_mask = [None] * model_bert.config.num_hidden_layers

        encoder_outputs = model_bert.encoder(
            embedding_input, extended_attention_mask, head_mask=head_mask
        )
        sequence_output = encoder_outputs[0]
        pooled_output = model_bert.pooler(sequence_output)
        outputs = (
            sequence_output,
            pooled_output,
        ) + encoder_outputs[1:]
        return outputs

    def forward(self, input_ids, attention_mask=None):
        """ Forward function.
        Args:
            input_ids: Input data
            attention_maks: Attention mask value

        Returns:
             output - Type of news for the given news snippet
        """
        embedding_input = self.bert_model.embeddings(input_ids)
        outputs = self.compute_bert_outputs(
            self.bert_model, embedding_input, attention_mask
        )
        pooled_output = outputs[1]
        output = torch.tanh(self.fc1(pooled_output))
        output = self.drop(output)
        output = self.out(output)
        return output

    def training_step(self, train_batch, batch_idx):
        """Training the data as batches and returns training loss on each
        batch.

        Args:
            train_batch Batch data
            batch_idx: Batch indices

        Returns:
            output - Training loss
        """
        input_ids = train_batch["input_ids"].to(self.device)
        attention_mask = train_batch["attention_mask"].to(self.device)
        targets = train_batch["targets"].to(self.device)
        output = self.forward(input_ids, attention_mask)
        _, y_hat = torch.max(output, dim=1)  #pylint: disable=no-member
        loss = F.cross_entropy(output, targets)
        self.train_acc(y_hat, targets)
        self.log("train_acc", self.train_acc.compute())
        self.log("train_loss", loss)
        return {"loss": loss, "acc": self.train_acc.compute()}

    def test_step(self, test_batch, batch_idx):
        """Performs test and computes the accuracy of the model.

        Args:
             test_batch: Batch data
             batch_idx: Batch indices

        Returns:
             output - Testing accuracy
        """
        input_ids = test_batch["input_ids"].to(self.device)
        attention_mask = test_batch["attention_mask"].to(self.device)
        targets = test_batch["targets"].to(self.device)
        output = self.forward(input_ids, attention_mask)
        _, y_hat = torch.max(output, dim=1)  #pylint: disable=no-member
        test_acc = accuracy_score(y_hat.cpu(), targets.cpu())
        self.test_acc(y_hat, targets)
        self.preds += y_hat.tolist()
        self.target += targets.tolist()
        self.log("test_acc", self.test_acc.compute())
        return {"test_acc": torch.tensor(test_acc)}  #pylint: disable=no-member

    def validation_step(self, val_batch, batch_idx):
        """Performs validation of data in batches.

        Args:
             val_batch: Batch data
             batch_idx: Batch indices

        Returns:
             output - valid step loss
        """

        input_ids = val_batch["input_ids"].to(self.device)
        attention_mask = val_batch["attention_mask"].to(self.device)
        targets = val_batch["targets"].to(self.device)
        output = self.forward(input_ids, attention_mask)
        _, y_hat = torch.max(output, dim=1)  #pylint: disable=no-member
        loss = F.cross_entropy(output, targets)
        self.val_acc(y_hat, targets)
        self.log("val_acc", self.val_acc.compute())
        self.log("val_loss", loss, sync_dist=True)
        return {"val_step_loss": loss, "acc": self.val_acc.compute()}

    def configure_optimizers(self):
        """Initializes the optimizer and learning rate scheduler.

        Returns:
             output - Initialized optimizer and scheduler
        """
        self.optimizer = AdamW(self.parameters(), lr=self.args.get("lr", 0.001))
        self.scheduler = {
            "scheduler":
                torch.optim.lr_scheduler.ReduceLROnPlateau(
                    self.optimizer,
                    mode="min",
                    factor=0.2,
                    patience=2,
                    min_lr=1e-6,
                    verbose=True,
                ),
            "monitor":
                "val_loss",
        }
        return [self.optimizer], [self.scheduler]
