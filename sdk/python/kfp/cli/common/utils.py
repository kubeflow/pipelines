import click
from ../.. import __VERSION__

def print_warning(data):
  click.echo(click.style(data, fg='yellow'))

def print_error(data):
  click.echo(click.style(data, fg='red'))
  #print(colored(data, 'red'))

def print_success(data):
  click.echo(click.style(data, fg='green'))

def write_line(file, data):
  file.write(data + '\n')

def current_version():
  return __VERSION__
