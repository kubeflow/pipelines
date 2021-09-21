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
"""News dataset script."""
import torch
from torch.utils.data import Dataset


class NewsDataset(Dataset):
    """Ag News Dataset
    Args:
        Dataset
    """

    def __init__(self, reviews, targets, tokenizer, max_length):
        """Performs initialization of tokenizer.

        Args:
             reviews: AG news text
             targets: labels
             tokenizer: bert tokenizer
             max_length: maximum length of the news text
        """
        self.reviews = reviews
        self.targets = targets
        self.tokenizer = tokenizer
        self.max_length = max_length

    def __len__(self):
        """
        Returns:
             returns the number of datapoints in the dataframe

        """
        return len(self.reviews)

    def __getitem__(self, item):
        """Returns the review text and the targets of the specified item.

        Args:
             item: Index of sample review

        Returns:
             Returns the dictionary of review text,
             input ids, attention mask, targets
        """
        review = str(self.reviews[item])
        target = self.targets[item]

        encoding = self.tokenizer.encode_plus(
            review,
            add_special_tokens=True,
            max_length=self.max_length,
            return_token_type_ids=False,
            padding="max_length",
            return_attention_mask=True,
            return_tensors="pt",
            truncation=True,
        )

        return {
            "review_text": review,
            "input_ids": encoding["input_ids"].flatten(),
            "attention_mask": encoding["attention_mask"].flatten(),  # pylint: disable=not-callable
            "targets": torch.tensor(target, dtype=torch.long),  # pylint: disable=no-member,not-callable
        }
