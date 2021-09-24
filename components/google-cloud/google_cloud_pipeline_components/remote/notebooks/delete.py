def coucou(*args, **kwargs):
  print(args)
  print(kwargs.get('project_id', ''))
  print(kwargs['input_notebook_file'])
  print(kwargs['labels'])


def main():
  import argparse
  _parser = argparse.ArgumentParser(prog='Execute notebook', description='Executes a notebook using the Notebooks Executor API.')
  _parser.add_argument("--project-id", dest="project_id", type=str, required=True, default=argparse.SUPPRESS)
  _parser.add_argument("--input-notebook-file", dest="input_notebook_file", type=str, required=True, default=argparse.SUPPRESS)
  _parser.add_argument("--labels", dest="labels", type=str, required=False, default='aa')
  _parsed_args = vars(_parser.parse_args())

  print(_parsed_args)

  args, unknown_args = _parser.parse_known_args()
  print(args)

  # coucou(**_parsed_args)

if __name__ == "__main__":
    main()