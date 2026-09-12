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

import os
import re
from typing import List, Tuple, Optional


def migrate_code(code: str) -> Tuple[str, List[str]]:
    """Transforms v1 code string to v2 and returns (migrated_code, list_of_warnings)."""
    warnings: List[str] = []
    modified = code

    # 1. Warn about ContainerOp
    if re.search(r'\bContainerOp\b', modified):
        warnings.append(
            "Found 'ContainerOp'. dsl.ContainerOp is removed in KFP v2. "
            "Please manually migrate this container task to use the @dsl.container_component decorator."
        )

    # 2. Upgrade kfp.v2 imports
    # from kfp.v2 import ... -> from kfp import ...
    modified, count = re.subn(r'\bfrom\s+kfp\.v2\s+import\b', 'from kfp import', modified)
    if count > 0:
        warnings.append(f"Updated {count} occurrence(s) of 'from kfp.v2 import' to 'from kfp import'.")

    # from kfp.v2.dsl import ... -> from kfp.dsl import ...
    # from kfp.v2.compiler import ... -> from kfp.compiler import ...
    modified, count = re.subn(r'\bfrom\s+kfp\.v2\.(\w+)\s+import\b', r'from kfp.\1 import', modified)
    if count > 0:
        warnings.append(f"Updated {count} occurrence(s) of 'from kfp.v2.<module>' to 'from kfp.<module>'.")

    # import kfp.v2.dsl -> import kfp.dsl
    modified, count = re.subn(r'\bimport\s+kfp\.v2\.(\w+)\b', r'import kfp.\1', modified)
    if count > 0:
        warnings.append(f"Updated {count} occurrence(s) of 'import kfp.v2.<module>' to 'import kfp.<module>'.")

    # import kfp.v2 -> import kfp
    modified, count = re.subn(r'\bimport\s+kfp\.v2\b', 'import kfp', modified)
    if count > 0:
        warnings.append(f"Updated {count} occurrence(s) of 'import kfp.v2' to 'import kfp'.")

    # 3. Upgrade components inputs/outputs imports
    # from kfp.components import InputPath, OutputPath -> from kfp.dsl import InputPath, OutputPath
    def replace_components_imports(match: re.Match) -> str:
        imports_block = match.group(1)
        imported_names = [name.strip() for name in imports_block.split(',')]
        dsl_imports = []
        comp_imports = []
        for name in imported_names:
            if name in ('InputPath', 'OutputPath'):
                dsl_imports.append(name)
            else:
                comp_imports.append(name)
        
        result_lines = []
        if dsl_imports:
            result_lines.append(f"from kfp.dsl import {', '.join(dsl_imports)}")
        if comp_imports:
            result_lines.append(f"from kfp.components import {', '.join(comp_imports)}")
        return '\n'.join(result_lines)

    modified, count = re.subn(
        r'\bfrom\s+kfp\.components\s+import\s+([a-zA-Z0-9_,\s]+)\b',
        replace_components_imports,
        modified
    )
    if count > 0:
        warnings.append("Updated 'from kfp.components import' statements to split 'InputPath'/'OutputPath' into 'kfp.dsl'.")

    # 4. Migrate func_to_container_op and create_component_from_func imports
    # from kfp.components import func_to_container_op -> from kfp.dsl import component
    # we need to be careful with multi-import lines.
    def replace_deprecated_factory_imports(match: re.Match) -> str:
        imports_block = match.group(1)
        imported_names = [name.strip() for name in imports_block.split(',')]
        dsl_component_imported = False
        remaining_comp_imports = []
        for name in imported_names:
            if name in ('func_to_container_op', 'create_component_from_func'):
                dsl_component_imported = True
            else:
                remaining_comp_imports.append(name)
        
        result_lines = []
        if dsl_component_imported:
            result_lines.append("from kfp.dsl import component")
        if remaining_comp_imports:
            result_lines.append(f"from kfp.components import {', '.join(remaining_comp_imports)}")
        return '\n'.join(result_lines)

    modified, count = re.subn(
        r'\bfrom\s+kfp\.components\s+import\s+([a-zA-Z0-9_,\s]+)\b',
        replace_deprecated_factory_imports,
        modified
    )
    if count > 0:
        warnings.append("Migrated legacy component factory imports (func_to_container_op / create_component_from_func) to 'kfp.dsl.component'.")

    # 5. Migrate usages of func_to_container_op and create_component_from_func
    # e.g., kfp.components.func_to_container_op -> kfp.dsl.component
    modified, count1 = re.subn(
        r'\bkfp\.components\.(func_to_container_op|create_component_from_func)\b',
        'kfp.dsl.component',
        modified
    )
    # components.func_to_container_op -> dsl.component
    modified, count2 = re.subn(
        r'\bcomponents\.(func_to_container_op|create_component_from_func)\b',
        'dsl.component',
        modified
    )
    # comp.func_to_container_op -> dsl.component
    modified, count3 = re.subn(
        r'\bcomp\.(func_to_container_op|create_component_from_func)\b',
        'dsl.component',
        modified
    )
    # func_to_container_op / create_component_from_func -> component
    modified, count4 = re.subn(
        r'\b(func_to_container_op|create_component_from_func)\b',
        'component',
        modified
    )
    total_factory_counts = count1 + count2 + count3 + count4
    if total_factory_counts > 0:
        warnings.append(f"Updated {total_factory_counts} usage(s) of legacy component factories to 'component'.")

    # Ensure from kfp import dsl is present if dsl is used
    if 'dsl.' in modified and not re.search(r'\bfrom\s+kfp\s+import\s+.*dsl\b', modified) and not re.search(r'\bimport\s+kfp\.dsl\b', modified):
        # Insert from kfp import dsl near the beginning, after future/docstring
        lines = modified.splitlines()
        insert_idx = 0
        for i, line in enumerate(lines[:10]):
            if line.startswith('import ') or line.startswith('from '):
                insert_idx = i
                break
        lines.insert(insert_idx, 'from kfp import dsl')
        modified = '\n'.join(lines)
        warnings.append("Added 'from kfp import dsl' since 'dsl.' was referenced.")

    # 6. Condition -> If migration
    # Replace from kfp.dsl import Condition with from kfp.dsl import If (or update inline)
    def replace_condition_import(match: re.Match) -> str:
        imports_block = match.group(1)
        imported_names = [name.strip() for name in imports_block.split(',')]
        new_names = []
        for name in imported_names:
            if name == 'Condition':
                new_names.append('If')
            else:
                new_names.append(name)
        # remove duplicates
        new_names = sorted(list(set(new_names)))
        return f"from kfp.dsl import {', '.join(new_names)}"

    modified, count = re.subn(
        r'\bfrom\s+kfp\.dsl\s+import\s+([a-zA-Z0-9_,\s]+)\b',
        replace_condition_import,
        modified
    )
    if count > 0:
        warnings.append("Updated Condition imports in 'from kfp.dsl import' to use 'If'.")

    # Replace usages of Condition with If (e.g. dsl.Condition, with Condition)
    modified, count1 = re.subn(r'\bdsl\.Condition\b', 'dsl.If', modified)
    modified, count2 = re.subn(r'\bwith\s+Condition\b', 'with If', modified)
    if count1 + count2 > 0:
        warnings.append(f"Updated {count1 + count2} usage(s) of 'Condition' to 'If'.")

    # 7. ParallelFor(loop_args=...) -> ParallelFor(items=...) migration
    def replace_parallel_for_args(match: re.Match) -> str:
        call_content = match.group(0)
        return call_content.replace('loop_args', 'items')

    modified, count = re.subn(
        r'\bParallelFor\s*\([^)]*\bloop_args\s*=',
        replace_parallel_for_args,
        modified
    )
    if count > 0:
        warnings.append(f"Updated {count} occurrence(s) of 'loop_args' in ParallelFor to 'items'.")

    return modified, warnings


