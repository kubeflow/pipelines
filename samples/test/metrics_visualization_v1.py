# Copyright 2021 The Kubeflow Authors
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

import kfp.dsl as dsl
from ..core.visualization.confusion_matrix import confusion_visualization
from ..core.visualization.html import html_visualization
from ..core.visualization.markdown import markdown_visualization
from ..core.visualization.roc import roc_visualization
from ..core.visualization.table import table_visualization

# Note: This test is to verify that visualization metrics on V1 is runnable by KFP.
# However, this pipeline is only runnable on V1 mode, but not V2 compatible mode.

@dsl.pipeline(
    name='metrics-visualization-v1-pipeline')
def metrics_visualization_v1_pipeline():
    confusion_visualization_task = confusion_visualization()
    html_visualization_task = html_visualization("")
    markdown_visualization_task = markdown_visualization()
    roc_visualization_task = roc_visualization()
    table_visualization_task = table_visualization()
