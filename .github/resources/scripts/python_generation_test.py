"""Regression coverage for SDK-owned protobuf generation and build guards."""

import importlib.util
from pathlib import Path
import sys
import tempfile
from types import ModuleType
import unittest
from unittest import mock
import zipfile

ROOT = Path(__file__).resolve().parents[3]


def load_script(relative_path: str) -> ModuleType:
    """Load a repository script without importing SDK runtime modules."""
    spec = importlib.util.spec_from_file_location('sdk_build_script',
                                                  ROOT / relative_path)
    if spec is None or spec.loader is None:
        raise ImportError(f'Cannot load script: {relative_path}')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class PythonGenerationTest(unittest.TestCase):

    def test_generators_own_sdk_outputs_and_always_regenerate(self) -> None:
        """Existing outputs must not hide changes to imported proto schemas."""
        for relative_path, package, name in (
            ('api/v2alpha1/python/generate_proto.py', 'pipeline_spec',
             'pipeline_spec'),
            ('kubernetes_platform/python/generate_proto.py', 'kubernetes',
             'kubernetes_executor_config'),
        ):
            with self.subTest(package=package):
                generator = load_script(relative_path)
                self.assertEqual(
                    Path(generator.PKG_DIR), ROOT / 'sdk/python/kfp' / package)
                with tempfile.TemporaryDirectory() as directory:
                    root = Path(directory)
                    source = root / f'{name}.proto'
                    source.write_text('syntax = "proto3";\n')
                    output = root / 'generated'
                    output.mkdir()
                    generated = output / f'{name}_pb2.py'
                    generated.write_text(
                        'import pipeline_spec_pb2 as pipeline__spec__pb2\n')
                    with mock.patch.object(generator, 'PKG_DIR', str(output)), \
                            mock.patch.object(generator, 'PROTOC', 'protoc'), \
                            mock.patch.object(generator.subprocess, 'run') as run:
                        generator.generate_proto(str(source))
                        generator.generate_proto(str(source))
                    self.assertEqual(run.call_count, 2)
                    command = run.call_args.args[0]
                    self.assertIn(f'--python_out={output}', command)
                    self.assertEqual(command[-1], str(source))
                    self.assertEqual(run.call_args.kwargs, {'check': True})
                    if package == 'kubernetes':
                        self.assertIn('from kfp.pipeline_spec import',
                                      generated.read_text())

    def test_missing_sources_and_compilers_fail_explicitly(self) -> None:
        """Never produce success-shaped output when generation cannot run."""
        for relative_path in (
                'api/v2alpha1/python/generate_proto.py',
                'kubernetes_platform/python/generate_proto.py',
        ):
            generator = load_script(relative_path)
            with tempfile.TemporaryDirectory() as directory:
                source = Path(directory) / 'source.proto'
                with self.assertRaises(FileNotFoundError):
                    generator.generate_proto(str(source))
                source.touch()
                with mock.patch.object(generator, 'PROTOC', None):
                    with self.assertRaisesRegex(RuntimeError, 'protoc'):
                        generator.generate_proto(str(source))


class PythonBuildGuardTest(unittest.TestCase):

    def test_distributions_require_each_binding_but_editable_bootstrap_does_not(
            self) -> None:
        """Test the hook independently of Hatch's isolated build
        environment."""
        with mock.patch.dict(
                sys.modules, {
                    'hatchling.builders.hooks.plugin.interface':
                        mock.Mock(BuildHookInterface=object)
                }):
            hook_module = load_script('sdk/python/hatch_build.py')
        required = (
            'kfp/pipeline_spec/pipeline_spec_pb2.py',
            'kfp/kubernetes/kubernetes_executor_config_pb2.py',
            'kfp/server_api/__init__.py',
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            hook = hook_module.CustomBuildHook()
            hook.root = directory
            hook.target_name = 'wheel'
            hook.initialize('editable', {})
            for relative_path in required:
                path = root / relative_path
                path.parent.mkdir(parents=True, exist_ok=True)
                path.touch()
            for target in ('wheel', 'sdist'):
                hook.target_name = target
                for relative_path in required:
                    with self.subTest(target=target, missing=relative_path):
                        path = root / relative_path
                        path.unlink()
                        with self.assertRaisesRegex(
                                RuntimeError, 'Missing generated SDK binding'):
                            hook.initialize('standard', {'force_include': {}})
                        path.touch()
                build_data = {'force_include': {}}
                hook.initialize('standard', build_data)
                self.assertEqual(
                    set(build_data['force_include'].values()), set(required))


class PythonDistributionTest(unittest.TestCase):

    def test_artifact_inventory_and_single_distribution_ownership(self) -> None:
        """Reject missing bindings, retired dependencies, and test payloads."""
        checker = load_script(
            '.github/resources/scripts/check_python_distribution.py')
        cases = (
            ({}, None, None),
            ({}, 'kfp/pipeline_spec/pipeline_spec_pb2.py', 'missing modules'),
            ({
                'kfp-1.dist-info/METADATA':
                    'Name: kfp\nRequires-Dist: kfp-server-api>=2\n'
            }, None, 'retired dependency'),
            ({
                'kfp/testing_tests.py': ''
            }, None, 'test payload'),
            ({
                'kfp/test_module.py': ''
            }, None, 'test payload'),
            ({
                'kfp/test_data/input.json': '{}'
            }, None, 'test payload'),
            ({
                'old-1.dist-info/METADATA': 'Name: old\n'
            }, None, 'expected one distribution'),
        )
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'kfp.whl'
            for additions, missing, error in cases:
                with self.subTest(error=error, additions=additions):
                    entries = {name: '' for name in checker.REQUIRED_MODULES}
                    entries['kfp-1.dist-info/METADATA'] = 'Name: kfp\n'
                    entries.update(additions)
                    if missing:
                        del entries[missing]
                    with zipfile.ZipFile(path, 'w') as wheel:
                        for name, content in entries.items():
                            wheel.writestr(name, content)
                    if error:
                        with self.assertRaisesRegex(AssertionError, error):
                            checker.check_wheel(path)
                    else:
                        checker.check_wheel(path)


if __name__ == '__main__':
    unittest.main()
