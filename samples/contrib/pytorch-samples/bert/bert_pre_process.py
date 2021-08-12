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
"""Bert Pre preprocess script."""
import os
import subprocess
from argparse import ArgumentParser
from pathlib import Path

import pyarrow.csv as pv
import pyarrow.parquet as pq
from torchtext.utils import download_from_url, extract_archive
from pytorch_kfp_components.components.visualization.component import (
    Visualization,
)

if __name__ == "__main__":

    parser = ArgumentParser()
    parser.add_argument(
        "--dataset_url",
        default=
        "https://kubeflow-dataset.s3.us-east-2.amazonaws.com/ag_news_csv.tar.gz", #pylint: disable=line-too-long
        type=str,
        help="URL to download AG News dataset",
    )

    parser.add_argument(
        "--output_path",
        default="output/processing",
        type=str,
        help="Path to write the ag news dataset",
    )

    parser.add_argument(
        "--mlpipeline_ui_metadata",
        type=str,
        help="Path to write mlpipeline-ui-metadata.json",
    )

    args = vars(parser.parse_args())

    dataset_url = args["dataset_url"]
    output_path = args["output_path"]

    Path(output_path).mkdir(parents=True, exist_ok=True)

    dataset_tar = download_from_url(dataset_url, root="./")
    extracted_files = extract_archive(dataset_tar)

    ag_news_csv = pv.read_csv("ag_news_csv/train.csv")

    pq.write_table(
        ag_news_csv, os.path.join(output_path, "ag_news_data.parquet")
    )

    entry_point = ["ls", "-R", output_path]
    run_code = subprocess.run(entry_point, stdout=subprocess.PIPE)  #pylint: disable=subprocess-run-check
    print(run_code.stdout)

    visualization_arguments = {
        "inputs": {
            "dataset_url": args["dataset_url"]
        },
        "output": {
            "mlpipeline_ui_metadata": args["mlpipeline_ui_metadata"],
        },
    }

    markdown_dict = {"storage": "inline", "source": visualization_arguments}

    print("Visualization arguments: ", markdown_dict)

    visualization = Visualization(
        mlpipeline_ui_metadata=args["mlpipeline_ui_metadata"],
        markdown=markdown_dict,
    )

    df = ag_news_csv.to_pandas()
    df_counts = df.iloc[:, 0].value_counts()
    print(df.iloc[:, 0].value_counts())
    label_names = ["World", "Sports", "Business", "Sci/Tech"]
    label_dict = {}
    total_count = len(df)
    for key, value in df_counts.iteritems():
        label_name = label_names[key - 1]
        label_dict[label_name.upper()] = value

    label_dict["TOTAL_COUNT"] = total_count

    markdown_dict = {"storage": "inline", "source": label_dict}

    visualization = Visualization(
        mlpipeline_ui_metadata=args["mlpipeline_ui_metadata"],
        markdown=markdown_dict,
    )
