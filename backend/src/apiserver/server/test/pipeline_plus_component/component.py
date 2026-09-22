# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Regenerate the native component and deterministic archive-selection fixtures."""

import gzip
import io
from pathlib import Path
import tarfile
import zipfile

from kfp import compiler
from kfp import dsl


@dsl.container_component
def whalesay(param1: str, param2: str):
    return dsl.ContainerSpec(
        image='docker/whalesay:latest',
        command=['cowsay'],
        args=[param1, param2])


def regenerate():
    directory = Path(__file__).parent
    compiler.Compiler().compile(whalesay, str(directory / 'component.yaml'))
    # Put the decoy first so choosing the first YAML cannot pass the tests.
    names = ('component.yaml', 'pipeline.yaml')
    with zipfile.ZipFile(directory / 'pipeline_plus_component.zip',
                         'w') as archive:
        for name in names:
            entry = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
            entry.external_attr = 0o100644 << 16
            archive.writestr(entry, (directory / name).read_bytes())
    with (directory / 'pipeline_plus_component.tar.gz').open('wb') as output:
        with gzip.GzipFile(
                filename='', fileobj=output, mode='wb', mtime=0) as compressed:
            with tarfile.open(
                    fileobj=compressed, mode='w',
                    format=tarfile.USTAR_FORMAT) as archive:
                for name in names:
                    contents = (directory / name).read_bytes()
                    entry = tarfile.TarInfo(name)
                    entry.mode = 0o644
                    entry.size = len(contents)
                    archive.addfile(entry, io.BytesIO(contents))


if __name__ == '__main__':
    regenerate()
