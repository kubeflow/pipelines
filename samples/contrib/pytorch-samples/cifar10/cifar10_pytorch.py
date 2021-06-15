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
"""Cifar10 training script."""
import os
from pathlib import Path
from argparse import ArgumentParser
import pytorch_lightning as pl
from pytorch_lightning.loggers import TensorBoardLogger
from pytorch_lightning.callbacks import (
    EarlyStopping,
    LearningRateMonitor,
    ModelCheckpoint,
)
from pytorch_kfp_components.components.visualization.component import Visualization
from pytorch_kfp_components.components.trainer.component import Trainer
from pytorch_kfp_components.components.mar.component import MarGeneration

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
    default="output/train/models",
    help="Path to save model checkpoints (default: output/train/models)",
)

parser.add_argument(
    "--dataset_path",
    type=str,
    default="output/processing",
    help="Cifar10 Dataset path (default: output/processing)",
)

parser.add_argument(
    "--model_name",
    type=str,
    default="resnet.pth",
    help="Name of the model to be saved as (default: resnet.pth)",
)

parser.add_argument(
    "--mlpipeline_ui_metadata",
    default="mlpipeline-ui-metadata.json",
    type=str,
    help="Path to write mlpipeline-ui-metadata.json",
)

parser.add_argument(
    "--mlpipeline_metrics",
    default="mlpipeline-metrics.json",
    type=str,
    help="Path to write mlpipeline-metrics.json",
)

parser.add_argument(
    "--confusion_matrix_url",
    type=str,
    default=None,
    help="Minio url to generate confusion matrix",
)

parser.add_argument(
    "--pod_template_spec",
    type=str,
    default=None,
    help="Pod template spec",
)

parser = pl.Trainer.add_argparse_args(parent_parser=parser)

args = vars(parser.parse_args())

# Enabling Tensorboard Logger, ModelCheckpoint, Earlystopping

lr_logger = LearningRateMonitor()
tboard = TensorBoardLogger(args["tensorboard_root"])
early_stopping = EarlyStopping(
    monitor="val_loss", mode="min", patience=5, verbose=True
)
checkpoint_callback = ModelCheckpoint(
    dirpath=args["checkpoint_dir"],
    filename="cifar10_{epoch:02d}",
    save_top_k=1,
    verbose=True,
    monitor="val_loss",
    mode="min",
)

if not args["max_epochs"]:
    args["max_epochs"] = 1

if args["accelerator"] and args["accelerator"] == "None":
    args["accelerator"] = None

# Setting the trainer specific arguments
trainer_args = {
    "logger": tboard,
    "checkpoint_callback": True,
    "callbacks": [lr_logger, early_stopping, checkpoint_callback],
}

if "profiler" in args and args["profiler"] != "":
    trainer_args["profiler"] = args["profiler"]

# Setting the datamodule specific arguments
data_module_args = {"train_glob": args["dataset_path"]}

# Creating parent directories
Path(args["tensorboard_root"]).mkdir(parents=True, exist_ok=True)
Path(args["checkpoint_dir"]).mkdir(parents=True, exist_ok=True)

# Initiating the training process
trainer = Trainer(
    module_file="cifar10_train.py",
    data_module_file="cifar10_datamodule.py",
    module_file_args=args,
    data_module_args=data_module_args,
    trainer_args=trainer_args,
)

model = trainer.ptl_trainer.get_model()

if trainer.ptl_trainer.global_rank == 0:
    # Mar file generation

    cifar_dir, _ = os.path.split(os.path.abspath(__file__))

    mar_config = {
        "MODEL_NAME":
            "cifar10_test",
        "MODEL_FILE":
            os.path.join(cifar_dir, "cifar10_train.py"),
        "HANDLER":
            "image_classifier",
        "SERIALIZED_FILE":
            os.path.join(args["checkpoint_dir"], args["model_name"]),
        "VERSION":
            "1",
        "EXPORT_PATH":
            args["checkpoint_dir"],
        "CONFIG_PROPERTIES":
            os.path.join(cifar_dir, "config.properties"),
        "REQUIREMENTS_FILE":
            os.path.join(cifar_dir, "requirements.txt")
    }

    MarGeneration(mar_config=mar_config, mar_save_path=args["checkpoint_dir"])

    classes = [
        "airplane",
        "automobile",
        "bird",
        "cat",
        "deer",
        "dog",
        "frog",
        "horse",
        "ship",
        "truck",
    ]

    # print(dir(trainer.ptl_trainer.model.module))
    # model = trainer.ptl_trainer.model

    target_index_list = list(set(model.target))

    class_list = []
    for index in target_index_list:
        class_list.append(classes[index])

    confusion_matrix_dict = {
        "actuals": model.target,
        "preds": model.preds,
        "classes": class_list,
        "url": args["confusion_matrix_url"],
    }

    test_accuracy = round(float(model.test_acc.compute()), 2)

    print("Model test accuracy: ", test_accuracy)

    visualization_arguments = {
        "input": {
            "tensorboard_root": args["tensorboard_root"],
            "checkpoint_dir": args["checkpoint_dir"],
            "dataset_path": args["dataset_path"],
            "model_name": args["model_name"],
            "confusion_matrix_url": args["confusion_matrix_url"],
        },
        "output": {
            "mlpipeline_ui_metadata": args["mlpipeline_ui_metadata"],
            "mlpipeline_metrics": args["mlpipeline_metrics"],
        },
    }

    markdown_dict = {"storage": "inline", "source": visualization_arguments}

    print("Visualization Arguments: ", markdown_dict)

    visualization = Visualization(
        test_accuracy=test_accuracy,
        confusion_matrix_dict=confusion_matrix_dict,
        mlpipeline_ui_metadata=args["mlpipeline_ui_metadata"],
        mlpipeline_metrics=args["mlpipeline_metrics"],
        markdown=markdown_dict,
    )

    checpoint_dir_contents = os.listdir(args["checkpoint_dir"])
    print(f"Checkpoint Directory Contents: {checpoint_dir_contents}")
