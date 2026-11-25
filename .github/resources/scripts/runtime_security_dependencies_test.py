import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


class RuntimeSecurityDependenciesTest(unittest.TestCase):
    def test_runtime_zlib_is_refreshed_before_unprivileged_execution(self):
        for name in (
            'launcher',
            'driver',
            'persistenceagent',
            'scheduledworkflow',
            'viewercontroller',
            'conformance',
        ):
            with self.subTest(image=name):
                content = (ROOT / f'backend/Dockerfile.{name}').read_text()
                runtime = re.split(r'^FROM ', content, flags=re.MULTILINE)[-1]
                self.assertIn("apk add --no-cache 'zlib>=1.4-r0'", runtime)
                self.assertLess(
                    runtime.index('apk add'),
                    runtime.index('USER ')
                    if 'USER ' in runtime
                    else len(runtime),
                )
        api = (
            (ROOT / 'backend/Dockerfile')
            .read_text()
            .split('FROM debian:stable-slim')[1]
        )
        self.assertIn('ca-certificates wget zlib1g', api)
        frontend = (
            (ROOT / 'frontend/Dockerfile')
            .read_text()
            .split('FROM node:${NODE_VERSION}-${BASE_IMAGE}')[1]
        )
        self.assertIn("apk add --no-cache 'zlib>=1.4-r0'", frontend)
        self.assertIn(
            'apt-get install -y --no-install-recommends zlib1g', frontend
        )

        for runtime in (api, frontend):
            self.assertIn(
                """dpkg --compare-versions "$(dpkg-query -W -f='${Version}' zlib1g)" ge '1:1.3.dfsg+really1.3.1-1+deb13u1'""",
                runtime,
            )

    def test_api_and_frontend_runtime_security_floors(self):
        api = (ROOT / 'backend/Dockerfile').read_text()
        self.assertRegex(
            api, r'(?m)^FROM debian:stable-slim@sha256:[0-9a-f]{64}$'
        )
        for package in ('libc6', 'libc-bin'):
            self.assertIn(
                f"""dpkg --compare-versions "$(dpkg-query -W -f='${{Version}}' {package})" ge '2.41-12+deb13u5'""",
                api,
            )

        frontend = (ROOT / 'frontend/Dockerfile').read_text()
        version = (
            (ROOT / 'frontend/.nvmrc').read_text().strip().removeprefix('v')
        )
        self.assertRegex(
            frontend,
            rf'(?m)^FROM node:{re.escape(version)}-alpine@sha256:[0-9a-f]{{64}} AS pinned-runtime$',
        )
        self.assertIn(
            f'ARG CUSTOM_TAG=${{RUNTIME_TAG#{version}-alpine}}', frontend
        )
        for package in ('libcrypto3', 'libssl3'):
            self.assertIn(f"'{package}>=3.5.9-r1'", frontend)

    def test_cert_manager_install_paths_use_qualified_release(self):
        makefile = (ROOT / 'backend/Makefile').read_text()
        self.assertIn('CERT_MANAGER_VERSION ?= v1.21.3', makefile)
        migration = (
            ROOT
            / '.github/workflows/kfp-kubernetes-native-migration-tests.yaml'
        ).read_text()
        self.assertIn('/releases/download/v1.21.3/cert-manager.yaml', migration)
        self.assertNotIn('/releases/download/v1.13.3/', migration)


if __name__ == '__main__':
    unittest.main()
