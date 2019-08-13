from typing import List, Union, Dict, Text, Any
import uuid

from kfp import dsl
from kfp.dsl import _metadata

ItemList = List[Union[int, float, str, Dict[Text, Any]]]


class LoopArguments(dsl.PipelineParam):
    PARAM_TYPE_NAME = 'loop_args'
    _loop_item_placeholder_name = 'loop-item-placeholder'

    def __init__(self, items: ItemList):
        self._code = uuid.uuid4().hex
        param_name = f'{self._loop_item_placeholder_name}-{self._code}'
        super().__init__(name=param_name, param_type=_metadata.TypeMeta(name=self.PARAM_TYPE_NAME))

        if not isinstance(items, (list, tuple)):
            raise ValueError(f"Expected list or tuple, got {type(items)}.")

        if len(items) == 0:
            self.is_dict_based = False
            self.items = items
        elif isinstance(items[0], dict):
            names = set(items[0].keys())
            for item in items:
                if not set(item.keys()) == names:
                    raise ValueError(f"If you input a list of dicts then all dicts should have the same keys. "
                                     f"Got: {items}.")

            # then this block creates loop_args.variable_a and loop_args.variable_b
            for name in names:
                setattr(self, name, dsl.PipelineParam(param_name))
            self.is_dict_based = True
            self.items = items
        else:
            self.is_dict_based = False
            self.items = items

    def to_list_for_task_yaml(self):
        return self.items


class LoopArgumentVariable(dsl.PipelineParam):
    PARAM_TYPE_NAME = 'loop_args_variable'

    def __init__(self, parent_full_name: Text, this_variable_name: Text):
        self._parent_name = parent_full_name
        self._this_variable_name = this_variable_name
        super().__init__(
            name=self.get_name(parent_full_name=parent_full_name, this_variable_name=this_variable_name),
            param_type=_metadata.TypeMeta(name=self.PARAM_TYPE_NAME),
        )

    @staticmethod
    def get_name(parent_full_name: Text, this_variable_name: Text):
        return f'{parent_full_name}-item_subvar-{this_variable_name}'

    def parent_name(self):
        return self._parent_name

    def variable_name(self):
        return self._this_variable_name

