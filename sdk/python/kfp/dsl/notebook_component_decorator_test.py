"""Unit tests for notebook components and embedded artifacts."""

import base64
import io
import json
import os
import tarfile
import tempfile
import unittest
import warnings

from kfp import dsl
from kfp.dsl import component_factory
from kfp.dsl.types.artifact_types import Dataset


class TestNotebookComponentDecorator(unittest.TestCase):

    def _make_temp_notebook(self, cells_source: str) -> str:
        nb = {
            'cells': [{
                'cell_type': 'code',
                'execution_count': None,
                'metadata': {},
                'outputs': [],
                'source': cells_source,
            }],
            'metadata': {
                'kernelspec': {
                    'display_name': 'Python 3',
                    'language': 'python',
                    'name': 'python3',
                },
                'language_info': {
                    'name': 'python',
                    'version': '3.11',
                },
            },
            'nbformat': 4,
            'nbformat_minor': 5,
        }
        tmpdir = tempfile.mkdtemp()
        nb_path = os.path.join(tmpdir, 'tmp.ipynb')
        with open(nb_path, 'w', encoding='utf-8') as f:
            json.dump(nb, f)
        return nb_path

    def test_notebook_component_default_packages_install(self):
        nb_path = self._make_temp_notebook(
            "import os\nos.makedirs('/tmp/kfp_nb_outputs', exist_ok=True)\n")

        @dsl.notebook_component(notebook_path=nb_path)
        def my_nb(text: str):
            dsl.run_notebook(text=text)

        container = my_nb.component_spec.implementation.container
        command = ' '.join(container.command)
        self.assertIn('nbclient>=0.10,<1', command)
        self.assertIn('ipykernel>=6,<7', command)
        self.assertIn('jupyter_client>=7,<9', command)

    def test_notebook_component_no_extra_packages_when_empty_list(self):
        nb_path = self._make_temp_notebook('pass\n')

        @dsl.notebook_component(notebook_path=nb_path, packages_to_install=[])
        def my_nb():
            dsl.run_notebook()

        container = my_nb.component_spec.implementation.container
        command = ' '.join(container.command)
        self.assertNotIn('nbclient>=0.10,<1', command)
        self.assertNotIn('ipykernel>=6,<7', command)
        self.assertNotIn('jupyter_client>=7,<9', command)


class TestNotebookExecutorTemplate(unittest.TestCase):

    def test_template_binds_run_notebook(self):
        from kfp.dsl.templates.notebook_executor import get_notebook_executor_source

        source = get_notebook_executor_source('ARCHIVE_B64', 'nb.ipynb')
        self.assertIn('dsl.run_notebook = kfp_run_notebook', source)
        self.assertIn('class KFPStreamingNotebookClient(NotebookClient):',
                      source)


class TestNotebookParameterInjection(unittest.TestCase):

    def _make_nb(self, with_parameters_cell: bool) -> dict:
        import nbformat
        nb = {
            'cells': [],
            'metadata': {
                'kernelspec': {
                    'display_name': 'Python 3',
                    'language': 'python',
                    'name': 'python3',
                },
                'language_info': {
                    'name': 'python',
                    'version': '3.11',
                },
            },
            'nbformat': 4,
            'nbformat_minor': 5,
        }
        if with_parameters_cell:
            nb['cells'].append({
                'cell_type': 'code',
                'execution_count': None,
                'metadata': {
                    'tags': ['parameters'],
                },
                'outputs': [],
                'source': '# default parameters\n',
            })
        # Regular code cell
        nb['cells'].append({
            'cell_type': 'code',
            'execution_count': None,
            'metadata': {},
            'outputs': [],
            'source': 'print("hello")\n',
        })
        return nbformat.from_dict(nb)

    def test_injects_after_parameters_cell(self):
        from kfp.dsl.templates import notebook_executor as nb_exec
        nb = self._make_nb(with_parameters_cell=True)

        params = {
            'x': 1,
            's': 'abc',
            'd': {
                'a': 1
            },
        }

        getattr(nb_exec, '__kfp_write_parameters_cell')(nb, params)

        # Expect new cell inserted immediately after the parameters cell (index 1)
        self.assertGreaterEqual(len(nb.get('cells', [])), 2)
        injected = nb['cells'][1]
        self.assertEqual(injected.get('cell_type'), 'code')
        tags = injected.get('metadata', {}).get('tags', []) or []
        self.assertIn('injected-parameters', tags)
        src = ''.join(injected.get('source', ''))
        self.assertIn('import json', src)
        # Ensure each variable assignment is present
        self.assertIn('x = json.loads(', src)
        self.assertIn('s = json.loads(', src)
        self.assertIn('d = json.loads(', src)

    def test_injects_at_top_when_no_parameters_cell(self):
        from kfp.dsl.templates import notebook_executor as nb_exec
        nb = self._make_nb(with_parameters_cell=False)

        getattr(nb_exec, '__kfp_write_parameters_cell')(nb, {'p': 42})

        self.assertGreaterEqual(len(nb.get('cells', [])), 2)
        injected = nb['cells'][0]
        self.assertEqual(injected.get('cell_type'), 'code')
        tags = injected.get('metadata', {}).get('tags', []) or []
        self.assertIn('injected-parameters', tags)
        src = ''.join(injected.get('source', ''))
        self.assertIn('p = json.loads(', src)


