"""Template for notebook executor code generation.

This module contains the actual Python functions that get embedded into notebook
components at runtime. Using inspect.getsource() to convert them to source code
provides better maintainability, testing, and IDE support.

Note: The template functions import their dependencies locally to avoid requiring
those dependencies when this module is imported for code generation.
"""

import inspect
import textwrap


def __kfp_write_parameters_cell(nb, params):
    """Inject parameters following Papermill semantics.

    - If a cell tagged with 'parameters' exists, insert an overriding
      'injected-parameters' cell immediately after it.
    - Otherwise, insert the 'injected-parameters' cell at the top.
    """
    import json

    import nbformat

    if not params:
        return

    # Build the injected parameters cell
    assignments = []
    for key, value in params.items():
        serialized = json.dumps(value)
        assignments.append(key + ' = json.loads(' + repr(serialized) + ')')
    source = 'import json\n' + '\n'.join(assignments) + '\n'
    cell = nbformat.v4.new_code_cell(source=source)
    cell.metadata.setdefault('tags', [])
    if 'injected-parameters' not in cell.metadata['tags']:
        cell.metadata['tags'].append('injected-parameters')

    # Locate the first 'parameters' tagged cell
    insert_idx = 0
    for idx, existing in enumerate(nb.get('cells', [])):
        if existing.get('cell_type') != 'code':
            continue
        tags = existing.get('metadata', {}).get('tags', []) or []
        if 'parameters' in tags:
            insert_idx = idx + 1
            break

    nb.cells.insert(insert_idx, cell)


def _kfp_stream_single_output(output, cell_idx):
    """Stream a single notebook output immediately during execution.

    Prints stdout/stderr and text/plain display outputs to the console
    so users see cell output as it happens (no need to wait until the
    notebook finishes).
    """
    import sys
    output_type = output.get('output_type')

    if output_type == 'stream':
        text = output.get('text', '')
        if text:
            try:
                print(f'[nb cell {cell_idx} stream] ', end='', flush=False)
            except Exception:
                pass
            print(text, end='' if text.endswith('\n') else '\n', flush=True)
    elif output_type == 'error':
        for line in output.get('traceback', []):
            print(line, file=sys.stderr, flush=True)
    else:
        # Handle display_data and execute_result
        data = output.get('data', {})
        if 'text/plain' in data:
            print(data['text/plain'], flush=True)
        elif 'application/json' in data:
            try:
                import json as __kfp_json
                parsed = data['application/json']
                # Some kernels send JSON as string; try to parse if needed
                if isinstance(parsed, str):
                    try:
                        parsed = __kfp_json.loads(parsed)
                    except Exception:
                        pass
                print(
                    __kfp_json.dumps(parsed, indent=2, ensure_ascii=False),
                    flush=True)
            except Exception:
                # Fallback to raw
                print(str(data.get('application/json')), flush=True)
        elif 'text/markdown' in data:
            # Print markdown as-is; frontends may render, logs will show raw markdown
            print(data['text/markdown'], flush=True)


def kfp_run_notebook(**kwargs):
    """Execute the embedded notebook with injected parameters.

    Parameters provided via kwargs are injected into the notebook
    following Papermill semantics (after a parameters cell if present,
    otherwise at top). Execution uses a Python kernel; nbclient and
    ipykernel must be available at runtime (installed via
    packages_to_install for notebook components).
    """
    import os
    import subprocess
    import sys

    from nbclient import NotebookClient
    import nbformat

    # Ensure a usable 'python3' kernel is present; install kernelspec if missing
    print('[KFP Notebook] Checking for Python kernel...', flush=True)
    try:
        from jupyter_client.kernelspec import KernelSpecManager  # type: ignore
        ksm = KernelSpecManager()
        have_py3 = 'python3' in ksm.find_kernel_specs()
        if not have_py3:
            print(
                '[KFP Notebook] Python3 kernel not found, installing...',
                flush=True)
            try:
                subprocess.run([
                    sys.executable, '-m', 'ipykernel', 'install', '--user',
                    '--name', 'python3', '--display-name', 'Python 3'
                ],
                               check=True,
                               stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL)
                print(
                    '[KFP Notebook] Python3 kernel installed successfully',
                    flush=True)
            except subprocess.CalledProcessError as e:
                raise RuntimeError(
                    "Failed to install 'python3' kernelspec for ipykernel. "
                    'Ensure ipykernel is available in the environment or include it via packages_to_install. '
                    f'Error: {e}') from e
        else:
            print('[KFP Notebook] Python3 kernel found', flush=True)
    except ImportError as e:
        raise RuntimeError(
            "jupyter_client is not available. Ensure it's installed in the environment or include it via packages_to_install. "
            f'Error: {e}') from e

    nb_path = os.path.join(__KFP_EMBEDDED_ASSET_DIR, __KFP_NOTEBOOK_REL_PATH)

    try:
        nb = nbformat.read(nb_path, as_version=4)
    except Exception as e:
        raise RuntimeError(
            f'Failed to read notebook {nb_path}. Ensure it is a valid Jupyter notebook. Error: {e}'
        ) from e

    try:
        __kfp_write_parameters_cell(nb, kwargs)
        print(
            f'[KFP Notebook] Executing notebook with {len(nb.get("cells", []))} cells',
            flush=True)

        # Use our custom streaming client for real-time output (defined in the
        # generated ephemeral source)
        client = KFPStreamingNotebookClient(
            nb,
            timeout=None,
            allow_errors=False,
            store_widget_state=False,
            kernel_name='python3')
        client.execute(cwd=__KFP_EMBEDDED_ASSET_DIR)

        print('[KFP Notebook] Execution complete', flush=True)

    except Exception as e:
        raise RuntimeError(f'Notebook execution failed. Error: {e}') from e


