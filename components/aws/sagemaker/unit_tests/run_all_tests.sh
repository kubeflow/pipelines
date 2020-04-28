export PYTHONPATH=../

coverage run -m pytest --junitxml ./log
coverage report --omit "*/usr/*,tests/*,*__init__*,*/Python/*"