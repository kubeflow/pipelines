import os 
import sys
import subprocess

def bootstrap_workspace():
    """Bootstraps the workspace by installing and cloning the github repo."""
    if not os.path.exists("samples"):
        print("[*] Working elsewhere: 'samples/ directory not found", flush=True)
        print("[*] Bootstrapping repository files into current workspace...", flush=True)
        
        repo_url = "https://github.com/kubeflow/pipelines.git"
        try:
            subprocess.run(["git", "clone", repo_url, "."], check=True)
            print(" [+] Bootstrap complete! Repository cloned successfully.", flush=True)
        except subprocess.CalledProcessError as e:
            print(f"[-] Bootstrap failed: Git clone exited with error code {e.returncode}", file=sys.stderr)
            sys.exit(1)
            

def launch_evaluator():
    current_dir = os.path.dirname(os.path.abspath(__file__))
    evaluator_script = os.path.join(current_dir, "scripts", "eval_output.py")
    
    if not os.path.exists(evaluator_script):
        print(f"[-] Critical Error: internal evaluator script not found at {evaluator_script}", file=sys.stderr)
        sys.exit(1)
        
    print(f"[*] Launching evaluator: {evaluator_script}...", flush=True)
    
    result = subprocess.run([sys.executable, evaluator_script] + sys.argv[1:])
    sys.exit(result.returncode)
    
if __name__ == "__main__":
    bootstrap_workspace()
    launch_evaluator()