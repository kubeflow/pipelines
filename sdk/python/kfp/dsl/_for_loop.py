import re
from typing import List, Union, Dict, Text, Any, Optional

from kfp import dsl
from kfp.dsl import _metadata

ItemList = List[Union[int, float, str, Dict[Text, Any]]]


class LoopArguments(dsl.PipelineParam):
    """Class representing the arguments that are looped over in a ParallelFor loop in the KFP DSL.
    This doesn't need to be instantiated by the end user, rather it will be automatically created by a
    ParallelFor ops group."""
    _loop_item_placeholder_name = 'loop-item-placeholder'

    @classmethod
    def param_is_this_type(cls, param: dsl.PipelineParam):
        """Return True if the given param is a LoopArgument param."""
        return cls.name_is_loop_arguments(param.name)

    @classmethod
    def make_name(cls, code: Text):
        """Make a name for this parameter."""
        return '{}-{}'.format(cls._loop_item_placeholder_name, code)

    @classmethod
    def name_is_loop_arguments(cls, param_name: Text):
        """Return True if the given parameter name looks like it came from a loop arguments parameter."""
        return re.match(
            '%s-[0-9a-f]{32}' % cls._loop_item_placeholder_name,
            param_name
        ) is not None

    def __init__(self, items: ItemList, code: Text):
        super().__init__(name=self.make_name(code))

        if not isinstance(items, (list, tuple)):
            raise ValueError("Expected list or tuple, got {}.".format(type(items)))

        if isinstance(items[0], dict):
            subvar_names = set(items[0].keys())
            for item in items:
                if not set(item.keys()) == subvar_names:
                    raise ValueError("If you input a list of dicts then all dicts should have the same keys. "
                                     "Got: {}.".format(items))

            # then this block creates loop_args.variable_a and loop_args.variable_b
            for subvar_name in subvar_names:
                setattr(self, subvar_name, dsl.LoopArgumentVariable(self.name, subvar_name))

        self.items = items

    def to_list_for_task_yaml(self):
        return self.items


class LoopArgumentVariable(dsl.PipelineParam):
    """Represents a subvariable for loop arguments.  This is used for cases where we're looping over maps,
    each of which contains several variables."""
    SUBVAR_NAME_DELIMITER = '-item-subvar-'

    def __init__(self, loop_args_name: Text, this_variable_name: Text):
        super().__init__(name=self.get_name(loop_args_name=loop_args_name, this_variable_name=this_variable_name))

    @classmethod
    def get_name(cls, loop_args_name: Text, this_variable_name: Text):
        """Get the name

        Args:
            loop_args_name: the name of the loop args parameter that this LoopArgsVariable is attached to.
            this_variable_name: the name of this LoopArgumentsVariable subvar.

        Returns: The name of this loop args variable.
        """
        return '{}{}{}'.format(loop_args_name, cls.SUBVAR_NAME_DELIMITER, this_variable_name)

    @classmethod
    def param_is_this_type(cls, param: dsl.PipelineParam):
        """Return True if the given param is a LoopArgumentVariable param."""
        return cls.name_is_loop_arguments_variable(param.name)

    @classmethod
    def name_is_loop_arguments_variable(cls, param_name: Text):
        """Return True if the given parameter name looks like it came from a LoopArgumentsVariable."""
        return re.match(
            '%s-[0-9a-f]{32}%s.*' % (LoopArguments._loop_item_placeholder_name, cls.SUBVAR_NAME_DELIMITER),
            param_name
        ) is not None

    @classmethod
    def parse_loop_args_name_and_this_var_name(cls, t: Text):
        """Get the loop arguments param name and this subvariable name from the given parameter name."""
        m = re.match('(?P<loop_args_name>.*){}(?P<this_var_name>.*)'.format(cls.SUBVAR_NAME_DELIMITER), t)
        if m is None:
            return None
        else:
            return m.groupdict()['loop_args_name'], m.groupdict()['this_var_name']

    @classmethod
    def get_subvar_name(cls, t: Text):
        """Get the subvariable name from a given LoopArgumentsVariable parameter name."""
        out = cls.parse_loop_args_name_and_this_var_name(t)
        if out is None:
            raise ValueError("Couldn't parse variable name: {}".format(t))
        return out[1]
