# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import sys
import os
import difflib
from typing import Optional

import click
from kfp.compiler import migration


@click.command(name='migrate')
@click.argument(
    'source-path',
    type=click.Path(exists=True, file_okay=True, dir_okay=True, readable=True),
)
@click.option(
    '--output-path',
    '-o',
    type=click.Path(writable=True),
    help='Path to write migrated file or directory. If specified, does not modify source files.',
)
@click.option(
    '--inplace',
    '-i',
    is_flag=True,
    default=False,
    help='Modify files in-place.',
)
def migrate(source_path: str, output_path: Optional[str] = None, inplace: bool = False) -> None:
    """Migrates KFP v1 SDK code to v2 SDK code.

    If neither --inplace nor --output-path is specified, runs a dry-run and prints the diff.
    """
    if os.path.isdir(source_path):
        if not inplace and not output_path:
            # Dry-run for a directory: run migration on each file and show diffs
            click.echo("Running dry-run for directory recursively. No files will be modified.\n", err=True)
            has_diffs = False
            for root, _, files in os.walk(source_path):
                for file in files:
                    if file.endswith('.py'):
                        file_source_path = os.path.join(root, file)
                        with open(file_source_path, 'r', encoding='utf-8') as f:
                            original_code = f.read()
                        
                        migrated_code, warnings = migration.migrate_code(original_code)
                        if original_code != migrated_code:
                            has_diffs = True
                            click.echo(f"--- Diff for {file_source_path} ---")
                            diff = difflib.unified_diff(
                                original_code.splitlines(keepends=True),
                                migrated_code.splitlines(keepends=True),
                                fromfile=f"a/{file_source_path}",
                                tofile=f"b/{file_source_path}"
                            )
                            sys.stdout.writelines(diff)
                            click.echo("\n")
                        
                        if warnings:
                            click.echo(f"Warnings for {file_source_path}:", err=True)
                            for warning in warnings:
                                click.echo(f"  - {warning}", err=True)
                            click.echo("\n", err=True)
            if not has_diffs:
                click.echo("No changes detected.")
            return

        # Perform migration of directory
        try:
            migration.migrate(source_path=source_path, output_path=output_path, inplace=inplace)
            click.echo("Directory migration completed successfully.")
        except Exception as e:
            click.echo(f"Error migrating directory: {e}", err=True)
            sys.exit(1)

    else:
        # Single file
        with open(source_path, 'r', encoding='utf-8') as f:
            original_code = f.read()

        if not inplace and not output_path:
            # Dry-run for single file
            click.echo("Running dry-run for file. No files will be modified.\n", err=True)
            migrated_code, warnings = migration.migrate_code(original_code)
            if original_code != migrated_code:
                diff = difflib.unified_diff(
                    original_code.splitlines(keepends=True),
                    migrated_code.splitlines(keepends=True),
                    fromfile=f"a/{source_path}",
                    tofile=f"b/{source_path}"
                )
                sys.stdout.writelines(diff)
            else:
                click.echo("No changes detected.")
            
            if warnings:
                click.echo("\nWarnings:", err=True)
                for warning in warnings:
                    click.echo(f"  - {warning}", err=True)
            return

        # Perform migration of single file
        try:
            migration.migrate(source_path=source_path, output_path=output_path, inplace=inplace)
            click.echo(f"File migration completed successfully. Output path: {output_path or source_path}")
        except Exception as e:
            click.echo(f"Error migrating file: {e}", err=True)
            sys.exit(1)
