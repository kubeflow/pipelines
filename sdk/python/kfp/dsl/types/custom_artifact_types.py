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
from typing import Callable, Dict, List, Union

from kfp.dsl import component_factory
from kfp.dsl.types import type_annotations
from kfp.dsl.types import type_utils

RETURN_PREFIX = 'return-'


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


def get_param_to_custom_artifact_class(func: Callable) -> Dict[str, type]:
    """Gets a map of parameter names to custom artifact classes.

    Return key is 'return-' for normal returns and 'return-<field>' for
    typing.NamedTuple returns.
    """
    param_to_artifact_cls: Dict[str, type] = {}
    kfp_artifact_classes = set(type_utils.ARTIFACT_CLASSES_MAPPING.values())

    signature = inspect.signature(func)
    for name, param in signature.parameters.items():
        annotation = param.annotation
        if type_annotations.is_Input_Output_artifact_annotation(annotation):
            artifact_class = type_annotations.get_io_artifact_class(annotation)
            if artifact_class not in kfp_artifact_classes:
                param_to_artifact_cls[name] = artifact_class
        elif type_annotations.issubclass_of_artifact(annotation):
            if annotation not in kfp_artifact_classes:
                param_to_artifact_cls[name] = artifact_class

    return_annotation = signature.return_annotation

    if return_annotation is inspect.Signature.empty:
        pass

    elif type_utils.is_typed_named_tuple_annotation(return_annotation):
        for name, annotation in return_annotation.__annotations__.items():
            if type_annotations.is_artifact_class(
                    annotation) and annotation not in kfp_artifact_classes:
                param_to_artifact_cls[f'{RETURN_PREFIX}{name}'] = annotation

    elif type_annotations.is_artifact_class(
            return_annotation
    ) and return_annotation not in kfp_artifact_classes:
        param_to_artifact_cls[RETURN_PREFIX] = return_annotation

    return param_to_artifact_cls


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


def get_symbol_import_path(artifact_class_base_symbol: str,
                           qualname: str) -> str:
    """Gets the fully qualified name of the symbol that must be imported for
    the custom artifact type annotation to be referenced successfully.

    Args:
        artifact_class_base_symbol: The base symbol from which the artifact class is referenced (e.g., aiplatform for aiplatform.VertexDataset).
        qualname: The fully qualified type annotation name as a string.

    Returns:
        The fully qualified names of the module or type to import.
    """
    split_qualname = qualname.split('.')
    if artifact_class_base_symbol in split_qualname:
        name_to_import = '.'.join(
            split_qualname[:split_qualname.index(artifact_class_base_symbol) +
                           1])
    else:
        raise TypeError(
            f"Module or type name aliases are not supported. You appear to be using an alias in your type annotation: '{qualname}'. This may be due to use of an 'as' statement in an import statement or a reassignment of a module or type to a new name. Reference the module and/or type using the name as defined in the source from which the module or type is imported."
        )
    return name_to_import


def traverse_ast_node_values_to_get_id(obj: Union[ast.Slice, None]) -> str:
    while not hasattr(obj, 'id'):
        obj = getattr(obj, 'value')
    return obj.id


def get_custom_artifact_base_symbol_for_parameter(func: Callable,
                                                  arg_name: str) -> str:
    """Gets the symbol required for the custom artifact type annotation to be
    referenced correctly."""
    module_node = ast.parse(
        component_factory._get_function_source_definition(func))
    args = module_node.body[0].args.args
    args = {arg.arg: arg for arg in args}
    annotation = args[arg_name].annotation
    return traverse_ast_node_values_to_get_id(annotation.slice)


def get_custom_artifact_base_symbol_for_return(func: Callable,
                                               return_name: str) -> str:
    """Gets the symbol required for the custom artifact type return annotation
    to be referenced correctly."""
    module_node = ast.parse(
        component_factory._get_function_source_definition(func))
    return_ann = module_node.body[0].returns

    if return_name == RETURN_PREFIX:
        if isinstance(return_ann, (ast.Name, ast.Attribute)):
            return traverse_ast_node_values_to_get_id(return_ann)
    elif isinstance(return_ann, ast.Call):
        func = return_ann.func
        # handles NamedTuple and typing.NamedTuple
        if (isinstance(func, ast.Attribute) and func.value.id == 'typing' and
                func.attr == 'NamedTuple') or (isinstance(func, ast.Name) and
                                               func.id == 'NamedTuple'):
            nt_field_list = return_ann.args[1].elts
            for el in nt_field_list:
                if f'{RETURN_PREFIX}{el.elts[0].s}' == return_name:
                    return traverse_ast_node_values_to_get_id(el.elts[1])

    raise TypeError(f"Unexpected type annotation '{return_ann}' for {func}.")


def get_custom_artifact_import_items_from_function(func: Callable) -> List[str]:
    """Gets the fully qualified name of the symbol that must be imported for
    the custom artifact type annotation to be referenced successfully from a
    component function."""

    param_to_ann_obj = get_param_to_custom_artifact_class(func)
    import_items = []
    for param_name, artifact_class in param_to_ann_obj.items():

        base_symbol = get_custom_artifact_base_symbol_for_return(
            func, param_name
        ) if param_name.startswith(
            RETURN_PREFIX) else get_custom_artifact_base_symbol_for_parameter(
                func, param_name)
        artifact_qualname = get_full_qualname_for_artifact(artifact_class)
        symbol_import_path = get_symbol_import_path(base_symbol,
                                                    artifact_qualname)

        # could use set here, but want to be have deterministic import ordering
        # in compilation
        if symbol_import_path not in import_items:
            import_items.append(symbol_import_path)

    return import_items
