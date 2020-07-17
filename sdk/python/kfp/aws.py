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

import os
import io
import platform
import re
import sys
import stat
import requests
import platform
import time
import logging
import zipfile
from selenium.webdriver import Chrome
from selenium.webdriver.chrome.options import Options


DRIVER_ROOT_URL = 'https://chromedriver.storage.googleapis.com'
DEFAULT_CHROME_DRIVER_PATH = os.path.join(os.path.expanduser('~'), '.config', 'kfp', 'chromedriver')
SUPPORTED_PLATFORMS = {
    'Windows': 'win32',
    'Darwin': 'mac64',
    'Linux': 'linux64'
}

def use_aws_secret(secret_name='aws-secret', aws_access_key_id_name='AWS_ACCESS_KEY_ID', aws_secret_access_key_name='AWS_SECRET_ACCESS_KEY'):
    """An operator that configures the container to use AWS credentials.

        AWS doesn't create secret along with kubeflow deployment and it requires users
        to manually create credential secret with proper permissions.
        ---
        apiVersion: v1
        kind: Secret
        metadata:
          name: aws-secret
        type: Opaque
        data:
          AWS_ACCESS_KEY_ID: BASE64_YOUR_AWS_ACCESS_KEY_ID
          AWS_SECRET_ACCESS_KEY: BASE64_YOUR_AWS_SECRET_ACCESS_KEY
    """

    def _use_aws_secret(task):
        from kubernetes import client as k8s_client
        (
            task.container
                .add_env_variable(
                    k8s_client.V1EnvVar(
                        name='AWS_ACCESS_KEY_ID',
                        value_from=k8s_client.V1EnvVarSource(
                            secret_key_ref=k8s_client.V1SecretKeySelector(
                                name=secret_name,
                                key=aws_access_key_id_name
                            )
                        )
                    )
                )
                .add_env_variable(
                    k8s_client.V1EnvVar(
                        name='AWS_SECRET_ACCESS_KEY',
                        value_from=k8s_client.V1EnvVarSource(
                            secret_key_ref=k8s_client.V1SecretKeySelector(
                                name=secret_name,
                                key=aws_secret_access_key_name
                            )
                        )
                    )
                )
        )
        return task

    return _use_aws_secret

class DiskCache:
    """ Helper class for caching data on disk.
    """

    _DEFAULT_CACHE_ROOT = os.path.join(os.path.expanduser('~'), '.config', 'kfp')

    def __init__(self, name):
        self.name = name
        self.cache_path = os.path.join(self._DEFAULT_CACHE_ROOT, self.name)
        if not os.path.exists(self._DEFAULT_CACHE_ROOT):
            os.makedirs(self._DEFAULT_CACHE_ROOT)

    def save(self, content):
        """ cache content in a known location

        Args:
            content (str): content to be cached
        """

        with open(self.cache_path, 'w') as cacheFile:
            cacheFile.write(content)

    def get(self, expires_in=None):
        """[summary]

        Args:
            expires_in (int, optional): Time in seconds since last modifed
                when the cache is valid. Defaults to None which means for ever.

        Returns:
            (str): retrived cached content or None if not found
        """

        try:
            if expires_in and time.time() - os.path.getmtime(self.cache_path) > expires_in:
                return

            with open(self.cache_path) as cacheFile:
                return cacheFile.read()
        except FileNotFoundError:
            return

def os_name():
    pl = sys.platform
    if pl == "linux" or pl == "linux2":
        return "linux"
    elif pl == "darwin":
        return "mac"
    elif pl == "win32":
        return "win"

def get_chrome_version():
    pattern = r'\d+\.\d+\.\d+'

    cmd_mapping = {
        "linux": 'google-chrome --version || google-chrome-stable --version',
        "mac": r'/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --version',
        "win": r'reg query "HKEY_CURRENT_USER\Software\Google\Chrome\BLBeacon" /v version'
    }

    cmd = cmd_mapping[os_name()]
    stdout = os.popen(cmd).read()
    version = re.search(pattern, stdout)
    if not version:
        raise ValueError('Could not get version for Chrome with this command: {}'.format(cmd))
    current_version = version.group(0)
    return current_version


def get_latest_release_version():
    latest_release_url = "{}/LATEST_RELEASE".format(DRIVER_ROOT_URL)
    chrome_version = get_chrome_version()

    logging.info("Get LATEST driver version for {}".format(chrome_version))
    resp = requests.get("{}_{}".format(latest_release_url,chrome_version))
    return resp.text.rstrip()

def download_driver():
    """ Helper function for downloading the driver
    """

    osy = SUPPORTED_PLATFORMS[platform.system()]
    chrome_release_version = get_latest_release_version()
    driver_url = '{}/{}/chromedriver_{}.zip'.format(DRIVER_ROOT_URL, chrome_release_version, osy);
    logging.info('Downloading driver {} ...'.format(driver_url))

    r = requests.get(driver_url)
    z = zipfile.ZipFile(io.BytesIO(r.content))
    path = os.path.dirname(DEFAULT_CHROME_DRIVER_PATH)

    if not os.path.exists(path):
        os.makedirs(path)

    z.extractall(path)
    os.chmod(DEFAULT_CHROME_DRIVER_PATH, stat.S_IEXEC)

    return os.path.join(DEFAULT_CHROME_DRIVER_PATH, "drivers", "chromedriver",
                        osy, chrome_release_version, "chromedriver")

def get_chrome_driver():
    """ Download Chrome driver if it doesn't already exists
    """

    options = Options()
    options.add_argument('--headless')
    options.add_argument('--disable-gpu')

    if not os.path.exists(DEFAULT_CHROME_DRIVER_PATH):
        logging.info('Selenium driver not found, trying to download one')
        download_driver()

    return Chrome(executable_path=DEFAULT_CHROME_DRIVER_PATH, options=options)