"""A main that wraps the underlying main for evaluation backend."""
import sys
import google.protobuf

from lib.eval_backend_main import main

# Import after applying the module aliases.

if __name__ == '__main__':
  print('Running main with (%r)' % sys.argv)
  main(sys.argv)
