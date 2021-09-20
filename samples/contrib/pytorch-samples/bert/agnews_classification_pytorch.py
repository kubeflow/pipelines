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
"""AG news Classification script."""
import os
from argparse import ArgumentParser
from pathlib import Path
from pytorch_lightning.loggers import TensorBoardLogger
from pytorch_lightning.callbacks import (
    EarlyStopping,
    LearningRateMonitor,
    ModelCheckpoint,
)
from pytorch_kfp_components.components.visualization.component import Visualization
from pytorch_kfp_components.components.trainer.component import Trainer
from pytorch_kfp_components.components.mar.component import MarGeneration
from pytorch_kfp_components.components.utils.argument_parsing import parse_input_args
# Argument parser for user defined paths
import pytorch_lightning
print("Using Pytorch Lighting: {}".format(pytorch_lightning.__version__)) #pylint: disable=no-member
parser = ArgumentParser()

parser.add_argument(
    "--dataset_path",
    type=str,
    default="output/processing",
    help="Path to input dataset",
)

parser.add_argument(
    "--mlpipeline_ui_metadata",
    type=str,
    default="mlpipeline-ui-metadata.json",
    help="Path to write mlpipeline-ui-metadata.json",
)

parser.add_argument(
    "--mlpipeline_metrics",
    type=str,
    default="mlpipeline-metrics",
    help="Path to write mlpipeline-metrics.json",
)

parser.add_argument(
    "--script_args",
    type=str,
    help="Arguments for bert agnews classification script",
)

parser.add_argument(
    "--ptl_args",
    type=str,
    help="Arguments specific to PTL trainer",
)

parser.add_argument(
    "--checkpoint_dir",
    default="output/train/models",
    type=str,
    help="Arguments specific to PTL trainer",
)

parser.add_argument(
    "--tensorboard_root",
    default="output/tensorboard",
    type=str,
    help="Arguments specific to PTL trainer",
)
args = vars(parser.parse_args())
script_args = args["script_args"]
ptl_args = args["ptl_args"]

TENSOBOARD_ROOT = args["tensorboard_root"]
CHECKPOINT_DIR = args["checkpoint_dir"]
DATASET_PATH = args["dataset_path"]

script_dict: dict = parse_input_args(input_str=script_args)
script_dict["checkpoint_dir"] = CHECKPOINT_DIR

ptl_dict: dict = parse_input_args(input_str=ptl_args)

# Enabling Tensorboard Logger, ModelCheckpoint, Earlystopping

lr_logger = LearningRateMonitor()
tboard = TensorBoardLogger(TENSOBOARD_ROOT)
early_stopping = EarlyStopping(
    monitor="val_loss", mode="min", patience=5, verbose=True
)
checkpoint_callback = ModelCheckpoint(
    dirpath=CHECKPOINT_DIR,
    filename="bert_{epoch:02d}",
    save_top_k=1,
    verbose=True,
    monitor="val_loss",
    mode="min",
)

if "accelerator" in ptl_dict and ptl_dict["accelerator"] == "None":
    ptl_dict["accelerator"] = None

# Setting the trainer specific arguments
trainer_args = {
    "logger": tboard,
    "checkpoint_callback": True,
    "callbacks": [lr_logger, early_stopping, checkpoint_callback],
}

if not ptl_dict["max_epochs"]:
    trainer_args["max_epochs"] = 1
else:
    trainer_args["max_epochs"] = ptl_dict["max_epochs"]

if "profiler" in ptl_dict and ptl_dict["profiler"] != "":
    trainer_args["profiler"] = ptl_dict["profiler"]

# Setting the datamodule specific arguments
data_module_args = {
    "train_glob": DATASET_PATH,
    "num_samples": script_dict["num_samples"]
}

# Creating parent directories
Path(TENSOBOARD_ROOT).mkdir(parents=True, exist_ok=True)
Path(CHECKPOINT_DIR).mkdir(parents=True, exist_ok=True)

# Updating all the input parameter to PTL dict

trainer_args.update(ptl_dict)

# Initiating the training process
trainer = Trainer(
    module_file="bert_train.py",
    data_module_file="bert_datamodule.py",
    module_file_args=script_dict,
    data_module_args=data_module_args,
    trainer_args=trainer_args,
)

print("Generated tensorboard files")
for root, dirs, files in os.walk(args["tensorboard_root"]):  # pylint: disable=unused-variable
    for file in files:
        print(file)

model = trainer.ptl_trainer.lightning_module

if trainer.ptl_trainer.global_rank == 0:
    # Mar file generation

    bert_dir, _ = os.path.split(os.path.abspath(__file__))

    mar_config = {
        "MODEL_NAME":
            "bert_test",
        "MODEL_FILE":
            os.path.join(bert_dir, "bert_train.py"),
        "HANDLER":
            os.path.join(bert_dir, "bert_handler.py"),
        "SERIALIZED_FILE":
            os.path.join(CHECKPOINT_DIR, script_dict["model_name"]),
        "VERSION":
            "1",
        "EXPORT_PATH":
            CHECKPOINT_DIR,
        "CONFIG_PROPERTIES":
            os.path.join(bert_dir, "config.properties"),
        "EXTRA_FILES":
            "{},{},{}".format(
                os.path.join(bert_dir, "bert-base-uncased-vocab.txt"),
                os.path.join(bert_dir, "index_to_name.json"),
                os.path.join(bert_dir, "wrapper.py")
            ),
        "REQUIREMENTS_FILE":
            os.path.join(bert_dir, "requirements.txt")
    }

    MarGeneration(mar_config=mar_config, mar_save_path=CHECKPOINT_DIR)

    classes = [
        "World",
        "Sports",
        "Business",
        "Sci/Tech",
    ]

    # model = trainer.ptl_trainer.model

    target_index_list = list(set(model.target))

    class_list = []
    for index in target_index_list:
        class_list.append(classes[index])

    confusion_matrix_dict = {
        "actuals": model.target,
        "preds": model.preds,
        "classes": class_list,
        "url": script_dict["confusion_matrix_url"],
    }

    test_accuracy = round(float(model.test_acc.compute()), 2)

    print("Model test accuracy: ", test_accuracy)

    visualization_arguments = {
        "input": {
            "tensorboard_root": TENSOBOARD_ROOT,
            "checkpoint_dir": CHECKPOINT_DIR,
            "dataset_path": DATASET_PATH,
            "model_name": script_dict["model_name"],
            "confusion_matrix_url": script_dict["confusion_matrix_url"],
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
