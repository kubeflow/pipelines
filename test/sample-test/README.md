# sample-test

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `../../backend/update_requirements.sh google/cloud-sdk:279.0.0` to
update and pin the transitive dependencies.
