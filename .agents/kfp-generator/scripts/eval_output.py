import os
import sys
import subprocess
import argparse
import json

def get_venv_python():
    """Locates the project's local virtual environment Python interpreter."""
    script_dir = os.path.dirname(os.path.abspath(__file__))
    candidate_roots = [
        os.path.abspath(os.path.join(script_dir, "..", "..", "..")),
        os.path.abspath(os.path.join(script_dir, "..", "..", "..", "..")),
        os.getcwd(),
    ]
   
    venv_paths = []
    for root in candidate_roots:
        venv_paths.extend([
            os.path.join(root, ".venv", "bin", "python"),
            os.path.join(root, ".venv", "Scripts", "python.exe"),
            os.path.join(root, "venv", "bin", "python"),
        ])
    for path in venv_paths:
        if os.path.exists(path):
            return path

    return sys.executable

def validate_pipeline(pipeline_path):
    """
    Attempts to compile the generated pipeline statically by executing it.
    Returns exit code, stdout, and stderr.
    """
    if not os.path.exists(pipeline_path):
        return 1, "", f"Error: Target file {pipeline_path} does not exist. Action: Verify the file path and generate the Python script before running evaluation."

    python_exe = get_venv_python()
    print(f"[*] Compiling pipeline statically using: {python_exe}")
   
    
    result = subprocess.run(
        [python_exe, pipeline_path],
        capture_output=True,
        text=True,
        timeout=300,
    )
   
    return result.returncode, result.stdout, result.stderr

