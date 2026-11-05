"""Command execution with optional dry-run mode."""

import subprocess
from pathlib import Path


class CommandRunner:
    """Runs commands with optional dry-run mode."""

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.commands: list[list[str]] = []

    def run(
        self, command: list[str], cwd: Path | None = None, check: bool = True
    ) -> subprocess.CompletedProcess[str]:
        """Run a command, recording it and optionally executing it.

        Args:
         command: Command as a list of strings.
         cwd: Working directory for command execution.
         check: Whether to raise on non-zero exit code.

        Returns:
         CompletedProcess instance.
        """
        self.commands.append(command)
        print(f'+ {" ".join(command)}')
        if self.dry_run:
            return subprocess.CompletedProcess(command, 0, '', '')
        return subprocess.run(command, cwd=cwd, check=check, text=True)

    def capture(self, command: list[str], cwd: Path | None = None) -> str:
        """Capture command output, recording command and optionally executing
        it.

        Args:
         command: Command as a list of strings.
         cwd: Working directory for command execution.

        Returns:
         Command output as string (empty string in dry-run mode).
        """
        self.commands.append(command)
        print(f'+ {" ".join(command)}')
        if self.dry_run:
            return ''
        return subprocess.check_output(command, cwd=cwd, text=True).strip()
