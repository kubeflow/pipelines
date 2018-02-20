# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import sys

license = [
    'Copyright 2018 Google LLC',
    '',
    'Licensed under the Apache License, Version 2.0 (the "License");',
    'you may not use this file except in compliance with the License.',
    'You may obtain a copy of the License at',
    '',
    'https://www.apache.org/licenses/LICENSE-2.0',
    '',
    'Unless required by applicable law or agreed to in writing, software',
    'distributed under the License is distributed on an "AS IS" BASIS,',
    'WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.',
    'See the License for the specific language governing permissions and',
    'limitations under the License.',
]

asterisk_comment_chars = ['/*', ' * ', ' */']
comment_style = {
    '.css': {'block': True, 'chars': asterisk_comment_chars, },
    '.go': {'block': False, 'chars':['// '], },
    '.html': {'block': True, 'chars': ['<!-- ', '  ', ' -->'], },
    '.js': {'block': True, 'chars': asterisk_comment_chars, },
    '.py': {'block': False, 'chars': ['# '], },
    '.sh': {'block': False, 'chars': ['# '], },
    '.ts': {'block': True, 'chars': asterisk_comment_chars, },
}


def fill_header(fname, extension):
    with open(fname, 'r+') as f:
        contents = f.read()
        if 'Licensed under the Apache License' not in contents:
            f.seek(0, 0)
            block = comment_style[extension]['block']
            pre = (comment_style[extension]['chars']
                   [0] + '\n') if block else ''
            char = comment_style[extension]['chars'][1 if block else 0]
            post = ('\n' + comment_style[extension]
                    ['chars'][2]) if block else ''

            commented_header = pre + \
                '\n'.join([(char + l).rstrip() for l in license]) + post
            f.write(commented_header + '\n\n' + contents)


include_dirs = ['..']
exclude_dirs = ['../frontend/node_modules', '../frontend/bower_components']

for d in include_dirs:
    sys.stdout.write('Working on ' + d + '\n')
    for dirpath, dirs, files in os.walk(d):
        dirs[:] = [d for d in dirs if os.path.join(
            dirpath, d) not in exclude_dirs]
        for f in files:
            fname = os.path.join(dirpath, f)
            _, ext = os.path.splitext(fname)
            if ext in comment_style.keys():
                sys.stdout.write(fname + '\n')
                fill_header(fname, ext)
