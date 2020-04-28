import click

def print_warning(data):
  click.echo(click.style(data, fg='yellow'))

def print_error(data):
  click.echo(click.style(data, fg='red'))
  #print(colored(data, 'red'))

def print_success(data):
  click.echo(click.style(data, fg='green'))

def input_must_have(msg):
  data = ''
  while not data:
    data = input(msg)
  return data

def input_must_have_boolean(msg):
  data = ''
  while not data or data not in ('y', 'n'):
    data = input(msg)
  return data
