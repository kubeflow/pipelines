# -*- coding: utf-8 -*-
# snapshottest: v1 - https://goo.gl/zC4yUc
from __future__ import unicode_literals

from snapshottest import Snapshot


snapshots = Snapshot()

snapshots['TestExporterMethods::test_create_cell_from_args_with_multiple_args 1'] = '''
<div class="output_wrapper">
<div class="output">


<div class="output_area">

    <div class="prompt"></div>


<div class="output_subarea output_stream output_stdout output_text">
<pre>[&#39;gs://ml-pipeline/data.csv&#39;, &#34;lambda x: (x[&#39;target&#39;] &gt; x[&#39;fare&#39;] * 0.2)&#34;]
</pre>
</div>
</div>

</div>
</div>



'''

snapshots['TestExporterMethods::test_create_cell_from_args_with_no_args 1'] = '''
<div class="output_wrapper">
<div class="output">


<div class="output_area">

    <div class="prompt"></div>


<div class="output_subarea output_stream output_stdout output_text">
<pre>{}
</pre>
</div>
</div>

</div>
</div>



'''

snapshots['TestExporterMethods::test_create_cell_from_args_with_one_arg 1'] = '''
<div class="output_wrapper">
<div class="output">


<div class="output_area">

    <div class="prompt"></div>


<div class="output_subarea output_stream output_stdout output_text">
<pre>[&#39;gs://ml-pipeline/data.csv&#39;]
</pre>
</div>
</div>

</div>
</div>



'''

snapshots['TestExporterMethods::test_create_cell_from_file 1'] = '''"""
test.py is used for test_server.py as a way to test the tornado web server
without having a reliance on the validity of visualizations. It does not serve
as a valid visualization and is only used for testing purposes.
"""

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

x = 2
print(x)
'''

snapshots['TestExporterMethods::test_generate_html_from_notebook 1'] = '''
<div class="output_wrapper">
<div class="output">


<div class="output_area">

    <div class="prompt"></div>


<div class="output_subarea output_stream output_stdout output_text">
<pre>2
</pre>
</div>
</div>

</div>
</div>



'''
