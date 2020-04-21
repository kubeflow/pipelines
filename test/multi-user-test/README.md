# Multi-user test

## Use alternative test profiles
The user accounts used in [profile.yaml](./profile.yaml) file are Google owned test accounts.
In case you need to use other accounts, for instance, if you're running the test in your own project and need to acquire new refresh tokens,
you can edit the file locally, replace the profile names and email addresses with your own choices.

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies, edit [requirements.in](requirements.in)
and run `cat ../../sdk/python/requirements.in requirements.in | ../../backend/update_requirements.sh google/cloud-sdk:279.0.0 >requirements.txt` to
update and pin the transitive dependencies.
