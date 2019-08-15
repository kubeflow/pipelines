import re
from typing import List, Union, Dict, Text, Any, Optional
import uuid

from kfp import dsl
from kfp.dsl import _metadata

ItemList = List[Union[int, float, str, Dict[Text, Any]]]


class LoopArguments(dsl.PipelineParam):
    PARAM_TYPE_NAME = 'loop_args'
    _loop_item_placeholder_name = 'loop-item-placeholder'

    @classmethod
    def param_is_this_type(cls, param: dsl.PipelineParam):
        return param.param_type.name == cls.PARAM_TYPE_NAME

    @classmethod
    def make_name(cls, code: Text):
        return f'{cls._loop_item_placeholder_name}-{code}'

    @classmethod
    def name_is_loop_arguments(cls, param_name: Text):
        return re.match(
            '%s-[0-9a-f]{32}' % cls._loop_item_placeholder_name,
            param_name
        ) is not None

    def __init__(self, items: ItemList, code: Text, op_name: Text):
        super().__init__(
            name=self.make_name(code),
            op_name=op_name,
            param_type=_metadata.TypeMeta(name=self.PARAM_TYPE_NAME),
        )

        if not isinstance(items, (list, tuple)):
            raise ValueError(f"Expected list or tuple, got {type(items)}.")

        if len(items) == 0:
            self.is_dict_based = False
            self.items = items
        elif isinstance(items[0], dict):
            subvar_names = set(items[0].keys())
            for item in items:
                if not set(item.keys()) == subvar_names:
                    raise ValueError(f"If you input a list of dicts then all dicts should have the same keys. "
                                     f"Got: {items}.")

            # then this block creates loop_args.variable_a and loop_args.variable_b
            for subvar_name in subvar_names:
                setattr(self, subvar_name, dsl.LoopArgumentVariable(self.name, subvar_name, op_name=op_name))
            self.is_dict_based = True
            self.items = items
        else:
            self.is_dict_based = False
            self.items = items

    def to_list_for_task_yaml(self):
        return self.items


class LoopArgumentVariable(dsl.PipelineParam):
    PARAM_TYPE_NAME = 'loop_args_variable'
    SUBVAR_NAME_DELIMITER = '-item-subvar-'

    def __init__(self, loop_args_name: Text, this_variable_name: Text, op_name: Optional[Text]=None):
        super().__init__(
            name=self.get_name(loop_args_name=loop_args_name, this_variable_name=this_variable_name),
            param_type=_metadata.TypeMeta(name=self.PARAM_TYPE_NAME),
            op_name=op_name,
        )

    @classmethod
    def get_name(cls, loop_args_name: Text, this_variable_name: Text):
        return f'{loop_args_name}{cls.SUBVAR_NAME_DELIMITER}{this_variable_name}'

    @classmethod
    def param_is_this_type(cls, param: dsl.PipelineParam):
        return param.param_type.name == cls.PARAM_TYPE_NAME

    @classmethod
    def name_is_loop_arguments_variable(cls, param_name: Text):
        return re.match(
            '%s-[0-9a-f]{32}%s.*' % (LoopArguments._loop_item_placeholder_name, cls.SUBVAR_NAME_DELIMITER),
            param_name
        ) is not None

    @classmethod
    def parse_loop_args_name_and_this_var_name(cls, t: Text):
        m = re.match(f'(?P<loop_args_name>.*){cls.SUBVAR_NAME_DELIMITER}(?P<this_var_name>.*)', t)
        if m is None:
            return None
        else:
            return m.groupdict()['loop_args_name'], m.groupdict()['this_var_name']

    @classmethod
    def get_subvar_name(cls, t: Text):
        out = cls.parse_loop_args_name_and_this_var_name(t)
        if out is None:
            raise ValueError(f"Couldn't parse variable name: {t}")
        return out[1]
