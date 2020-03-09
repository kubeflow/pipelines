import re
from typing import List, Union, Dict, Text, Any, Tuple, Optional

from kfp import dsl

ItemList = List[Union[int, float, str, Dict[Text, Any]]]


class LoopArguments(dsl.PipelineParam):
    """Class representing the arguments that are looped over in a ParallelFor loop in the KFP DSL.
    This doesn't need to be instantiated by the end user, rather it will be automatically created by a
    ParallelFor ops group."""
    LOOP_ITEM_NAME_BASE = 'loop-item'
    LOOP_ITEM_PARAM_NAME_BASE = 'loop-item-param'
    # number of characters in the code which is passed to the constructor
    NUM_CODE_CHARS = 8
    LEGAL_SUBVAR_NAME_REGEX = re.compile(r'[a-zA-Z_][0-9a-zA-Z_]*')

    @classmethod
    def _subvar_name_is_legal(cls, proposed_variable_name: Text):
        return re.match(cls.LEGAL_SUBVAR_NAME_REGEX, proposed_variable_name) is not None

    def __init__(self, items: Union[ItemList, dsl.PipelineParam], code: Text, name_override: Optional[Text]=None, op_name: Optional[Text]=None, *args, **kwargs):
        """LoopArguments represent the set of items to loop over in a ParallelFor loop.  This class shouldn't be
        instantiated by the user but rather is created by _ops_group.ParallelFor.

        Args:
            items: List of items to loop over.  If a list of dicts then, all dicts must have the same keys and every
                key must be a legal Python variable name.
            code: A unique code used to identify these loop arguments.  Should match the code for the ParallelFor
                ops_group which created these _LoopArguments.  This prevents parameter name collisions.
        """
        if name_override is None:
            super().__init__(name=self._make_name(code), *args, **kwargs)
        else:
            super().__init__(name=name_override, op_name=op_name, *args, **kwargs)

        if not isinstance(items, (list, tuple, dsl.PipelineParam)):
            raise TypeError("Expected list, tuple, or PipelineParam, got {}.".format(type(items)))

        if isinstance(items, tuple):
            items = list(items)

        if isinstance(items, list) and isinstance(items[0], dict):
            subvar_names = set(items[0].keys())
            for item in items:
                if not set(item.keys()) == subvar_names:
                    raise ValueError("If you input a list of dicts then all dicts should have the same keys. "
                                     "Got: {}.".format(items))

            # then this block creates loop_args.variable_a and loop_args.variable_b
            for subvar_name in subvar_names:
                if not self._subvar_name_is_legal(subvar_name):
                    raise ValueError("Tried to create subvariable named {} but that's not a legal Python variable "
                                     "name.".format(subvar_name))
                setattr(self, subvar_name, LoopArgumentVariable(self.name, subvar_name, loop_args_op_name=self.op_name))

        self.items_or_pipeline_param = items
        self.referenced_subvar_names = []

    @classmethod
    def from_pipeline_param(cls, param: dsl.PipelineParam) -> 'LoopArguments':
        return LoopArguments(
            items=param,
            code=None,
            name_override=param.name + '-' + cls.LOOP_ITEM_NAME_BASE,
            op_name=param.op_name,
            value=param.value,
        )

    def __getattr__(self, item):
        # this is being overridden so that we can access subvariables of the LoopArguments (i.e.: item.a) without
        # knowing the subvariable names ahead of time
        self.referenced_subvar_names.append(item)
        return LoopArgumentVariable(self.name, item, loop_args_op_name=self.op_name)

    def to_list_for_task_yaml(self):
        if isinstance(self.items_or_pipeline_param, (list, tuple)):
            return self.items_or_pipeline_param
        else:
            raise ValueError("You should only call this method on loop args which have list items, "
                             "not pipeline param items.")

    @classmethod
    def _make_name(cls, code: Text):
        """Make a name for this parameter.  Code is a """
        return '{}-{}'.format(cls.LOOP_ITEM_PARAM_NAME_BASE, code)

    @classmethod
    def name_is_withitems_loop_argument(cls, param_name: Text) -> bool:
        """Return True if the given parameter name looks like it came from a loop arguments parameter."""
        return re.match(
            '%s-[0-9a-f]{%s}' % (cls.LOOP_ITEM_PARAM_NAME_BASE, cls.NUM_CODE_CHARS),
            param_name,
        ) is not None

    @classmethod
    def name_is_withparams_loop_argument(cls, param_name: Text) -> bool:
        """Return True if the given parameter name looks like it came from a withParams loop item."""
        return ('-' + cls.LOOP_ITEM_NAME_BASE) in param_name

class LoopArgumentVariable(dsl.PipelineParam):
    """Represents a subvariable for loop arguments.  This is used for cases where we're looping over maps,
    each of which contains several variables."""
    SUBVAR_NAME_DELIMITER = '-subvar-'

    def __init__(
        self,
        loop_args_name: Text,
        this_variable_name: Text,
        loop_args_op_name: Text,
    ):
        """
        If the user ran:
            with dsl.ParallelFor([{'a': 1, 'b': 2}, {'a': 3, 'b': 4}]) as item:
                ...
        Then there's be one _LoopArgumentsVariable for 'a' and another for 'b'.

        Args:
            loop_args_name:  the name of the _LoopArguments object that this is a subvariable to.
            this_variable_name: the name of this subvariable, which is the name of the dict key that spawned
                this subvariable.
        """
        super().__init__(
            name=self.get_name(loop_args_name=loop_args_name,
            this_variable_name=this_variable_name),
            op_name=loop_args_op_name,
        )

    @classmethod
    def get_name(cls, loop_args_name: Text, this_variable_name: Text) -> Text:
        """Get the name

        Args:
            loop_args_name: the name of the loop args parameter that this LoopArgsVariable is attached to.
            this_variable_name: the name of this LoopArgumentsVariable subvar.

        Returns: The name of this loop args variable.
        """
        return '{}{}{}'.format(loop_args_name, cls.SUBVAR_NAME_DELIMITER, this_variable_name)

    @classmethod
    def name_is_loop_arguments_variable(cls, param_name: Text) -> bool:
        """Return True if the given parameter name looks like it came from a LoopArgumentsVariable."""
        return re.match(
            '.+%s.+' % cls.SUBVAR_NAME_DELIMITER,
            param_name
        ) is not None

    @classmethod
    def parse_loop_args_name_and_this_var_name(cls, t: Text) -> Tuple[Text, Text]:
        """Get the loop arguments param name and this subvariable name from the given parameter name."""
        m = re.match('(?P<loop_args_name>.*){}(?P<this_var_name>.*)'.format(cls.SUBVAR_NAME_DELIMITER), t)
        if m is None:
            return None
        else:
            return m.groupdict()['loop_args_name'], m.groupdict()['this_var_name']

    @classmethod
    def get_subvar_name(cls, t: Text) -> Text:
        """Get the subvariable name from a given LoopArgumentsVariable parameter name."""
        out = cls.parse_loop_args_name_and_this_var_name(t)
        if out is None:
            raise ValueError("Couldn't parse variable name: {}".format(t))
        return out[1]
