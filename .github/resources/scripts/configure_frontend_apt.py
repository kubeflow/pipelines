# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Bound browser dependency acquisition on disposable hosted Ubuntu runners."""
import os
from pathlib import Path
import re

AZURE_MIRROR = re.compile(
    r'https?://azure\.archive\.ubuntu\.com/ubuntu(?=[/\s]|$)')
APT_LIMITS = '''Acquire::http::Timeout "30";
Acquire::https::Timeout "30";
Acquire::Retries "2";
'''


def configure(root):
    sources = [root / 'apt-mirrors.txt', root / 'sources.list']
    for pattern in ('*.list', '*.sources'):
        sources.extend((root / 'sources.list.d').glob(pattern))
    for source in sources:
        if source.is_file():
            before = source.read_text()
            after = AZURE_MIRROR.sub('https://archive.ubuntu.com/ubuntu',
                                     before)
            if after != before:
                source.write_text(after)
                print(
                    f'Replaced unavailable Azure Ubuntu mirror in {source.name}'
                )
    (root / 'apt.conf.d' / '99-kfp-browser-acquisition').write_text(APT_LIMITS)


def main():
    if not (os.environ.get('CI') == 'true' and
            os.environ.get('GITHUB_ACTIONS') == 'true' and
            os.environ.get('RUNNER_ENVIRONMENT') == 'github-hosted'):
        raise RuntimeError(
            'APT configuration requires a disposable GitHub runner')
    configure(Path('/etc/apt'))


if __name__ == '__main__':
    main()