class TestEmbeddedArtifacts(unittest.TestCase):

    def _make_tar_gz_b64_for_paths(self, root_dir: str) -> str:
        buf = io.BytesIO()
        with tarfile.open(fileobj=buf, mode='w') as tar:
            tar.add(root_dir, arcname='.')
        raw_bytes = buf.getvalue()
        gz_bytes = __import__('gzip').compress(raw_bytes)
        return base64.b64encode(gz_bytes).decode('ascii')

    def test_shared_extraction_helper_exposes_dir_and_file(self):
        tmpdir = tempfile.mkdtemp()
        file_basename = 'data.txt'
        with open(
                os.path.join(tmpdir, file_basename), 'w',
                encoding='utf-8') as f:
            f.write('hello')

        archive_b64 = self._make_tar_gz_b64_for_paths(tmpdir)
        helper_src = component_factory._generate_shared_extraction_helper(
            embedded_archive_b64=archive_b64, file_basename=file_basename)

        g: dict = {}
        exec(helper_src, g, g)
        self.assertIn('__KFP_EMBEDDED_ASSET_DIR', g)
        self.assertIn('__KFP_EMBEDDED_ASSET_FILE', g)
        self.assertTrue(os.path.isdir(g['__KFP_EMBEDDED_ASSET_DIR']))
        self.assertTrue(os.path.isfile(g['__KFP_EMBEDDED_ASSET_FILE']))
        with open(g['__KFP_EMBEDDED_ASSET_FILE'], 'r', encoding='utf-8') as f:
            self.assertEqual(f.read(), 'hello')
        # sys.path must include the extracted dir (at position 0 per helper)
        import sys
        self.assertIn(g['__KFP_EMBEDDED_ASSET_DIR'], sys.path)

    def test_embedded_input_excluded_from_interface_and_runtime_injected(self):

        def comp_sig(cfg: dsl.EmbeddedInput[Dataset], out: dsl.Output[Dataset]):
            pass

        spec = component_factory.extract_component_interface(comp_sig)
        self.assertIsNone(spec.inputs)

        tmpdir = tempfile.mkdtemp()
        payload_path = os.path.join(tmpdir, 'x.txt')
        with open(payload_path, 'w', encoding='utf-8') as f:
            f.write('PAYLOAD')

        def use_embedded(cfg: dsl.EmbeddedInput[Dataset],
                         out: dsl.Output[Dataset]):
            import os as _os
            with open(
                    _os.path.join(cfg.path, 'x.txt'), 'r',
                    encoding='utf-8') as _f:
                data = _f.read()
            with open(out.path, 'w', encoding='utf-8') as _g:
                _g.write(data)

        # Build a minimal executor input for a single output artifact
        executor_input = {
            'inputs': {},
            'outputs': {
                'artifacts': {
                    'out': {
                        'artifacts': [{
                            'name': 'out',
                            'type': {
                                'schemaTitle': 'system.Dataset',
                                'schemaVersion': '0.0.1',
                            },
                            'uri': os.path.join(tempfile.mkdtemp(), 'out'),
                            'metadata': {},
                        }]
                    }
                },
                'outputFile':
                    os.path.join(tempfile.mkdtemp(), 'executor_output.json'),
            },
        }

        from kfp.dsl import executor as _executor

        # Inject globals emulating extraction helper behavior
        use_embedded.__globals__['__KFP_EMBEDDED_ASSET_DIR'] = tmpdir
        e = _executor.Executor(executor_input, function_to_execute=use_embedded)
        e.execute()

        out_path = e.get_output_artifact_path('out')
        with open(out_path, 'r', encoding='utf-8') as f:
            self.assertEqual(f.read(), 'PAYLOAD')

    def test_large_embedded_artifact_emits_warning(self):
        # Create >1MB of incompressible data so gzip compressed size stays >1MB
        tmpdir = tempfile.mkdtemp()
        bigfile = os.path.join(tmpdir, 'big.bin')
        with open(bigfile, 'wb') as f:
            f.write(os.urandom(3 * 1024 * 1024))

        @dsl.component(embedded_artifact_path=tmpdir)
        def dummy():
            pass

        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter('always')
            _ = dummy

            # Building the component happens at decoration time above; to force
            # the path through component_factory, explicitly rebuild:
            def f():
                pass

            component_factory.create_component_from_func(
                func=f, embedded_artifact_path=tmpdir)
            self.assertTrue(
                any(
                    isinstance(ww.message, UserWarning) and
                    'Embedded artifact archive is large' in str(ww.message)
                    for ww in w),
                msg='Expected a UserWarning about large embedded artifact')


if __name__ == '__main__':
    unittest.main()
