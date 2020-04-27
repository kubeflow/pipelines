from termcolor import colored

def print_warning(data):
  print(colored(data, 'yellow'))

def print_error(data):
  print(colored(data, 'red'))

def print_success(data):
  print(colored(data, 'green'))
