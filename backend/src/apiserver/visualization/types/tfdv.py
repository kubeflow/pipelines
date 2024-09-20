# Copyright 2019 The Kubeflow Authors
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

import base64
import tensorflow_data_validation as tfdv
from IPython.display import display
from IPython.display import HTML
from tensorflow_metadata.proto.v0 import statistics_pb2
from typing import Text

# The following variables are provided through dependency injection. These
# variables come from the specified input path and arguments provided by the
# API post request.
#
# source

# train_stats = tfdv.generate_statistics_from_csv(data_location=source)
# tfdv.visualize_statistics(train_stats) 

def get_statistics_html(
    lhs_statistics: statistics_pb2.DatasetFeatureStatisticsList
) -> Text:
  """Build the HTML for visualizing the input statistics using Facets.
  Args:
    lhs_statistics: A DatasetFeatureStatisticsList protocol buffer.
  Returns:
    HTML to be embedded for visualization.
  Raises:
    TypeError: If the input argument is not of the expected type.
    ValueError: If the input statistics protos does not have only one dataset.
  """

  rhs_statistics = None
  lhs_name = 'lhs_statistics'
  rhs_name = 'rhs_statistics'

  if not isinstance(lhs_statistics,
                    statistics_pb2.DatasetFeatureStatisticsList):
    raise TypeError(
        'lhs_statistics is of type %s, should be '
        'a DatasetFeatureStatisticsList proto.' % type(lhs_statistics).__name__)

  if len(lhs_statistics.datasets) != 1:
    raise ValueError('lhs_statistics proto contains multiple datasets. Only '
                     'one dataset is currently supported.')

  if lhs_statistics.datasets[0].name:
    lhs_name = lhs_statistics.datasets[0].name

  # Add lhs stats.
  combined_statistics = statistics_pb2.DatasetFeatureStatisticsList()
  lhs_stats_copy = combined_statistics.datasets.add()
  lhs_stats_copy.MergeFrom(lhs_statistics.datasets[0])
  lhs_stats_copy.name = lhs_name

  protostr = base64.b64encode(
      combined_statistics.SerializeToString()).decode('utf-8')

  # pylint: disable=line-too-long
  # Note that in the html template we currently assign a temporary id to the
  # facets element and then remove it once we have appended the serialized proto
  # string to the element. We do this to avoid any collision of ids when
  # displaying multiple facets output in the notebook.
  html_template = r"""<iframe id='facets-iframe' width="100%" height="500px"></iframe>
        <script>
        facets_iframe = document.getElementById('facets-iframe');
        facets_html = '<script src="https://cdnjs.cloudflare.com/ajax/libs/webcomponentsjs/1.3.3/webcomponents-lite.js"><\/script><link rel="import" href="https://raw.githubusercontent.com/PAIR-code/facets/master/facets-dist/facets-jupyter.html"><facets-overview proto-input="protostr"></facets-overview>';
        facets_iframe.srcdoc = facets_html;
         facets_iframe.id = "";
         setTimeout(() => {
           facets_iframe.setAttribute('height', facets_iframe.contentWindow.document.body.offsetHeight + 'px')
         }, 1500)
         </script>"""
  # pylint: enable=line-too-long
  html = html_template.replace('protostr', protostr)

  return html
  
stats = tfdv.load_statistics(source)
html = get_statistics_html(stats)
display(HTML(html))
