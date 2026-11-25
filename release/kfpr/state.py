"""Checkpoint state persisted between release steps."""

import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List

STATE_FILE = 'release-state.json'


@dataclass
class ReleaseState:
    """Checkpoint for release process state.

    Stores answers to prompts and completed steps in JSON format.
    Problems with the file's shape are kept in load_errors, never saved.
    """

    path: Path
    answers: Dict[str, str] = field(default_factory=dict)
    completed_steps: List[str] = field(default_factory=list)
    load_errors: List[str] = field(
        default_factory=list, repr=False, compare=False
    )

    @classmethod
    def load(cls, path: Path) -> 'ReleaseState':
        """Load state from JSON file.

        Args:
          path: Path to state file.

        Returns:
          ReleaseState instance with loaded data.

        Raises:
          ValueError: If the state file contains malformed JSON.
        """
        if not path.exists():
            return cls(path)
        try:
            with open(path) as f:
                data = json.load(f)
        except json.JSONDecodeError as e:
            raise ValueError(
                f'Invalid state file: {path} contains malformed JSON. '
                f'Please fix or delete the file. Details: {e}'
            ) from e
        if not isinstance(data, dict):
            return cls(path, load_errors=['state file must be an object'])
        answers = data.get('answers', {})
        completed_steps = data.get('completed_steps', [])
        load_errors = []
        if not isinstance(answers, dict):
            answers = {}
            load_errors.append('answers must be an object')
        if not isinstance(completed_steps, list):
            completed_steps = []
            load_errors.append('completed_steps must be a list')
        elif not all(isinstance(step, str) for step in completed_steps):
            load_errors.append('completed_steps must be a list of strings')
        return cls(
            path=path,
            answers=answers,
            completed_steps=completed_steps,
            load_errors=load_errors,
        )

    def save(self) -> None:
        """Save state to JSON file atomically.

        Writes to a temporary file with .tmp suffix and then replaces
        the original file to ensure atomicity and avoid corruption on
        partial writes.

        Raises:
          ValueError: If the state was loaded from an invalid file, which
            saving would overwrite.
        """
        if self.load_errors:
            raise ValueError(
                f'Refusing to overwrite invalid state file {self.path}: '
                + '; '.join(self.load_errors)
            )
        data = {
            'answers': self.answers,
            'completed_steps': self.completed_steps,
        }
        self.path.parent.mkdir(parents=True, exist_ok=True)
        tmp_path = self.path.with_suffix(self.path.suffix + '.tmp')
        try:
            with open(tmp_path, 'w') as f:
                json.dump(data, f, indent=2)
            tmp_path.replace(self.path)
        except OSError:
            try:
                tmp_path.unlink(missing_ok=True)
            except OSError:
                pass
            raise

    def is_done(self, step_id: str) -> bool:
        """Check if a step has been marked as done.

        Args:
          step_id: Identifier for the step.

        Returns:
          True if the step is in completed_steps, False otherwise.
        """
        return step_id in self.completed_steps

    def mark_done(self, step_id: str) -> None:
        """Mark a step as done.

        Args:
          step_id: Identifier for the step.
        """
        if step_id not in self.completed_steps:
            self.completed_steps.append(step_id)

    def reset_step(self, step_id: str) -> None:
        """Remove a step from the completed checkpoint list."""
        self.completed_steps = [
            completed
            for completed in self.completed_steps
            if completed != step_id
        ]
