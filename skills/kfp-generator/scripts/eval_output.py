import os
import sys
import subprocess
import argparse
import json

def get_venv_python():
    """Locates the project's local virtual environment Python interpreter."""
    # Since this script runs from within the monorepo, resolve paths relative to the root
    script_dir = os.path.dirname(os.path.abspath(__file__))
    repo_root = os.path.abspath(os.path.join(script_dir, "..", "..", "..", ".."))
   
    venv_paths = [
        os.path.join(repo_root, ".venv", "bin", "python"),
        os.path.join(repo_root, ".venv", "Scripts", "python.exe"),
        os.path.join(repo_root, "venv", "bin", "python"),
    ]
    for path in venv_paths:
        if os.path.exists(path):
            return path
    # Fallback to the globally executing python interpreter
    return sys.executable

def validate_pipeline(pipeline_path):
    """
    Attempts to compile the generated pipeline statically by executing it.
    Returns exit code, stdout, and stderr.
    """
    if not os.path.exists(pipeline_path):
        return 1, "", f"Error: Target file {pipeline_path} does not exist."

    python_exe = get_venv_python()
    print(f"[*] Compiling pipeline statically using: {python_exe}")
   
    # Run the compiled file to trigger KFP v2 YAML generation
    result = subprocess.run(
        [python_exe, pipeline_path],
        capture_output=True,
        text=True,
        timeout=300,
    )
   
    return result.returncode, result.stdout, result.stderr

def run_backend_verification(pipeline_path, host, namespace=None, timeout=600, pipeline_args=None):
    """
    Executes the pipeline function on a real KFP backend using create_run_from_pipeline_func,
    waits for completion, and captures evidence of success using get_run.
    """
    python_exe = get_venv_python()
    print(f"\n[*] Submitting pipeline to live backend ({host}) for E2E verification...")

    runner_script = """
import sys
import os
import json
import importlib.util
import kfp

pipeline_path = sys.argv[1]
host = sys.argv[2]
namespace = sys.argv[3] if len(sys.argv) > 3 and sys.argv[3] != "None" else None
timeout = int(sys.argv[4]) if len(sys.argv) > 4 else 600
raw_args = sys.argv[5] if len(sys.argv) > 5 and sys.argv[5] != "None" else "{}"

try:
    pipeline_arguments = json.loads(raw_args)
except Exception as e:
    print(f"[!] Invalid JSON for pipeline arguments: {e}", file=sys.stderr)
    sys.exit(1)

# Dynamically load the pipeline script
abs_path = os.path.abspath(pipeline_path)
spec = importlib.util.spec_from_file_location("eval_target_pipeline", abs_path)
if not spec or not spec.loader:
    print(f"[!] Could not load spec for pipeline file at {abs_path}", file=sys.stderr)
    sys.exit(1)

module = importlib.util.module_from_spec(spec)
sys.modules["eval_target_pipeline"] = module
spec.loader.exec_module(module)

# Search for the pipeline function
pipeline_func = None
for attr_name in dir(module):
    attr = getattr(module, attr_name)
    if type(attr).__name__ in ("Pipeline", "GraphComponent") or getattr(attr, "is_pipeline", False):
        pipeline_func = attr
        break

if not pipeline_func:
    for attr_name in dir(module):
        attr = getattr(module, attr_name)
        if hasattr(attr, "pipeline_spec"):
            pipeline_func = attr
            break


if not pipeline_func:
    from kfp.dsl import base_component
    for attr_name in dir(module):
        attr = getattr(module, attr_name)
        if isinstance(attr, base_component.BaseComponent):
            pipeline_func = attr
            break

if not pipeline_func:
    print(f"[!] Error: Could not locate a @dsl.pipeline decorated function in {pipeline_path}", file=sys.stderr)
    sys.exit(1)

# Instantiate client and submit run using create_run_from_pipeline_func
client = kfp.Client(host=host, namespace=namespace)
try:
    if client.get_kfp_healthz().multi_user:
        if not namespace:
            namespace = 'user'
            client.set_user_namespace('user')
        for api_attr in ('_experiment_api', '_pipelines_api', '_run_api', '_upload_api'):
            api_obj = getattr(client, api_attr, None)
            if api_obj and hasattr(api_obj, 'api_client'):
                api_obj.api_client.set_default_header('kubeflow-userid', 'user@example.com')
except Exception as e:
    pass

run_name = f"eval_{os.path.splitext(os.path.basename(pipeline_path))[0]}"
pipe_name = getattr(pipeline_func, "name", None) or getattr(pipeline_func, "__name__", "pipeline")

print(f"[*] Calling create_run_from_pipeline_func for pipeline '{pipe_name}'...")
try:
    run_result = client.create_run_from_pipeline_func(
        pipeline_func=pipeline_func,
        arguments=pipeline_arguments,
        run_name=run_name
    )
except Exception as e:
    print(f"[!] Failed during create_run_from_pipeline_func call: {e}", file=sys.stderr)
    sys.exit(1)

run_id = run_result.run_id
print(f"[✓] Run created successfully via create_run_from_pipeline_func. Run ID: {run_id}")

print(f"[*] Waiting for run execution on backend (timeout={timeout}s)...")
try:
    client.wait_for_run_completion(run_id=run_id, timeout=timeout)
except Exception as e:
    print(f"[!] Timeout or error while waiting for run completion: {e}", file=sys.stderr)

print(f"[*] Fetching final execution evidence using get_run(run_id='{run_id}')...")
try:
    run_info = client.get_run(run_id=run_id)
except Exception as e:
    print(f"[!] Failed to retrieve run info using get_run: {e}", file=sys.stderr)
    sys.exit(1)

state = getattr(run_info, "state", None)
if state is None and hasattr(run_info, "run"):
    state = getattr(run_info.run, "state", None)

state_str = str(state).upper() if state else "UNKNOWN"
print(f"[*] Run State captured via get_run: {state_str}")

if "SUCCEEDED" in state_str or "COMPLETED" in state_str:
    print(f"[✓] BACKEND VERIFICATION PASSED: Pipeline run {run_id} executed and SUCCEEDED on backend.")
    sys.exit(0)
else:
    print(f"[!] BACKEND VERIFICATION FAILED: Pipeline run {run_id} finished with non-successful state: {state_str}", file=sys.stderr)
    sys.exit(1)
"""

    args_str = json.dumps(pipeline_args) if pipeline_args else "None"
    result = subprocess.run(
        [
            python_exe,
            "-c",
            runner_script,
            pipeline_path,
            host,
            str(namespace) if namespace else "None",
            str(timeout),
            args_str,
        ],
        capture_output=True,
        text=True,
        timeout=timeout + 30,
    )

    return result.returncode, result.stdout, result.stderr

