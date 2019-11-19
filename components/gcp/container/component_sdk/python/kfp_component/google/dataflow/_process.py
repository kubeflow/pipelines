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
import subprocess
import logging

class Process:
    def __init__(self, cmd):
        self._cmd = cmd
        self.process = subprocess.Popen(cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            close_fds=True,
            shell=False)

    def read_lines(self):
        # stdout will end with empty bytes when process exits.
        for line in iter(self.process.stdout.readline, b''):
            logging.info('subprocess: {}'.format(line))
            yield line

    def wait_and_check(self):
        for _ in self.read_lines():
            pass
        self.process.stdout.close()
        return_code = self.process.wait()
        logging.info('Subprocess exit with code {}.'.format(
            return_code))
        if return_code:
            raise subprocess.CalledProcessError(return_code, self._cmd)