export PYTHONPATH=../../

coverage run -m pytest --ignore=tests/test_utils.py --junitxml ./unit_tests.log
coverage report --omit "*/usr/*,tests/*,*__init__*,*/Python/*"