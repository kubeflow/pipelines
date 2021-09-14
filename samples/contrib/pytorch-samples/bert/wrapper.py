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
"""Bert Wrapper."""
import torch
import torch.nn as nn
import torch.nn.functional as F


class AGNewsmodelWrapper(nn.Module):
    """Warapper Class."""

    def __init__(self, model):
        super(  # pylint: disable=super-with-arguments
            AGNewsmodelWrapper, self
        ).__init__()
        self.model = model

    def compute_bert_outputs(  # pylint: disable=no-self-use
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
            attention_mask = torch.ones(  # pylint: disable=no-member
                embedding_input.shape[0], embedding_input.shape[1]
            ).to(embedding_input)

        extended_attention_mask = attention_mask.unsqueeze(1).unsqueeze(2)

        extended_attention_mask = extended_attention_mask.to(
            dtype=next(model_bert.parameters()).dtype
        )  # fp16 compatibility
        extended_attention_mask = (1.0 - extended_attention_mask) * -10000.0

        if head_mask is not None:
            if head_mask.dim() == 1:
                head_mask = (
                    head_mask.unsqueeze(0).unsqueeze(0).unsqueeze(-1).
                    unsqueeze(-1)
                )
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

    def forward(self, embeddings, attention_mask=None):
        """Forward function.

        Args:
              embeddings : bert embeddings.
              attention_mask: Attention mask value
        """
        outputs = self.compute_bert_outputs(
            self.model.bert_model, embeddings, attention_mask
        )
        pooled_output = outputs[1]
        output = F.relu(self.model.fc1(pooled_output))
        output = self.model.drop(output)
        output = self.model.out(output)
        return output
