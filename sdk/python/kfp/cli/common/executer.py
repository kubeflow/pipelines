from typing import List, Text

# TODO: move diagnose_me/utility.py to common package
from ..diagnose_me import utility

def execute_stdout(command_list: List[Text]):
  """Executes the command in command_list.
  """
  return utility.ExecutorResponse().execute_command(command_list).stdout

def execute(command_list: List[Text]):
  """Executes the command in command_list.
  """
  return utility.ExecutorResponse().execute_command(command_list)