def get_notebook_executor_source(archive_b64_placeholder: str,
                                 notebook_relpath_placeholder: str) -> str:
    """Generate the notebook execution helper source code.

    Uses inspect.getsource() to extract the actual function definitions,
    providing better maintainability, and includes archive extraction at import.

    Args:
        archive_b64_placeholder: The base64-encoded gzip-compressed tar archive containing the notebook and assets
        notebook_relpath_placeholder: The relative path to the notebook within the extracted archive root

    Returns:
        Python source code for notebook execution helpers
    """
    # Get source code for helper functions that don't require nbclient
    functions = [
        __kfp_write_parameters_cell, _kfp_stream_single_output, kfp_run_notebook
    ]

    # Extract and dedent source code for all helper functions
    function_sources = [
        textwrap.dedent(inspect.getsource(func)) for func in functions
    ]

    # Define the streaming client class inline in the generated source to avoid
    # importing nbclient in the SDK environment.
    streaming_client_source = textwrap.dedent("""
        class KFPStreamingNotebookClient(NotebookClient):
            # Streams outputs in real-time by emitting outputs during message processing.
            def process_message(self, msg, cell, cell_index):
                # Call the parent implementation to handle the message normally
                output = super().process_message(msg, cell, cell_index)

                # If an output was created, stream it immediately
                if output is not None:
                    _kfp_stream_single_output(output, cell_index)

                return output
        """)

    # Combine everything into the final source with archive extraction at import
    functions_code = streaming_client_source + '\n' + '\n'.join(
        function_sources)
    return f"""__KFP_EMBEDDED_ARCHIVE_B64 = '{archive_b64_placeholder}'
__KFP_NOTEBOOK_REL_PATH = '{notebook_relpath_placeholder}'

import base64 as __kfp_b64
import gzip as __kfp_gzip
import io as __kfp_io
import os as __kfp_os
import sys as __kfp_sys
import tarfile as __kfp_tarfile
import tempfile as __kfp_tempfile
from nbclient import NotebookClient

# Extract embedded archive at import time to ensure sys.path and globals are set
print('[KFP] Extracting embedded notebook archive...', flush=True)
__kfp_tmpdir = __kfp_tempfile.TemporaryDirectory()
__KFP_EMBEDDED_ASSET_DIR = __kfp_tmpdir.name
try:
    __kfp_bytes = __kfp_b64.b64decode(__KFP_EMBEDDED_ARCHIVE_B64.encode('ascii'))
    with __kfp_tarfile.open(fileobj=__kfp_io.BytesIO(__kfp_bytes), mode='r:gz') as __kfp_tar:
        __kfp_tar.extractall(path=__KFP_EMBEDDED_ASSET_DIR)
    print(f'[KFP] Notebook archive extracted to: {{__KFP_EMBEDDED_ASSET_DIR}}', flush=True)
except Exception as __kfp_e:
    raise RuntimeError(f'Failed to extract embedded notebook archive: {{__kfp_e}}')

# Always prepend the extracted directory to sys.path for import resolution
if __KFP_EMBEDDED_ASSET_DIR not in __kfp_sys.path:
    __kfp_sys.path.insert(0, __KFP_EMBEDDED_ASSET_DIR)
    print(f'[KFP] Added notebook archive directory to Python path', flush=True)

# Optional convenience for generic embedded file variable name
__KFP_EMBEDDED_ASSET_FILE = __kfp_os.path.join(__KFP_EMBEDDED_ASSET_DIR, __KFP_NOTEBOOK_REL_PATH)

{functions_code}

# Bind helper into dsl namespace so user code can call dsl.run_notebook(...)
dsl.run_notebook = kfp_run_notebook"""
