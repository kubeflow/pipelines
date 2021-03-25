from kfp.components.executor import Executor


def executor_main():
  import argparse
  import json

  parser = argparse.ArgumentParser(description='Process some integers.')
  parser.add_argument('--executor_input', type=str)
  parser.add_argument('--function_to_execute', type=str)

  args, _ = parser.parse_known_args()
  executor_input = json.loads(args.executor_input)
  function_to_execute = globals()[args.function_to_execute]

  executor = Executor(executor_input=executor_input,
                      function_to_execute=function_to_execute)

  executor.execute()
