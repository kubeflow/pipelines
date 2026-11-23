# Copyright 2026 The Kubeflow Authors
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

import json
from pathlib import Path
import unittest

import yaml

INSTALL = Path(__file__).resolve().parents[1] / 'third-party/istio/install'


class IstioImageVariantTest(unittest.TestCase):

    def test_injector_and_prerequisites_use_distroless(self):
        overlay = yaml.safe_load((INSTALL / 'kustomization.yaml').read_text())
        injector = json.loads((INSTALL / 'injector-values.json').read_text())
        version = injector['global']['tag']
        self.assertEqual(injector['global']['variant'], 'distroless')
        images = {image['name']: image['newTag'] for image in overlay['images']}
        self.assertEqual(
            images, {
                'docker.io/istio/' + name: version + '-distroless'
                for name in ('install-cni', 'pilot', 'proxyv2')
            })
        cni_patch = next(entry for entry in overlay['patches']
                         if entry['target']['name'] == 'istio-cni-config')
        settings = yaml.safe_load(cni_patch['patch'])
        agent_version = next(
            setting['value']
            for setting in settings
            if setting['path'] == '/data/CURRENT_AGENT_VERSION')
        self.assertEqual(agent_version, version)


if __name__ == '__main__':
    unittest.main()
