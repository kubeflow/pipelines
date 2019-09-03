# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
from pathlib import Path
from bokeh.layouts import row
from bokeh.plotting import figure
from bokeh.io import output_notebook, show
from bokeh.models import HoverTool
# gcsfs is required for pandas GCS integration.
import gcsfs
import pandas as pd
from sklearn.metrics import roc_curve
from tensorflow.python.lib.io import file_io

# The following variables are provided through dependency injection. These
# variables come from the specified input path and arguments provided by the
# API post request.
#
# is_generated
# source
# target_lambda
# trueclass
# true_score_column

if not variables.get("is_generated", False):
    # Create data from specified csv file(s).
    # The schema file provides column names for the csv file that will be used
    # to generate the roc curve.
    schema_file = Path(source) / "schema.json"
    schema = json.loads(file_io.read_file_to_string(schema_file))
    names = [x["name"] for x in schema]

    dfs = []
    files = file_io.get_matching_files(source)
    for f in files:
        dfs.append(pd.read_csv(f, names=names))

    df = pd.concat(dfs)
    if variables.get("target_lambda", False):
        df["target"] = df.apply(eval(variables.get("target_lambda", "")), axis=1)
    else:
        df["target"] = df["target"].apply(lambda x: 1 if x == variables.get("trueclass", "true") else 0)
    fpr, tpr, thresholds = roc_curve(df["target"], df[variables.get("true_score_column", "true")])
    df = pd.DataFrame({"fpr": fpr, "tpr": tpr, "thresholds": thresholds})
else:
    # Load data from generated csv file.
    df = pd.read_csv(
        source,
        header=None,
        names=["fpr", "tpr", "thresholds"]
    )

# Create visualization.
output_notebook()

p = figure(tools="pan,wheel_zoom,box_zoom,reset,hover,previewsave")
p.line("fpr", "tpr", line_width=2, source=df)

hover = p.select(dict(type=HoverTool))
hover.tooltips = [("Threshold", "@thresholds")]

show(row(p, sizing_mode="scale_width"))
