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
            
            * `_repr_markdown_`: it returns a markdown content which will be converted into a 
                web-app metadata to KFP UI.
            * `_repr_kfpmetadata_`: it returns a KFP metadata json object, which follows
                the convention from https://www.kubeflow.org/docs/pipelines/output-viewer/.

            The supported builtin objects are markdown, Tensorboard, Link.
    """
    obj_dir = dir(obj)
    if '_repr_markdown_' in obj_dir:
        display_markdown(obj)

    if '_repr_kfpmetadata_' in obj_dir:
        display_kfpmetadata(obj)

    logging.info(str(obj))

def display_markdown(obj):
    """Display markdown representation to KFP UI.
    """
    if '_repr_markdown_' not in dir(obj):
        raise ValueError('_repr_markdown_ function is not present.')
    markdown = obj._repr_markdown_()
    _output_ui_metadata({
        'type': 'markdown',
        'source': markdown,
        'storage': 'inline'
    })

def display_kfpmetadata(obj):
    """Display from KFP UI metadata
    """
    if '_repr_kfpmetadata_' not in dir(obj):
        raise ValueError('_repr_kfpmetadata_ function is not present.')
    kfp_metadata = obj._repr_kfpmetadata_()
    _output_ui_metadata(kfp_metadata)

def _output_ui_metadata(output):
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

class Markdown(object):
    """Class to hold markdown raw data.
    """
    def __init__(self, data):
        self._data = data

    def _repr_markdown_(self):
        return self._data

    def __repr__(self):
        return self._data

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

    def __repr__(self):
        return 'Open Tensorboard at: {}'.format(self._job_dir)

class Link(Markdown):
    """Class to hold an markdown hyperlink data.
    """
    def __init__(self, href, text):
        super(Link, self).__init__(
            '## [{}]({})'.format(text, href))
        self._href = href
        self._text = text

    def __repr__(self):
        return '{}: {}'.format(self._text, self._href)