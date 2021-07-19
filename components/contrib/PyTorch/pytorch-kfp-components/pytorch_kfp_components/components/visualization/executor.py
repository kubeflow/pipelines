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

"""Visualization Executor Class."""
# pylint: disable=C0103
# pylint: disable=R0201
import json
import os
import tempfile
from pathlib import Path
from urllib.parse import urlparse

import pandas as pd
from sklearn.metrics import confusion_matrix

from pytorch_kfp_components.components.base.base_executor import BaseExecutor
from pytorch_kfp_components.components.minio.component import MinIO
from pytorch_kfp_components.types import standard_component_specs


class Executor(BaseExecutor):  # pylint: disable=R0903
    """Visualization Executor Class."""

    def __init__(self):
        super(Executor, self).__init__()  # pylint: disable=R1725
        self.mlpipeline_ui_metadata = None
        self.mlpipeline_metrics = None

    def _write_ui_metadata(
        self, metadata_filepath, metadata_dict, key="outputs"
    ):
        """Function to write the metadata to UI."""
        if not os.path.exists(metadata_filepath):
            metadata = {key: [metadata_dict]}
        else:
            with open(metadata_filepath) as fp:
                metadata = json.load(fp)
                metadata_outputs = metadata[key]
                metadata_outputs.append(metadata_dict)

        print("Writing to file: {}".format(metadata_filepath))
        with open(metadata_filepath, "w") as fp:
            json.dump(metadata, fp)

    def _generate_markdown(self, markdown_dict):
        """Generates a markdown.

        Args:
            markdown_dict : dict of markdown specifications
        """
        source_str = json.dumps(
            markdown_dict["source"], sort_keys=True, indent=4
        )
        source = f"```json \n {source_str} ```"
        markdown_metadata = {
            "storage": markdown_dict["storage"],
            "source": source,
            "type": "markdown",
        }

        self._write_ui_metadata(
            metadata_filepath=self.mlpipeline_ui_metadata,
            metadata_dict=markdown_metadata,
        )

    def _generate_confusion_matrix_metadata(
        self, confusion_matrix_path, classes
    ):
        """Generates the confusion matrix metadata and writes in ui."""
        print("Generating Confusion matrix Metadata")
        metadata = {
            "type": "confusion_matrix",
            "format": "csv",
            "schema": [
                {"name": "target", "type": "CATEGORY"},
                {"name": "predicted", "type": "CATEGORY"},
                {"name": "count", "type": "NUMBER"},
            ],
            "source": confusion_matrix_path,
            "labels": list(map(str, classes)),
        }

        self._write_ui_metadata(
            metadata_filepath=self.mlpipeline_ui_metadata,
            metadata_dict=metadata,
        )

    def _upload_confusion_matrix_to_minio(
        self, confusion_matrix_url, confusion_matrix_output_path
    ):
        """Uploads the generated confusion matrix to minio"""
        parse_obj = urlparse(confusion_matrix_url, allow_fragments=False)
        bucket_name = parse_obj.netloc
        folder_name = str(parse_obj.path).lstrip("/")

        # TODO:  # pylint: disable=W0511
        endpoint = "minio-service.kubeflow:9000"
        MinIO(
            source=confusion_matrix_output_path,
            bucket_name=bucket_name,
            destination=folder_name,
            endpoint=endpoint,
        )

    def _generate_confusion_matrix(
        self, confusion_matrix_dict
    ):  # pylint: disable=R0914
        """Generates confusion matrix in minio."""
        actuals = confusion_matrix_dict["actuals"]
        preds = confusion_matrix_dict["preds"]
        confusion_matrix_url = confusion_matrix_dict["url"]

        # Generating confusion matrix
        df = pd.DataFrame(
            list(zip(actuals, preds)), columns=["target", "predicted"]
        )
        vocab = list(df["target"].unique())
        cm = confusion_matrix(df["target"], df["predicted"], labels=vocab)
        data = []
        for target_index, target_row in enumerate(cm):
            for predicted_index, count in enumerate(target_row):
                data.append(
                    (vocab[target_index], vocab[predicted_index], count)
                )

        confusion_matrix_df = pd.DataFrame(
            data, columns=["target", "predicted", "count"]
        )

        confusion_matrix_output_dir = str(tempfile.mkdtemp())
        confusion_matrix_output_path = os.path.join(
            confusion_matrix_output_dir, "confusion_matrix.csv"
        )
        # saving confusion matrix
        confusion_matrix_df.to_csv(
            confusion_matrix_output_path, index=False, header=False
        )

        self._upload_confusion_matrix_to_minio(
            confusion_matrix_url=confusion_matrix_url,
            confusion_matrix_output_path=confusion_matrix_output_path,
        )

        # Generating metadata
        self._generate_confusion_matrix_metadata(
            confusion_matrix_path=os.path.join(
                confusion_matrix_url, "confusion_matrix.csv"
            ),
            classes=vocab,
        )

    def _visualize_accuracy_metric(self, accuracy):
        """Generates the visualization for accuracy."""
        metadata = {
            "name": "accuracy-score",
            "numberValue": accuracy,
            "format": "PERCENTAGE",
        }
        self._write_ui_metadata(
            metadata_filepath=self.mlpipeline_metrics,
            metadata_dict=metadata,
            key="metrics",
        )

    def _get_fn_args(self, input_dict: dict, exec_properties: dict):
        """Extracts the confusion matrix dict, test accuracy, markdown from the
        input dict and mlpipeline ui metadata & metrics from exec_properties.

        Args:
            input_dict : a dictionary of inputs,
                        example: confusion matrix dict, markdown
            exe_properties : a dict of execution properties
                           example : mlpipeline_ui_metadata
        Returns:
            confusion_matrix_dict : dict of confusion metrics
            test_accuracy : model test accuracy metrics
            markdown : markdown dict
            mlpipeline_ui_metadata : path of ui metadata
            mlpipeline_metrics : metrics to be uploaded
        """
        confusion_matrix_dict = input_dict.get(
            standard_component_specs.VIZ_CONFUSION_MATRIX_DICT
        )
        test_accuracy = input_dict.get(
            standard_component_specs.VIZ_TEST_ACCURACY
        )
        markdown = input_dict.get(standard_component_specs.VIZ_MARKDOWN)

        mlpipeline_ui_metadata = exec_properties.get(
            standard_component_specs.VIZ_MLPIPELINE_UI_METADATA
        )
        mlpipeline_metrics = exec_properties.get(
            standard_component_specs.VIZ_MLPIPELINE_METRICS
        )

        return (
            confusion_matrix_dict,
            test_accuracy,
            markdown,
            mlpipeline_ui_metadata,
            mlpipeline_metrics,
        )

    def _set_defalt_mlpipeline_path(
        self, mlpipeline_ui_metadata: str, mlpipeline_metrics: str
    ):
        """Sets the default mlpipeline path."""

        if mlpipeline_ui_metadata:
            Path(os.path.dirname(mlpipeline_ui_metadata)).mkdir(
                parents=True, exist_ok=True
            )
        else:
            mlpipeline_ui_metadata = "/mlpipeline-ui-metadata.json"

        if mlpipeline_metrics:
            Path(os.path.dirname(mlpipeline_metrics)).mkdir(
                parents=True, exist_ok=True
            )
        else:
            mlpipeline_metrics = "/mlpipeline-metrics.json"

        return mlpipeline_ui_metadata, mlpipeline_metrics

    def Do(self, input_dict: dict, output_dict: dict, exec_properties: dict):
        """Executes the visualization process and uploads to minio
        Args:
             input_dict : a dictionary of inputs,
                         example: confusion matrix dict, markdown
             output_dict :
             exec_properties : a dict of execution properties
                            example : mlpipeline_ui_metadata
        """

        (
            confusion_matrix_dict,
            test_accuracy,
            markdown,
            mlpipeline_ui_metadata,
            mlpipeline_metrics,
        ) = self._get_fn_args(
            input_dict=input_dict, exec_properties=exec_properties
        )

        (
            self.mlpipeline_ui_metadata,
            self.mlpipeline_metrics,
        ) = self._set_defalt_mlpipeline_path(
            mlpipeline_ui_metadata=mlpipeline_ui_metadata,
            mlpipeline_metrics=mlpipeline_metrics,
        )

        if not (confusion_matrix_dict or test_accuracy or markdown):
            raise ValueError(
                "Any one of these keys should be set - "
                "confusion_matrix_dict, test_accuracy, markdown"
            )

        if confusion_matrix_dict:
            self._generate_confusion_matrix(
                confusion_matrix_dict=confusion_matrix_dict,
            )

        if test_accuracy:
            self._visualize_accuracy_metric(accuracy=test_accuracy)

        if markdown:
            self._generate_markdown(markdown_dict=markdown)

        output_dict[
            standard_component_specs.VIZ_MLPIPELINE_UI_METADATA
        ] = self.mlpipeline_ui_metadata
        output_dict[
            standard_component_specs.VIZ_MLPIPELINE_METRICS
        ] = self.mlpipeline_metrics
