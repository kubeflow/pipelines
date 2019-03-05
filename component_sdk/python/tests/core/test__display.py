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

from kfp_component.core import display

import mock
import unittest

@mock.patch('kfp_component.core._display.json')
@mock.patch('kfp_component.core._display.os')
@mock.patch('kfp_component.core._display.open')
class DisplayTest(unittest.TestCase):

    def test_display_markdown(self, mock_open, mock_os, mock_json):
        mock_os.path.isfile.return_value = False

        display.display(display.Markdown('# test'))

        mock_json.dump.assert_called_with({
            'outputs': [{
                'type': 'markdown',
                'source': '# test',
                'storage': 'inline'
            }]
        }, mock.ANY)

    def test_display_markdown_append(self, mock_open, mock_os, mock_json):
        mock_os.path.isfile.return_value = True
        mock_json.load.return_value = {
            'outputs': [{
                'type': 'markdown',
                'source': '# test 1',
                'storage': 'inline'
            }]
        }

        display.display(display.Markdown('# test 2'))

        mock_json.dump.assert_called_with({
            'outputs': [{
                'type': 'markdown',
                'source': '# test 1',
                'storage': 'inline'
            },{
                'type': 'markdown',
                'source': '# test 2',
                'storage': 'inline'
            }]
        }, mock.ANY)

    def test_display_tensorboard(self, mock_open, mock_os, mock_json):
        mock_os.path.isfile.return_value = False

        display.display(display.Tensorboard('gs://job/dir'))

        mock_json.dump.assert_called_with({
            'outputs': [{
                'type': 'tensorboard',
                'source': 'gs://job/dir'
            }]
        }, mock.ANY)

    def test_display_link(self, mock_open, mock_os, mock_json):
        mock_os.path.isfile.return_value = False

        display.display(display.Link('https://test/link', 'Test Link'))

        mock_json.dump.assert_called_with({
            'outputs': [{
                'type': 'markdown',
                'source': '## [Test Link](https://test/link)',
                'storage': 'inline'
            }]
        }, mock.ANY)

    def test___repr__(self, mock_open, mock_os, mock_json):
        self.assertEqual('# Title', str(display.Markdown('# Title')))
        self.assertEqual('Open Tensorboard at: gs://trained/model/', 
            str(display.Tensorboard('gs://trained/model/')))
        self.assertEqual('title: https://test/uri', 
            str(display.Link('https://test/uri', 'title')))
