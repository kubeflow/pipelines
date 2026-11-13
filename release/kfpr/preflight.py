"""Non-mutating preflight diagnostics for the release environment."""

import shutil
import subprocess
from pathlib import Path

from .validation import validate_answers


def doctor_errors(answers: dict[str, object], root: Path) -> list[str]:
    """Return release preflight diagnostics without mutating state."""
    _, errors = validate_answers(answers)
    for tool in ['git', 'gh', 'docker', 'python3']:
        if shutil.which(tool) is None:
            errors.append(f'missing required tool: {tool}')

    if shutil.which('gh') is not None:
        auth = subprocess.run(
            ['gh', 'auth', 'status'],
            check=False,
            capture_output=True,
            text=True,
        )
        if auth.returncode != 0:
            errors.append('gh auth status failed')

    if shutil.which('git') is not None:
        status = subprocess.run(
            ['git', 'status', '--short'],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
        )
        if status.returncode != 0:
            errors.append('git status failed')
        elif status.stdout.strip():
            errors.append('git working tree is dirty')
    return errors