def migrate(
    source_path: str,
    output_path: Optional[str] = None,
    inplace: bool = False,
) -> Optional[str]:
    """Reads a file/directory of v1 python code, transforms it, and writes the output or returns it.

    Args:
        source_path: Path to the source file or directory to migrate.
        output_path: Path to write the migrated file or directory.
        inplace: Whether to overwrite the source file(s) in-place.

    Returns:
        The migrated code string if source_path is a single file and neither output_path nor inplace is set.
        Otherwise, writes to disk and returns None.
    """
    if not os.path.exists(source_path):
        raise FileNotFoundError(f"Source path '{source_path}' does not exist.")

    if os.path.isdir(source_path):
        if not inplace and not output_path:
            raise ValueError(
                "When migrating a directory, you must specify either --inplace (-i) or --output-path (-o)."
            )
        
        # Walk directory
        for root, _, files in os.walk(source_path):
            for file in files:
                if file.endswith('.py'):
                    file_source_path = os.path.join(root, file)
                    
                    # Read content
                    with open(file_source_path, 'r', encoding='utf-8') as f:
                        code = f.read()
                    
                    migrated_code, warnings = migrate_code(code)
                    
                    # Determine where to write
                    if inplace:
                        file_output_path = file_source_path
                    else:
                        assert output_path is not None
                        rel_path = os.path.relpath(file_source_path, source_path)
                        file_output_path = os.path.join(output_path, rel_path)
                    
                    # Write content
                    os.makedirs(os.path.dirname(file_output_path), exist_ok=True)
                    with open(file_output_path, 'w', encoding='utf-8') as f:
                        f.write(migrated_code)
                    
                    if warnings:
                        print(f"Warnings for {file_source_path}:")
                        for warning in warnings:
                            print(f"  - {warning}")
        return None

    else:
        # Single file
        with open(source_path, 'r', encoding='utf-8') as f:
            code = f.read()
        
        migrated_code, warnings = migrate_code(code)
        
        if inplace:
            with open(source_path, 'w', encoding='utf-8') as f:
                f.write(migrated_code)
            if warnings:
                print(f"Warnings for {source_path}:")
                for warning in warnings:
                    print(f"  - {warning}")
            return None
        
        elif output_path:
            os.makedirs(os.path.dirname(output_path), exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                f.write(migrated_code)
            if warnings:
                print(f"Warnings for {source_path} (output written to {output_path}):")
                for warning in warnings:
                    print(f"  - {warning}")
            return None
        
        else:
            # Return string
            return migrated_code
