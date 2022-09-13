# Copyright 2022 The Kubeflow Authors
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

import ast
import inspect
import sys
from typing import Any, Callable, Dict, List

from kfp.components import component_factory
from kfp.components.types import type_annotations

PYTHON_VERSION = str(sys.version_info.major) + '.' + str(sys.version_info.minor)


def get_custom_artifact_type_import_statements(func: Callable) -> List[str]:
    """Gets a list of custom artifact type import statements from a lightweight
    Python component function."""
    artifact_imports = get_custom_artifact_import_items_from_function(func)
    imports_source = []
    for obj_str in artifact_imports:
        if '.' in obj_str:
            path, name = obj_str.rsplit('.', 1)
            imports_source.append(f'from {path} import {name}')
        else:
            imports_source.append(f'import {obj_str}')
    return imports_source


def get_param_to_annotation_object(func: Callable) -> Dict[str, Any]:
    """Gets a dictionary of parameter name to type annotation object from a
    function.

    Args:
        func: The function.

    Returns:
        Dictionary of parameter name to type annotation object.
    """
    signature = inspect.signature(func)
    return {
        name: parameter.annotation
        for name, parameter in signature.parameters.items()
    }


def get_full_qualname_for_artifact(obj: type) -> str:
    """Gets the fully qualified name for an object. For example, for class Foo
    in module bar.baz, this function returns bar.baz.Foo.

    Note: typing.get_type_hints purports to do the same thing, but it behaves
    differently when executed within the scope of a test, so preferring this
    approach instead.

    Args:
        obj: The class or module for which to get the fully qualified name.

    Returns:
        The fully qualified name for the class.
    """
    module = obj.__module__
    name = obj.__qualname__
    if module is not None:
        name = module + '.' + name
    return name


def get_import_item_from_annotation_string_and_ann_obj(artifact_str: str,
                                                       qualname: str) -> str:
    """Gets the fully qualified names of the module or type to import from the
    annotation string and the annotation object.

    Args:
        artifact_str: The type annotation string.
        qualname: The fully qualified type annotation name as a string.

    Returns:
        The fully qualified names of the module or type to import.
    """
    split_qualname = qualname.split('.')
    if artifact_str in split_qualname:
        name_to_import = '.'.join(
            split_qualname[:split_qualname.index(artifact_str) + 1])
    else:
        raise TypeError(
            f"Module or type name aliases are not supported. You appear to be using an alias in your type annotation: '{qualname}'. This may be due to use of an 'as' statement in an import statement or a reassignment of a module or type to a new name. Reference the module and/or type using the name as defined in the source from which the module or type is imported."
        )
    return name_to_import


def func_to_annotation_object(func: Callable) -> Dict[str, str]:
    """Gets a dict of parameter name to annotation object for a function."""
    signature = inspect.signature(func)
    return {
        name: parameter.annotation
        for name, parameter in signature.parameters.items()
    }


def func_to_root_annotation_symbols_for_artifacts(
        func: Callable) -> Dict[str, str]:
    """Gets a map of parameter name to the root symbol used to reference an
    annotation for a function.

    For example, the annotation `type_module.Artifact` has the root
    symbol `type_module` and the annotation `Artifact` has the root
    symbol `Artifact`.
    """
    module_node = ast.parse(
        component_factory._get_function_source_definition(func))
    args = module_node.body[0].args.args

    def convert_ast_arg_node_to_param_annotation_dict_for_artifacts(
            args: List[ast.arg]) -> Dict[str, str]:
        """Converts a list of ast.arg nodes to a dictionary of the parameter
        name to type annotation as a string (for artifact annotations only).

        Args:
            args: AST argument nodes for a function.

        Returns:
            A dictionary of parameter name to type annotation as a string.
        """
        arg_to_ann = {}
        for arg in args:
            annotation = arg.annotation
            # Handle InputPath() and OutputPath()
            if isinstance(annotation, ast.Call):
                if isinstance(annotation.func, ast.Name):
                    arg_to_ann[arg.arg] = annotation.func.id
                elif isinstance(annotation.func, ast.Attribute):
                    arg_to_ann[arg.arg] = annotation.func.value.id
                else:
                    raise TypeError(
                        f'Unexpected type annotation for {annotation.func}.')
            # annotations with a subtype like Input[Artifact]
            elif isinstance(annotation, ast.Subscript):
                # get inner type
                if PYTHON_VERSION < '3.9':
                    # see "Changed in version 3.9" https://docs.python.org/3/library/ast.html#node-classes
                    if isinstance(annotation.slice.value, ast.Name):
                        arg_to_ann[arg.arg] = annotation.slice.value.id
                    if isinstance(annotation.slice.value, ast.Attribute):
                        arg_to_ann[arg.arg] = annotation.slice.value.value.id
                else:
                    if isinstance(annotation.slice, ast.Name):
                        arg_to_ann[arg.arg] = annotation.slice.id
                    if isinstance(annotation.slice, ast.Attribute):
                        arg_to_ann[arg.arg] = annotation.slice.value.id
            # annotations like type_annotations.Input[Artifact]
            elif isinstance(annotation, ast.Attribute):
                arg_to_ann[arg.arg] = annotation.value.id
        return arg_to_ann

    return convert_ast_arg_node_to_param_annotation_dict_for_artifacts(args)


def get_custom_artifact_import_items_from_function(func: Callable) -> List[str]:
    """Gets a list of fully qualified names of the modules or types to import.

    Args:
        func: The component function.

    Returns:
        A list containing the fully qualified names of the module or type to import.
    """

    param_to_ann_obj = func_to_annotation_object(func)
    param_to_ann_string = func_to_root_annotation_symbols_for_artifacts(func)

    import_items = []
    seen = set()
    for param_name, ann_string in param_to_ann_string.items():
        # don't process the same annotation string multiple times
        # use the string, not the object in the seen set, since the same object can be referenced different ways in the same function signature
        if ann_string in seen:
            continue
        else:
            seen.add(ann_string)

        actual_ann = param_to_ann_obj.get(param_name)

        if actual_ann is not None and type_annotations.is_artifact_annotation(
                actual_ann):
            artifact_class = type_annotations.get_io_artifact_class(actual_ann)
            artifact_qualname = get_full_qualname_for_artifact(artifact_class)
            if not artifact_qualname.startswith('kfp.'):
                # kfp artifacts are already imported by default
                import_items.append(
                    get_import_item_from_annotation_string_and_ann_obj(
                        ann_string, artifact_qualname))

    return import_items
