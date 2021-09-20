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
"""BERT Data Module Script."""

import numpy as np
import pyarrow.parquet as pq
import pytorch_lightning as pl
import torch
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader
from transformers import BertTokenizer
from news_dataset import NewsDataset


class BertDataModule(pl.LightningDataModule):  # pylint: disable=too-many-instance-attributes
    """Data Module Class."""

    def __init__(self, **kwargs):
        """Initialization of inherited lightning data module."""
        super(BertDataModule, self).__init__()  # pylint: disable=super-with-arguments
        self.pre_trained_model_name = "bert-base-uncased"
        self.df_train = None
        self.df_val = None
        self.df_test = None
        self.train_data_loader = None
        self.val_data_loader = None
        self.test_data_loader = None
        self.max_length = 100
        self.encoding = None
        self.tokenizer = None
        self.args = kwargs

    def prepare_data(self):
        """Implementation of abstract class."""

    @staticmethod
    def process_label(rating):
        """Puts labels to ratings"""
        rating = int(rating)
        return rating - 1

    def setup(self, stage=None):
        """Downloads the data, parse it and split the data into train, test,
        validation data.

        Args:
            stage: Stage - training or testing
        """

        num_samples = self.args.get("num_samples", 1000)

        data_path = self.args["train_glob"]

        print("\n\nTRAIN GLOB")
        print(data_path)
        print("\n\n")

        df_parquet = pq.ParquetDataset(self.args["train_glob"])

        dataframe = df_parquet.read_pandas().to_pandas()

        dataframe.columns = ["label", "title", "description"]
        dataframe.sample(frac=1)
        dataframe = dataframe.iloc[:num_samples]

        dataframe["label"] = dataframe.label.apply(self.process_label)

        self.tokenizer = BertTokenizer.from_pretrained(
            self.pre_trained_model_name
        )

        random_seed = 42
        np.random.seed(random_seed)
        torch.manual_seed(random_seed)

        self.df_train, self.df_test = train_test_split(
            dataframe,
            test_size=0.2,
            random_state=random_seed,
            stratify=dataframe["label"],
        )
        self.df_val, self.df_test = train_test_split(
            self.df_test,
            test_size=0.2,
            random_state=random_seed,
            stratify=self.df_test["label"],
        )

    def create_data_loader(self, dataframe, tokenizer, max_len, batch_size):  # pylint: disable=unused-argument
        """Generic data loader function.

        Args:
         dataframe: Input dataframe
         tokenizer: bert tokenizer
         max_len: Max length of the news datapoint
         batch_size: Batch size for training

        Returns:
             Returns the constructed dataloader
        """
        dataset = NewsDataset(
            reviews=dataframe.description.to_numpy(),
            targets=dataframe.label.to_numpy(),
            tokenizer=tokenizer,
            max_length=max_len,
        )

        return DataLoader(
            dataset,
            batch_size=self.args.get("batch_size", 4),
            num_workers=self.args.get("num_workers", 1),
        )

    def train_dataloader(self):
        """Train data loader
        Returns:
             output - Train data loader for the given input
        """
        self.train_data_loader = self.create_data_loader(
            self.df_train,
            self.tokenizer,
            self.max_length,
            self.args.get("batch_size", 4),
        )
        return self.train_data_loader

    def val_dataloader(self):
        """Validation data loader.
        Returns:
            output - Validation data loader for the given input
        """
        self.val_data_loader = self.create_data_loader(
            self.df_val,
            self.tokenizer,
            self.max_length,
            self.args.get("batch_size", 4),
        )
        return self.val_data_loader

    def test_dataloader(self):
        """Test data loader.
        Return:
             output - Test data loader for the given input
        """
        self.test_data_loader = self.create_data_loader(
            self.df_test,
            self.tokenizer,
            self.max_length,
            self.args.get("batch_size", 4),
        )
        return self.test_data_loader
