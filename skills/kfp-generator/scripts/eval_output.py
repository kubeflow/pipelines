import os
import sys
import subprocess
import argparse

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
    Attempts to compile the generated pipeline. 
    Returns exit code and captured errors.
    """
    if not os.path.exists(pipeline_path):
        return 1, "", f"Error: Target file {pipeline_path} does not exist."

    python_exe = get_venv_python()
    print(f"[*] Compiling pipeline using: {python_exe}")
    
    # Run the compiled file to trigger KFP v2 YAML generation
    result = subprocess.run(
        [python_exe, pipeline_path],
        capture_output=True,
        text=True,
        timeout=300,
    )
    
    return result.returncode, result.stdout, result.stderr

def main():
    parser = argparse.ArgumentParser(description="KFP pipeline compilation validator (executes the target .py; only run on trusted inputs).")
    parser.add_argument("pipeline_file", type=str, help="Path to a trusted Python pipeline file to execute for compilation validation")
    args = parser.parse_args()

    # Step 1: Run the compilation validation
    return_code, stdout, stderr = validate_pipeline(args.pipeline_file)
    
    if return_code == 0:
        print("\n[✓] VALIDATION PASSED: The pipeline compiles into a valid KFP v2 YAML workflow specification.")
        # Ensure a YAML file was actually produced
        expected_yaml = args.pipeline_file + ".yaml"
        if not os.path.exists(expected_yaml):
             sys.stderr.write(f"[!] VALIDATION FAILED: Expected output YAML not found at: {expected_yaml}\n")
             sys.exit(1)
         print(f"[✓] Output specification saved successfully at: {expected_yaml}")
        sys.exit(0)
    else:
        # Stream the exact traceback to stderr so the Agent (Cursor/Claude Code) catches it
        sys.stderr.write(f"\n[!] VALIDATION FAILED (Exit Code {return_code})\n")
        sys.stderr.write("--- KFP Compiler Traceback Error ---\n")
        sys.stderr.write(stderr)
        sys.stderr.write("\n------------------------------------\n")
        sys.exit(1)

if __name__ == "__main__":
    main()