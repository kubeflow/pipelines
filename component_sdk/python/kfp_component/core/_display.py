# Copyright 2018 Google LLC
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
import threading
import logging

_OUTPUT_PATH = os.environ.get('KFP_UI_METADATA_PATH', '/mlpipeline-ui-metadata.json')
_OUTPUT_FILE_LOCK = threading.Lock()

def display(obj):
    """Display an object to KFP UI.

    Args:
        obj (object): the object to output the display metadata. It follows same 
            convention defined by IPython display API. The currently supported representation
            functions:
            
            * `_repr_html_`: it returns a html content which will be converted into a 
                web-app metadata to KFP UI.
            * `_repr_kfpmetadata_`: it returns a KFP metadata json object, which follows
                the convention from https://www.kubeflow.org/docs/pipelines/output-viewer/.

            The supported builtin objects are HTML, Tensorboard, Link.
    """
    obj_dir = dir(obj)
    if '_repr_html_' in obj_dir:
        display_html(obj)

    if '_repr_kfpmetadata_' in obj_dir:
        display_kfpmetadata(obj)

def display_html(obj):
    """Display html representation to KFP UI.
    """
    if '_repr_html_' not in dir(obj):
        raise ValueError('_repr_html_ function is not present.')
    html = obj._repr_html_()
    _output_ui_metadata({
        'type': 'web-app',
        'html': html
    })

def display_kfpmetadata(obj):
    """Display from KFP UI metadata
    """
    if '_repr_kfpmetadata_' not in dir(obj):
        raise ValueError('_repr_kfpmetadata_ function is not present.')
    kfp_metadata = obj._repr_kfpmetadata_()
    _output_ui_metadata(kfp_metadata)

def _output_ui_metadata(output):
    logging.info('Dumping metadata: {}'.format(output))
    with _OUTPUT_FILE_LOCK:
        metadata = {}
        if os.path.isfile(_OUTPUT_PATH):
            with open(_OUTPUT_PATH, 'r') as f:
                metadata = json.load(f)

        with open(_OUTPUT_PATH, 'w') as f:
            if 'outputs' not in metadata:
                metadata['outputs'] = []
            metadata['outputs'].append(output)
            json.dump(metadata, f)

class HTML(object):
    """Class to hold html raw data.
    """
    def __init__(self, data):
        self._html = data

    def _repr_html_(self):
        return self._html

class Tensorboard(object):
    """Class to hold tensorboard metadata.
    """
    def __init__(self, job_dir):
        self._job_dir = job_dir

    def _repr_kfpmetadata_(self):
        return {
            'type': 'tensorboard',
            'source': self._job_dir
        }

class Link(HTML):
    """Class to hold an HTML hyperlink data.
    """
    def __init__(self, href, text):
        super(Link, self).__init__(
            '<a href="{}">{}</a>'.format(href, text))