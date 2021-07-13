import re
from contextlib import contextmanager
from functools import wraps
from typing import List, Union, Dict, Text, Any, Tuple, Optional, Callable, \
  TypeVar

from kfp import dsl

ItemList = List[Union[int, float, str, Dict[Text, Any]]]


_loop_arguments_decomposing = True

@contextmanager
def loop_arguments_decomposing_deactivated():
  """Deactivates decomposing in LoopArguments. Any field access works as in PipelineParam."""
  global _loop_arguments_decomposing
  old_loop_arguments_decomposing = _loop_arguments_decomposing
  _loop_arguments_decomposing = False
  try:
    yield
  finally:
    _loop_arguments_decomposing = old_loop_arguments_decomposing

_C = TypeVar('_C', bound=Callable)
def with_loop_arguments_decomposing_deactivated(f: _C) -> _C:
  @wraps(f)
  def inner(*args, **kwargs):
    with loop_arguments_decomposing_deactivated():
      return f(*args, **kwargs)
  return inner

class LoopArguments(dsl.PipelineParam):
  """Class representing the arguments that are looped over in a ParallelFor loop in the KFP DSL.

  This doesn't need to be instantiated by the end user, rather it will be
  automatically created by a
  ParallelFor ops group.
  """
  LOOP_ITEM_NAME_BASE = 'loop-item'
  LOOP_ITEM_PARAM_NAME_BASE = 'loop-item-param'
  # number of characters in the code which is passed to the constructor
  NUM_CODE_CHARS = 8
  LEGAL_SUBVAR_NAME_REGEX = re.compile(r'[a-zA-Z_][0-9a-zA-Z_]*')

  @classmethod
  def _subvar_name_is_legal(cls, proposed_variable_name: Text):
    return re.match(cls.LEGAL_SUBVAR_NAME_REGEX,
                    proposed_variable_name) is not None

  @with_loop_arguments_decomposing_deactivated
  def __init__(self,
               items: Union[ItemList, dsl.PipelineParam],
               code: Text,
               name_override: Optional[Text] = None,
               op_name: Optional[Text] = None,
               *args,
               **kwargs):
    """LoopArguments represent the set of items to loop over in a ParallelFor loop.

    This class shouldn't be instantiated by the user but rather is created by
    _ops_group.ParallelFor.

    Args:
      items: List of items to loop over.  If a list of dicts then, all
        dicts must have the same keys and every key must be a legal Python
        variable name.
      code: A unique code used to identify these loop arguments.  Should
        match the code for the ParallelFor ops_group which created these
        _LoopArguments.  This prevents parameter name collisions.
    """
    if name_override is None:
      super().__init__(name=self._make_name(code), *args, **kwargs)
    else:
      super().__init__(name=name_override, op_name=op_name, *args, **kwargs)

    if not isinstance(items, (list, tuple, dsl.PipelineParam)):
      raise TypeError('Expected list, tuple, or PipelineParam, got {}.'.format(
          type(items)))

    if isinstance(items, tuple):
      items = list(items)

    if isinstance(items, list) and isinstance(items[0], dict):
      subvar_names = set(items[0].keys())
      for item in items:
        if not set(item.keys()) == subvar_names:
          raise ValueError(
              'If you input a list of dicts then all dicts should have the same keys. '
              'Got: {}.'.format(items))

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

  def __getattribute__(self, item: str) -> 'LoopArgumentVariable':
    """When constructing pipeline, gets a sub-variable of the loop item."""
    global _loop_arguments_decomposing

    # if no decomposing is expected - behave like a normal class
    if not _loop_arguments_decomposing:
      return super().__getattribute__(item)

    # otherwise - behave like a dictionary accessed with dots
    referenced_subvar_names = super().__getattribute__('referenced_subvar_names')
    name = super().__getattribute__('name')
    op_name = super().__getattribute__('op_name')

    referenced_subvar_names.append(item)
    return LoopArgumentVariable(name, item, loop_args_op_name=op_name)

  def to_list_for_task_yaml(self):
    if isinstance(self.items_or_pipeline_param, (list, tuple)):
      return self.items_or_pipeline_param
    else:
      raise ValueError(
          'You should only call this method on loop args which have list items, '
          'not pipeline param items.')

  @classmethod
  def _make_name(cls, code: Text):
    """Make a name for this parameter.  Code is a """
    return '{}-{}'.format(cls.LOOP_ITEM_PARAM_NAME_BASE, code)

  @classmethod
  def name_is_loop_argument(cls, param_name: Text) -> bool:
    """Return True if the given parameter name looks like a loop argument.

    Either it came from a withItems loop item or withParams loop item.
    """
    return cls.name_is_withitems_loop_argument(param_name) \
      or cls.name_is_withparams_loop_argument(param_name)

  @classmethod
  def name_is_withitems_loop_argument(cls, param_name: Text) -> bool:
    """Return True if the given parameter name looks like it came from a loop arguments parameter."""
    return (cls.LOOP_ITEM_PARAM_NAME_BASE + '-') in param_name

  @classmethod
  def name_is_withparams_loop_argument(cls, param_name: Text) -> bool:
    """Return True if the given parameter name looks like it came from a withParams loop item."""
    return ('-' + cls.LOOP_ITEM_NAME_BASE) in param_name


class LoopArgumentVariable(dsl.PipelineParam):
  """Represents a subvariable for loop arguments.

  This is used for cases where we're looping over maps,
    each of which contains several variables.
  """
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
      loop_args_name:  the name of the _LoopArguments object that this is
        a subvariable to.
      this_variable_name: the name of this subvariable, which is the name
        of the dict key that spawned this subvariable.
    """
    super().__init__(
        name=self.get_name(
            loop_args_name=loop_args_name,
            this_variable_name=this_variable_name),
        op_name=loop_args_op_name,
    )

  @classmethod
  def get_name(cls, loop_args_name: Text, this_variable_name: Text) -> Text:
    """Get the name

    Args:
      loop_args_name: the name of the loop args parameter that this
        LoopArgsVariable is attached to.
      this_variable_name: the name of this LoopArgumentsVariable subvar.

    Returns:
      The name of this loop args variable.
    """
    return '{}{}{}'.format(loop_args_name, cls.SUBVAR_NAME_DELIMITER,
                           this_variable_name)

  @classmethod
  def name_is_loop_arguments_variable(cls, param_name: Text) -> bool:
    """Return True if the given parameter name looks like it came from a LoopArgumentsVariable."""
    return re.match('.+%s.+' % cls.SUBVAR_NAME_DELIMITER,
                    param_name) is not None

  @classmethod
  def parse_loop_args_name_and_this_var_name(cls, t: Text) -> Tuple[Text, Text]:
    """Get the loop arguments param name and this subvariable name from the given parameter name."""
    m = re.match(
        '(?P<loop_args_name>.*){}(?P<this_var_name>.*)'.format(
            cls.SUBVAR_NAME_DELIMITER), t)
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
