import click

def print_warning(data):
  click.echo(click.style(data, fg='yellow'))

def print_error(data):
  click.echo(click.style(data, fg='red'))
  #print(colored(data, 'red'))

def print_success(data):
  click.echo(click.style(data, fg='green'))

def write_line(file, data):
  file.write(data + '\n')
