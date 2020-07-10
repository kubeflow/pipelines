export PYTHONPATH=../../

coverage run -m pytest --junitxml ./unit_tests.log
coverage report -m --omit "*/usr/*,tests/*,*__init__*,*/Python/*"