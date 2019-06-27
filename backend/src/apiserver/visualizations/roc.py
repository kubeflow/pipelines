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

import pandas as pd
from bokeh.plotting import figure
from bokeh.io import output_notebook, show
from bokeh.models import HoverTool

with open('roc.csv') as f:
  df = pd.read_csv(f, names=['fpr', 'tpr', 'thresholds'])

output_notebook()

p = figure(width=500, height=500,
           tools="pan,wheel_zoom,box_zoom,reset,hover,previewsave")
p.line('fpr', 'tpr', line_width=2, source=df)

hover = p.select(dict(type=HoverTool))
hover.tooltips = [("Threshold", "@thresholds")]

show(p)
