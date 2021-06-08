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

from argparse import ArgumentParser

import pytorch_lightning as pl

from pytorch_pipeline.components.trainer.component import Trainer

# Argument parser for user defined paths
parser = ArgumentParser()

parser.add_argument(
    "--tensorboard_root",
    type=str,
    default="output/tensorboard",
    help="Tensorboard Root path (default: output/tensorboard)",
)

parser.add_argument(
    "--checkpoint_dir",
    type=str,
    default="output",
    help="Path to save model checkpoints (default: output/train/models)",
)

parser.add_argument(
    "--model_name",
    type=str,
    default="iris.pt",
    help="Name of the model to be saved as (default: iris.pt)",
)

parser = pl.Trainer.add_argparse_args(parent_parser=parser)

args = vars(parser.parse_args())

if not args["max_epochs"]:
    max_epochs = 5
else:
    max_epochs = args["max_epochs"]

args["max_epochs"] = max_epochs

trainer_args = {}

# Initiating the training process
trainer = Trainer(
    module_file="iris_classification.py",
    data_module_file="iris_data_module.py",
    module_file_args=args,
    data_module_args=None,
    trainer_args=trainer_args,
)
