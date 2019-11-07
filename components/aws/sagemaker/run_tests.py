# Configures and runs the unit tests for all the components

import os
import sys
import unittest


# Taken from http://stackoverflow.com/a/17004263/2931197
def load_and_run_tests():
  setup_file = sys.modules['__main__'].__file__
  setup_dir = os.path.abspath(os.path.dirname(setup_file))

  test_loader = unittest.defaultTestLoader  
  test_runner = unittest.TextTestRunner()
  test_suite = test_loader.discover(setup_dir, pattern="test_*.py")

  test_runner.run(test_suite)

if __name__ == '__main__':
  load_and_run_tests()