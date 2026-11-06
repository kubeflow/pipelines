"""Regenerate release previews while retaining reviewed Markdown edits."""

import json
import os
import re
import subprocess
from pathlib import Path
from tempfile import NamedTemporaryFile, TemporaryDirectory, mkdtemp

from .generated import restore_generated_sections
from .ranges import ReleaseRange


def has_conflicts(text: str) -> bool:
    """Detect unresolved Git merge markers before publication."""
    return bool(
        re.search(r'(?m)^(?:<{7,}(?: |$)|>{7,}(?: |$)|[|]{7,}(?: |$))', text)
    )


def _write_atomic(path: Path, contents: bytes) -> None:
    temporary = None
    try:
        with NamedTemporaryFile(dir=path.parent, delete=False) as file:
            temporary = Path(file.name)
            file.write(contents)
        os.replace(temporary, path)
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


def regenerate_preview(
    path: Path, selected: ReleaseRange, generated_notes: str
) -> bool:
    """Merge reviewed edits onto newly generated notes, retaining backups.

    Return False when the saved preview needs manual conflict
    resolution. Invalid inputs or merge failures leave the preview and
    sidecar untouched.
    """
    provenance_path = path.with_suffix('.json')
    try:
        original = path.read_bytes()
        original_provenance = provenance_path.read_bytes()
        current = original.decode('utf-8')
        provenance = json.loads(original_provenance)
    except (OSError, UnicodeError, ValueError) as error:
        raise ValueError(
            'Cannot read the preview and provenance. Preserve both files and '
            'restore their original generated baseline before regenerating.'
        ) from error
    if (
        not isinstance(provenance, dict)
        or not all(
            isinstance(provenance.get(key), str) and provenance[key].strip()
            for key in (
                'base_ref',
                'target_ref',
                'base_sha',
                'target_sha',
                'generated_notes',
            )
        )
        or not all(
            re.fullmatch(r'(?:[0-9a-f]{40}|[0-9a-f]{64})', provenance[key])
            for key in ('base_sha', 'target_sha')
        )
    ):
        raise ValueError(
            'Preview provenance lacks a valid range or generated_notes baseline. '
            'Preserve the existing files, generate a fresh preview separately, '
            'and reapply reviewed edits manually.'
        )
    baseline = provenance['generated_notes']
    generated_notes = generated_notes.rstrip() + '\n'
    if not current.strip() or not generated_notes.strip():
        raise ValueError(
            'Preview and regenerated notes must be nonempty before regeneration.'
        )
    if any(
        has_conflicts(text) for text in (current, baseline, generated_notes)
    ):
        raise ValueError(
            'Resolve existing conflict markers before regenerating the preview.'
        )
    current = restore_generated_sections(current, baseline)
    with TemporaryDirectory(prefix='kfpr-preview-merge-') as directory:
        inputs = [
            Path(directory) / name
            for name in ('reviewed.md', 'baseline.md', 'generated.md')
        ]
        for file, text in zip(inputs, (current, baseline, generated_notes)):
            file.write_bytes(text.encode('utf-8'))
        try:
            result = subprocess.run(
                [
                    'git',
                    'merge-file',
                    '-p',
                    '-L',
                    'reviewed notes',
                    '-L',
                    'previous generated notes',
                    '-L',
                    'new generated notes',
                    *map(str, inputs),
                ],
                capture_output=True,
                check=False,
            )
        except OSError as error:
            raise ValueError(
                'Cannot run git merge-file. Install Git and retry regeneration.'
            ) from error
    if not 0 <= result.returncode <= 127:
        raise ValueError(
            'git merge-file failed; existing preview and provenance were not changed.'
        )
    try:
        merged = result.stdout.decode('utf-8')
    except UnicodeError as error:
        raise ValueError(
            'git merge-file produced invalid Markdown; existing files were not changed.'
        ) from error
    clean = result.returncode == 0
    conflict_markers = has_conflicts(merged)
    if (
        not merged.strip()
        or (clean and conflict_markers)
        or (not clean and not conflict_markers)
    ):
        raise ValueError(
            'git merge-file produced an unexpected result; existing files were not changed.'
        )
    if clean:
        merged = restore_generated_sections(merged, generated_notes)
    updated = selected.as_dict() | {'generated_notes': generated_notes}
    encoded_provenance = (json.dumps(updated, indent=2) + '\n').encode('utf-8')
    if (
        path.read_bytes() != original
        or provenance_path.read_bytes() != original_provenance
    ):
        raise ValueError(
            'Preview changed during regeneration. Review the latest edits and retry.'
        )
    backup = Path(mkdtemp(prefix=path.stem + '.backup-', dir=path.parent))
    try:
        (backup / 'original.md').write_bytes(original)
        (backup / 'provenance.json').write_bytes(original_provenance)
    except OSError as error:
        raise ValueError(
            'Could not create a complete backup; existing preview and provenance '
            'were not changed. Check available disk space and permissions.'
        ) from error
    print(f'Release notes backup: {backup}')
    try:
        _write_atomic(path, merged.encode('utf-8'))
        _write_atomic(provenance_path, encoded_provenance)
    except OSError as error:
        restore_failed = False
        for file, contents in (
            (path, original),
            (provenance_path, original_provenance),
        ):
            try:
                _write_atomic(file, contents)
            except OSError:
                restore_failed = True
        recovery = (
            f'Restore originals from {backup} before retrying.'
            if restore_failed
            else f'Original files restored; backup retained at {backup}.'
        )
        raise ValueError(
            f'Could not save regenerated notes. {recovery}'
        ) from error
    return clean
