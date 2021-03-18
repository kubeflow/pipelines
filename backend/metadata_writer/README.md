# metadata\_writer

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `../update_requirements.sh python:3.7 <requirements.in >requirements.txt` to update and pin the transitive
dependencies.