def run_backend_verification(pipeline_path, host, namespace=None, timeout=600, pipeline_args=None):
    """
    Executes the compiled pipeline YAML package on a real KFP backend using create_run_from_pipeline_package, waits for completion, and captures evidence using get_run."""
    python_exe = get_venv_python()
    expected_yaml = pipeline_path + ".yaml"
    
    if not os.path.exists(expected_yaml):
        return 1, "", f"Error: Compiled YAML package {expected_yaml} not found. Run validation first."
    print(f"\n[*] Submitting compiled YAML package {expected_yaml} to live backend ({host}) for E2E verification...")
    
    runner_script = """ 
import sys
import os
import json
import kfp

yaml_path = sys.argv[1]
host = sys.argv[2]
namespace = sys.argv[3] if (len(sys.argv) > 3 and sys.argv[3] != "None") else None
timeout = int(sys.argv[4]) if len(sys.argv) > 4 else 600
raw_args = sys.argv[5] if len(sys.argv) > 5 and sys.argv[5] != "None" else "{}"

try:
  pipeline_arguments =  json.loads(raw_args)
except Exception as e:
    print(f"[!] Invalid JSON for pipeline arguments: {e}", file=sys.stderr)
    sys.exit(1)
    
# Instantiate the KFP Client
client = kfp.Client(host=host, namespace=namespace)
try:
    if client.get_kfp_healthz().multi_user:
        if namespace:
           client.set_user_namespace(namespace)
           
        for api_attr in ('_experiment_api', '_pipelines_api', '_run_api', '_upload_api'):
           api_obj = getattr(client, api_attr, None)
           if api_obj and hasattr(api_obj, 'api_client'):
              existing_headers = getattr(api_obj.api_client, 'default_headers', {})
              if 'kubeflow-userid' not in existing_headers:
                  env_user = os.getenv("KFP_USERID", os.getenv("USER", "default-user"))
                  api_obj.api_client.set_default_header('kubeflow-userid', env_user)

except Exception as e:
    pass
    
run_name = f"eval_{os.path.splitext(os.path.basename(yaml_path))[0]}"

print(f"[*] Calling create_run_from_pipeline_package for package '{yaml_path}...'")
try:
   run_result = client.create_run_from_pipeline_package(
       pipeline_file=yaml_path,
       arguments=pipeline_arguments,
       run_name=run_name,
   )

except Exception as e:
    print(f"[!] Failed during create_run_from_pipeline_package call: {e}", file=sys.stderr)
    sys.exit(1)
    
run_id = run_result.run_id
print(f"[✓] Run created successfully with run_id: {run_id}")

print(f"[*] Waiting for run execution on backend (timeout={timeout}s)...")
try:
    client.wait_for_run_completion(run_id=run_id, timeout=timeout)
except Exception as e:
    print(f"[!] Timeout or error while waiting for run completion: {e}", file=sys.stderr)


print(f"[*] Fetching final execution evidence using get_run(run_id='{run_id}')...")
try:
   run_info = client.get_run(run_id=run_id)
   run_data = {
       "run_id": run_info.run_id,
       "display_name": run_info.display_name,
       "state": str(run_info.state),
       "created_at": str(run_info.created_at),
       "finished_at": str(run_info.finished_at)
   }
   
   output_file = yaml_path + ".run.json"
   with open(output_file, "w") as f:
        json.dump(run_data, f, indent=2)
   print(f"[✓] Execution evidence saved to: {output_file}")
    
except Exception as e:
    print(f"[!] Failed to retrieve run info using get_run: {e}", file=sys.stderr)
    sys.exit(1)
    
if "SUCCEEDED" in run_data["state"] or "COMPLETED" in run_data["state"]:
    print(f"[✓] BACKEND VERIFICATION PASSED: Pipeline run {run_id} executed and SUCCEEDED on backend.")
    sys.exit(0)
    
else:
    print(f"[!] BACKEND VERIFICATION FAILED: Pipeline run {run_id} finished with non-successful state: {run_data['state']}", file=sys.stderr)
    sys.exit(1)
    
"""

    args_str = json.dumps(pipeline_args) if pipeline_args else "None"
    result = subprocess.run(
       [python_exe, 
       "-c", 
       runner_script, 
       expected_yaml, 
       host, str(namespace) 
       if namespace else "None", 
       str(timeout), args_str,], 
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
    parser.add_argument("--output-dir", type=str, default=None, help="Directory to save telemetry (timing.json, grading.json). Defaults to pipeline's directory.")
    
    parser.add_argument(
        "--run", "--run-e2e",
        action="store_true",
        dest="run_e2e",
        help="Execute the pipeline on a real backend using create_run_from_pipeline_package and verify success using get_run."
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
    
    expected_yaml = args.pipeline_file + ".yaml"
    if os.path.exists(expected_yaml):
        try:
            os.remove(expected_yaml)
            print(f"[*] Removed stale YAML artifacts: {expected_yaml}")
        except Exception as e:
            sys.stderr.write(f"[!] Warning: Could not remove stale YAML file: {e}\"\n")

    return_code, stdout, stderr = validate_pipeline(args.pipeline_file)
    is_success = (return_code == 0)

    backend_passed = False
    backend_exit_code = 0
    backend_stdout = ""
    backend_stderr = ""
    run_evidence = "Live backend execution was not requested or skipped."
    
    if return_code != 0:
        sys.stderr.write(f"\n[!] STATIC COMPILATION FAILED (Exit Code {return_code})\n")
        sys.stderr.write("--- KFP Compiler Traceback Error ---\n")
        sys.stderr.write(stderr)
        sys.stderr.write("\n------------------------------------\n")
        sys.exit(1)

    print("\n[✓] STATIC VALIDATION PASSED: The pipeline compiles into a valid KFP v2 YAML workflow specification.")
    expected_yaml = args.pipeline_file + ".yaml"
    if not os.path.exists(expected_yaml):
        sys.stderr.write(
            f"[!] VALIDATION FAILED: Expected output YAML not found at: {expected_yaml}\n"
            f"Action: Ensure the pipeline file includes an entrypoint block that compiles "
            f"the pipeline using the KFP compiler:\n\n"
            f"   if __name__ == '__main__':\n"
            f"       compiler.Compiler().compile(pipeline_file={args.pipeline_file.split('.')[0]}, __file__ + '.yaml')\n\n"
            )
        sys.exit(1)
    print(f"[✓] Output specification saved successfully at: {expected_yaml}")

    if is_success and args.run_e2e:
        pipeline_args = json.loads(args.arguments) if args.arguments else None
        backend_exit_code, backend_stdout, backend_stderr = run_backend_verification(
            args.pipeline_file,
            host=args.host,
            namespace=args.namespace,
            timeout=args.timeout,
            pipeline_args=pipeline_args,
        )
        if backend_stdout:
            print(backend_stdout)
        backend_passed = (backend_exit_code == 0)
        if backend_exit_code != 0:
            sys.stderr.write(f"\n[!] BACKEND EXECUTION VERIFICATION FAILED (Exit Code {backend_exit_code})\n")
            if backend_stderr:
                sys.stderr.write(backend_stderr)
            
        run_json_path = expected_yaml + ".run.json"
        if os.path.exists(run_json_path):
            try:
                with open(run_json_path, "r") as rf:
                    run_info = json.load(rf)
                    state = run_info.get("state", "")
                    if "SUCCEEDED" in state or "COMPLETED" in state:
                        run_evidence = f"Run {run_info.get('run_id')} finished with state: {state}"
                    else:
                        run_evidence = f"Run finished with non-successful state: {state}"
            except Exception as e:
                run_evidence = f"Failed to parse run.json evidence: {e}"
        else:
            run_evidence = backend_stderr.strip() or "Backend connection refused or execution failed before completion."
            
            
    yaml_exists = os.path.exists(expected_yaml)
    assertions = [
        {
            "text": "Pipeline file exists on disk",
            "passed": os.path.exists(pipeline_path),
            "evidence": f"Found pipeline file at {pipeline_path}" if os.path.exists(pipeline_path) else "File missing"
        },
        {
            "text": "Pipeline compiles successfully into a valid KFP v2 YAML workflow",
            "passed": is_success,
            "evidence": stdout.strip() if is_success else stderr.strip()
        },
        {
            "text": "Compiled YAML artifact generated successfully",
            "passed": yaml_exists,
            "evidence": f"Found artifact {expected_yaml} ({os.path.getsize(expected_yaml)} bytes)" if yaml_exists else "YAML artifact missing"
        }
    ]

    if args.run_e2e:
        assertions.append({
            "text": "Live backend cluster execution completed successfully",
            "passed": backend_passed,
            "evidence": run_evidence
        })

    passed_count = sum(1 for a in assertions if a["passed"])
    failed_count = len(assertions) - passed_count
    pass_rate = round(passed_count / len(assertions), 2) if assertions else 0.0

    grading_data = {
        "assertion_results": assertions,
        "summary": {
            "passed": passed_count,
            "failed": failed_count,
            "total": len(assertions),
            "pass_rate": pass_rate
        }
    }

    grading_path = os.path.join(output_dir, "grading.json")
    with open(grading_path, "w") as f:
        json.dump(grading_data, f, indent=2)
    print(f"[✓] Saved grading scorecard to {grading_path}")
    
    if not is_success or (args.run_e2e and not backend_passed):
        sys.exit(1)
    sys.exit(0)
    
if __name__ == "__main__":
    main()

