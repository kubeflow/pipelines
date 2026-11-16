#!/usr/bin/env python3
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
"""Guards against the cert-manager install race in TLS-enabled deployments.

cert-manager pods being Ready does not guarantee the cert-manager
webhook is serving. If Issuer/Certificate CRs are applied before the
webhook is up (or before the cert-manager CRDs are established), the CRs
can be rejected, the kfp-api-tls-cert Secret is never issued, and pods
mounting it fail with FailedMount. These tests pin the ordering contract
that prevents that race.
"""

import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('deploy-kfp.sh')
MAKEFILE = Path(__file__).resolve().parents[3] / 'backend' / 'Makefile'


class InstallCertManagerWaitsTest(unittest.TestCase):
    def setUp(self):
        makefile = MAKEFILE.read_text(encoding='utf-8')
        start = makefile.index('install-cert-manager:')
        end = makefile.index('\n.PHONY', start)
        # Exact-line entries: substring matches would let a `|| true` suffix
        # (the codebase's usual flaky-lane silencer) quietly turn the waits
        # into best-effort and reintroduce this race.
        self.target = [
            line.strip() for line in makefile[start:end].splitlines()
        ]

    def test_waits_for_webhook_and_cainjector_deployments(self):
        self.assertIn(
            'kubectl wait --for=condition=Available deployment -n cert-manager '
            'cert-manager-webhook cert-manager-cainjector '
            '--timeout=$(CERT_MANAGER_WAIT_TIMEOUT)',
            self.target,
        )

    def test_waits_for_cert_manager_crds_established(self):
        self.assertIn(
            'kubectl wait --for condition=established '
            'crd/certificates.cert-manager.io crd/issuers.cert-manager.io '
            '--timeout=$(CERT_MANAGER_WAIT_TIMEOUT)',
            self.target,
        )

    def test_makefile_changes_trigger_the_ci_scripts_suite(self):
        """The workflow/inventory sync test only catches drift between those
        two files.

        Removing `backend/Makefile` from both at once would stay green
        while Makefile edits stop triggering these unit tests.
        """
        workflow = (
            Path(__file__).resolve().parents[3]
            / '.github'
            / 'workflows'
            / 'ci-scripts-tests.yml'
        ).read_text(encoding='utf-8')

        self.assertIn("'backend/Makefile'", workflow)


class DeployKfpCertificateWaitTest(unittest.TestCase):
    def setUp(self):
        self.script = SCRIPT.read_text(encoding='utf-8')

    def test_tls_lane_waits_for_certificates_before_pod_readiness(self):
        kfp_apply = self.script.index('echo "Deploying ${TEST_MANIFESTS}..."')
        cert_wait = self.script.index('wait --for=condition=Ready certificate')
        pods_ready = self.script.index('wait_for_pods || EXIT_CODE=$?')

        self.assertLess(kfp_apply, cert_wait)
        self.assertLess(cert_wait, pods_ready)

    def test_manifest_apply_retries_transient_webhook_rejections(self):
        """Predicting exactly when the cert-manager webhook starts serving is
        fragile; the apply is idempotent, so retry it instead.

        This absorbs any residual rejection window without heuristics.
        """
        apply_call = self.script.index('retry 5 5 apply_manifests')
        cert_wait = self.script.index('wait --for=condition=Ready certificate')

        self.assertLess(apply_call, cert_wait)
        # The retried command must be the full kustomize apply pipeline.
        self.assertIn(
            'kubectl kustomize --load-restrictor LoadRestrictionsNone '
            '"${TEST_MANIFESTS}" | kubectl apply -f -',
            self.script,
        )

    def test_certificate_wait_covers_all_cert_backed_lanes(self):
        """The wait must cover every lane that applies Certificate resources:
        the TLS lane (base-tls-certs) and the kubernetes-native lane
        (base-webhook-certs) — the same condition that installs cert-manager.
        """
        gate_text = (
            '"${PIPELINES_STORE}" == "kubernetes" ] || '
            '[ "${POD_TO_POD_TLS_ENABLED}" == "true"'
        )
        cert_manager_install = self.script.index(
            'make -C ./backend install-cert-manager'
        )
        # The wait's gate is the second occurrence of this condition (the
        # first guards the cert-manager install itself).
        gate = self.script.index(gate_text, cert_manager_install)
        cert_wait = self.script.index('wait --for=condition=Ready certificate')
        # Containment: the wait must run inside the gated block. An ungated
        # wait breaks every lane without cert-manager CRDs installed, because
        # `kubectl wait certificate` errors on the unknown resource kind.
        gate_end = self.script.index('\nfi', gate)

        self.assertLess(cert_manager_install, gate)
        self.assertLess(gate, cert_wait)
        self.assertLess(cert_wait, gate_end)

        # Symmetry: the exact gate line must appear exactly twice — once for
        # the cert-manager install and once for this wait. If the install gate
        # grows a new cert-backed lane without the wait gate following, that
        # lane reintroduces the race these tests pin.
        gate_line = (
            'if [ "${PIPELINES_STORE}" == "kubernetes" ] || '
            '[ "${POD_TO_POD_TLS_ENABLED}" == "true" ]; then'
        )
        self.assertEqual(self.script.count(gate_line), 2)

    def test_certificate_wait_failure_is_fatal(self):
        """Pins the full wait line and the EXIT_CODE wiring.

        Under `set -e`, a bare `kubectl wait` would exit at the wait
        itself — losing the diagnostics block — while every string these
        tests assert on would still exist in the file.
        """
        cert_wait = self.script.index(
            'kubectl wait --for=condition=Ready certificate '
            '--all -n kubeflow --timeout=180s || EXIT_CODE=$?'
        )
        error_check = self.script.index('if [[ $EXIT_CODE -ne 0 ]]', cert_wait)
        failure = self.script.index(
            'TLS certificates did not become Ready', cert_wait
        )

        self.assertLess(cert_wait, error_check)
        self.assertLess(error_check, failure)

    def test_certificate_wait_failure_collects_diagnostics(self):
        """A stuck Certificate must leave state behind, mirroring the
        diagnostics contract of wait_for_pods_ready: a bare timeout line
        makes intermittent issuance failures undiagnosable in CI logs.
        """
        failure = self.script.index('TLS certificates did not become Ready')
        exit_line = self.script.index('exit 1', failure)
        diagnostics = [
            'describe certificate',
            'describe issuer',
            'logs deployment/cert-manager',
            'logs deployment/cert-manager-webhook',
            'logs deployment/cert-manager-cainjector',
        ]

        for diagnostic in diagnostics:
            position = self.script.index(diagnostic, failure)
            self.assertLess(failure, position)
            self.assertLess(position, exit_line)


if __name__ == '__main__':
    unittest.main()
