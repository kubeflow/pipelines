# sample-test

## Updating python dependencies

[pip-tools](https://github.com/jazzband/pip-tools) is used to manage python
dependencies. To update dependencies:
1. edit [requirements.in](requirements.in)
1. run
    ```
    cat ../../sdk/python/requirements.in requirements.in | \
        ../../backend/update_requirements.sh google/cloud-sdk:315.0.0 >requirements.txt
    ```
    to update and pin the transitive dependencies.
