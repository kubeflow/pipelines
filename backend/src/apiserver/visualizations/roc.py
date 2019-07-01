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


import os
import json
from tensorflow.python.lib.io import file_io
import pandas as pd
from sklearn.metrics import roc_curve
from bokeh.plotting import figure
from bokeh.io import output_notebook, show
from bokeh.models import HoverTool

# Expected arguments:
# predictions
#
# Optional arguments:
# target_lambda
#
# trueclass
# true_score_column
#
# either target_lambda OR trueclass AND true_score_column are required

# Load data
schema_file = os.path.join(os.path.dirname(predictions),
                           'schema.json')
schema = json.loads(file_io.read_file_to_string(schema_file))
names = [x['name'] for x in schema]

dfs = []
files = file_io.get_matching_files(predictions)
for file in files:
    with file_io.FileIO(file, 'r') as f:
        dfs.append(pd.read_csv(f, names=names))

df = pd.concat(dfs)
if target_lambda:
    df['target'] = df.apply(eval(target_lambda), axis=1)
else:
    df['target'] = df['target'].apply(lambda x: 1 if x == trueclass else 0)
fpr, tpr, thresholds = roc_curve(df['target'], df[true_score_column])
source = pd.DataFrame({'fpr': fpr, 'tpr': tpr, 'thresholds': thresholds})

# Create visualization
output_notebook()

p = figure(width=500, height=500,
           tools="pan,wheel_zoom,box_zoom,reset,hover,previewsave")
p.line('fpr', 'tpr', line_width=2, source=source)

hover = p.select(dict(type=HoverTool))
hover.tooltips = [("Threshold", "@thresholds")]

show(p)
