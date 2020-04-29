import subprocess
from typing import List, Text

# TODO: move diagnose_me/utility.py to common package
from ..diagnose_me import utility

def execute_subprocess(command):
  return subprocess.run(command.split())

def execute(command):
  return utility.ExecutorResponse().execute_command(command.split())
