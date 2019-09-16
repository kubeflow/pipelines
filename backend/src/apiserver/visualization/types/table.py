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

# gcsfs is required for pandas GCS integration.
import gcsfs
from itables import show
# itables is requires as importing it changes the way pandas DataFrames are
# rendered.
import itables.interactive
from itables.javascript import load_datatables
import itables.options as opts
import pandas as pd
from tensorflow.python.lib.io import file_io

# Forcefully load required JavaScript and CSS for datatables.
load_datatables()

# Remove maxByte limit to prevent issues where entire table cannot be rendered
# due to size of data.
opts.maxBytes = 0

dfs = []
files = file_io.get_matching_files(source)

# Read data from file and write it to a DataFrame object.
if not variables.get("headers", False):
    # If no headers are provided, use the first row as headers
    for f in files:
        dfs.append(pd.read_csv(f))
else:
    # If headers are provided, do not set headers for DataFrames
    for f in files:
        dfs.append(pd.read_csv(f, header=None))

# Display DataFrame as output.
df = pd.concat(dfs)
if variables.get("headers", False):
    df.columns = variables.get("headers")
show(df)
