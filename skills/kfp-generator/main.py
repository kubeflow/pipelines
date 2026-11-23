import os
import subprocess
import sys


def get_skill_root() -> str:
    return os.path.dirname(os.path.abspath(__file__)) # gets the skills directory after installation.


def bootstrap_reference(ref-dir: str | None = None) -> str:
    """Bootstraps the workspace by installing the skill and look if KFP reference checkout exists."""
    target_dir = ref_dir or os.environ.get("KFP_REFERENCE_DIR", os.path.join(get_skill_dir(), ".kfp-ref"))
    repo_url = 'https://github.com/kubeflow/pipelines.git'
    
    if os.path.isdir(os.path.join(target_dir, ".git")):
        print(f"[*] Using existing KFP reference checkout at {target_dir}", flush=True)
        return target_dir
    
    if os.path.exists(target_dir) and os.listdir(target_dir):
        print(f"[-] Reference target '{target_dir}' exists and is not empty.", file=sys.stderr,)
        sys.exit(1)
        
    
    os.makedirs(os.path.dirname(target_dir), exist_ok=True)
    print(f"[*] Cloning KFP reference repository into {target_dir}...", flush=True)
        try:
            subprocess.run(
                [ "git", 'clone', "--depth", "1", repo_url, target_dir], 
                check=True,
                )
            print(
                ' [+] KFP reference checkout ready.',
                flush=True,
            )
        except subprocess.CalledProcessError as e:
            print(
                f'[-] Bootstrap failed: Git clone exited with error code {e.returncode}',
                file=sys.stderr,
            )
            sys.exit(1)

        return target_dir

def launch_evaluator(ref_dir: str) -> None:
    evaluator_script = os.path.join(get_skill_root(), 'scripts', 'eval_output.py')

    if not os.path.exists(evaluator_script):
        print(
            f'[-] Critical Error: internal evaluator script not found at {evaluator_script}',
            file=sys.stderr,
        )
        sys.exit(1)
        
    env = os.environ.copy()
    env["KFP_REFERENCE_DIR"] = ref_dir
    env["KFP_SAMPLES_DIR"] = os.path.join(ref_dir, "samples")

    print(f'[*] Launching evaluator: {evaluator_script}...', flush=True)

    result = subprocess.run([sys.executable, evaluator_script] + sys.argv[1:], env=env)
    sys.exit(result.returncode)


if __name__ == '__main__':
    reference_path = bootstrap_reference()
    launch_evaluator(reference_path)