def main():
    parser = argparse.ArgumentParser(
        description="KFP pipeline evaluation tool: static YAML compilation and live backend execution verification."
    )
    parser.add_argument(
        "pipeline_file",
        type=str,
        help="Path to a Python pipeline file to evaluate."
    )
    parser.add_argument(
        "--run", "--run-e2e",
        action="store_true",
        dest="run_e2e",
        help="Execute the pipeline on a real backend using create_run_from_pipeline_func and verify success using get_run."
    )
    parser.add_argument(
        "--host",
        type=str,
        default=os.getenv("KFP_ENDPOINT", "http://localhost:8080"),
        help="KFP API server endpoint host URL (default: http://localhost:8080 or KFP_ENDPOINT env var)."
    )
    parser.add_argument(
        "--namespace",
        type=str,
        default=None,
        help="Kubernetes namespace (for multi-user deployments)."
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=600,
        help="Timeout in seconds to wait for backend run completion (default: 600)."
    )
    parser.add_argument(
        "--arguments",
        type=str,
        default=None,
        help="JSON string of arguments to pass to the pipeline function."
    )

    args = parser.parse_args()

    # Step 1: Run static compilation validation
    return_code, stdout, stderr = validate_pipeline(args.pipeline_file)
   
    if return_code != 0:
        sys.stderr.write(f"\n[!] STATIC COMPILATION FAILED (Exit Code {return_code})\n")
        sys.stderr.write("--- KFP Compiler Traceback Error ---\n")
        sys.stderr.write(stderr)
        sys.stderr.write("\n------------------------------------\n")
        sys.exit(1)

    print("\n[✓] STATIC VALIDATION PASSED: The pipeline compiles into a valid KFP v2 YAML workflow specification.")
    expected_yaml = args.pipeline_file + ".yaml"
    if not os.path.exists(expected_yaml):
        sys.stderr.write(f"[!] VALIDATION FAILED: Expected output YAML not found at: {expected_yaml}\n")
        sys.exit(1)
    print(f"[✓] Output specification saved successfully at: {expected_yaml}")

    # Step 2: Live Backend Execution Verification (if --run / --run-e2e flag is set)
    if args.run_e2e:
        pipeline_args = json.loads(args.arguments) if args.arguments else None
        b_code, b_stdout, b_stderr = run_backend_verification(
            args.pipeline_file,
            host=args.host,
            namespace=args.namespace,
            timeout=args.timeout,
            pipeline_args=pipeline_args,
        )
        if b_stdout:
            print(b_stdout)
        if b_code != 0:
            sys.stderr.write(f"\n[!] BACKEND EXECUTION VERIFICATION FAILED (Exit Code {b_code})\n")
            if b_stderr:
                sys.stderr.write(b_stderr)
            sys.exit(1)

    sys.exit(0)

if __name__ == "__main__":
    main()

